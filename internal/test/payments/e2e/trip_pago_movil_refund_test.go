package e2e_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsapp "Encargalo.app-api.go/internal/trips/app"
	tripsmodels "Encargalo.app-api.go/internal/trips/domain/models"
	tripsports "Encargalo.app-api.go/internal/trips/domain/ports"
	tripsrepo "Encargalo.app-api.go/internal/trips/repo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestE2E_CancelVerifiedPagoMovilCreditsWallet(t *testing.T) {
	customerID := newCustomer(t)
	trip := insertPagoMovilTrip(t, customerID, tripsmodels.TripStatusSearching, nil)
	svc := newTripsSvc()

	resp, err := svc.CancelTrip(context.Background(), trip.ID, customerID)
	require.NoError(t, err)
	require.Equal(t, string(tripsmodels.TripStatusCancelled), resp.Status)
	require.NotNil(t, resp.Payment)
	require.NotNil(t, resp.Payment.TransactionID)
	require.InDelta(t, 200.0, *resp.Payment.CreditedBs, 0.001)
	require.NotNil(t, resp.Payment.RefundablePMBs)
	require.NotNil(t, resp.Payment.RefundEligibleUntil)

	assertPagoMovilRefundInDB(t, customerID, trip.ID, *resp.Payment.TransactionID)

	_, err = svc.CancelTrip(context.Background(), trip.ID, customerID)
	require.ErrorIs(t, err, errcustom.ErrTripNotCancellable)
	require.Len(t, ledgerForReference(t, trip.ID), 1)
}

func TestE2E_NoDriverCreditsPagoMovilOnce(t *testing.T) {
	customerID := newCustomer(t)
	trip := insertPagoMovilTrip(t, customerID, tripsmodels.TripStatusSearching, nil)
	repo := tripsrepo.NewTripsRepo(db)

	ok, err := repo.UpdateTripStatus(context.Background(), trip.ID, tripsmodels.TripStatusSearching, tripsmodels.TripStatusNoDriver)
	require.NoError(t, err)
	require.True(t, ok)

	loaded, err := repo.GetTripByID(context.Background(), trip.ID)
	require.NoError(t, err)
	tripsapp.CreditUnservedPagoMovil(context.Background(), reviewsApp, loaded, repo.SaveTripPayment)
	tripsapp.CreditUnservedPagoMovil(context.Background(), reviewsApp, loaded, repo.SaveTripPayment)

	rows := ledgerForReference(t, trip.ID)
	require.Len(t, rows, 1)
	require.Equal(t, "order_rejected_refund", rows[0].MovementType)
	require.InDelta(t, 200.0, rows[0].Amount, 0.001)

	var txID string
	err = sqlDB.QueryRowContext(context.Background(), `SELECT payment->>'transaction_id' FROM trips.trips WHERE id = $1`, trip.ID).Scan(&txID)
	require.NoError(t, err)
	require.Equal(t, rows[0].ID.String(), txID)
}

func TestE2E_CancelDuringPaymentReviewDoesNotCredit(t *testing.T) {
	customerID := newCustomer(t)
	trip := insertPagoMovilTrip(t, customerID, tripsmodels.TripStatusPaymentReview, nil)

	_, err := newTripsSvc().CancelTrip(context.Background(), trip.ID, customerID)
	require.ErrorIs(t, err, errcustom.ErrTripPaymentReviewPending)
	require.Empty(t, ledgerForReference(t, trip.ID))

	var status string
	err = sqlDB.QueryRowContext(context.Background(), `SELECT status FROM trips.trips WHERE id = $1`, trip.ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, string(tripsmodels.TripStatusPaymentReview), status)

	var payment *string
	err = sqlDB.QueryRowContext(context.Background(), `SELECT payment::text FROM trips.trips WHERE id = $1`, trip.ID).Scan(&payment)
	require.NoError(t, err)
	require.Nil(t, payment)

	var balance float64
	err = sqlDB.QueryRowContext(context.Background(), `
		SELECT COALESCE(amount, 0) FROM finance.balances WHERE entity_id = $1 AND entity_type = 'customer'
	`, customerID).Scan(&balance)
	require.Error(t, err)
}

func TestE2E_CancelBalanceTripRefundsDebitNotPagoMovil(t *testing.T) {
	customerID := newCustomer(t)
	paid := 200.0
	trip := insertPagoMovilTrip(t, customerID, tripsmodels.TripStatusAccepted, &paid)

	_, err := newTripsSvc().CancelTrip(context.Background(), trip.ID, customerID)
	require.NoError(t, err)

	rows := ledgerForReference(t, trip.ID)
	require.Len(t, rows, 1)
	require.Equal(t, "balance_payment_refund", rows[0].MovementType)

	var payment *string
	err = sqlDB.QueryRowContext(context.Background(), `SELECT payment::text FROM trips.trips WHERE id = $1`, trip.ID).Scan(&payment)
	require.NoError(t, err)
	require.Nil(t, payment)
}

func assertPagoMovilRefundInDB(t *testing.T, customerID, tripID, txID uuid.UUID) {
	t.Helper()
	rows := ledgerForReference(t, tripID)
	require.Len(t, rows, 1)
	require.Equal(t, txID, rows[0].ID)
	require.Equal(t, "order_rejected_refund", rows[0].MovementType)
	require.Equal(t, "trip", *rows[0].ReferenceType)
	require.Equal(t, customerID, rows[0].EntityID)
	require.InDelta(t, 200.0, rows[0].Amount, 0.001)

	var refundable float64
	err := sqlDB.QueryRowContext(context.Background(), `
		SELECT refundable_pm_bs FROM finance.transactions WHERE id = $1
	`, txID).Scan(&refundable)
	require.NoError(t, err)
	require.InDelta(t, 200.0, refundable, 0.001)

	var storedTx string
	var credited float64
	err = sqlDB.QueryRowContext(context.Background(), `
		SELECT payment->>'transaction_id', (payment->>'credited_bs')::float8
		FROM trips.trips WHERE id = $1 AND status = 'cancelled'
	`, tripID).Scan(&storedTx, &credited)
	require.NoError(t, err)
	require.Equal(t, txID.String(), storedTx)
	require.InDelta(t, 200.0, credited, 0.001)

	var balanceAmount float64
	err = sqlDB.QueryRowContext(context.Background(), `
		SELECT amount FROM finance.balances WHERE entity_id = $1 AND entity_type = 'customer'
	`, customerID).Scan(&balanceAmount)
	require.NoError(t, err)
	require.InDelta(t, 200.0, balanceAmount, 0.001)

	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.NotNil(t, balance.Transactions[0].TripID)
	require.Equal(t, tripID.String(), *balance.Transactions[0].TripID)
	require.Nil(t, balance.Transactions[0].OrderID)
}

func insertPagoMovilTrip(t *testing.T, customerID uuid.UUID, status tripsmodels.TripStatus, balancePaid *float64) *tripsmodels.Trip {
	t.Helper()
	fare := 5.0
	rate := bcvRate
	trip := &tripsmodels.Trip{
		ID:               uuid.New(),
		CustomerID:       customerID,
		Status:           status,
		PickupAddress:    "Origen",
		PickupLatitude:   10.5,
		PickupLongitude:  -66.9,
		DropoffAddress:   "Destino",
		DropoffLatitude:  10.51,
		DropoffLongitude: -66.91,
		Fare:             &fare,
		BcvRate:          &rate,
		BalancePaidBs:    balancePaid,
	}
	_, err := db.NewInsert().Model(trip).Exec(context.Background())
	require.NoError(t, err)
	return trip
}

func newTripsSvc() tripsports.TripsApp {
	repo := tripsrepo.NewTripsRepo(db)
	return tripsapp.NewTripsApp(repo, nil, fixedBCV{}, nil, nil, nil, nil, nil, reviewsApp, nil)
}
