package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	tripsmodels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Caso real de prod: 1,20 USD × 871,3689 = 1.045,64 Bs; 500,64 de cartera y Pago Móvil de 1.000.
const (
	mixedFareUSD   = 1.20
	mixedRate      = 871.3689
	mixedBalanceBs = 500.64
	mixedPMBs      = 545.00
)

func TestE2E_HTTP_MixedTripCancelRefundsBalanceAndPagoMovil(t *testing.T) {
	customerID := newCustomer(t)
	seedWallet(t, customerID, mixedBalanceBs)
	tripID := createMixedTripHTTP(t, customerID)

	staffResolveTrip(t, tripID, map[string]any{
		"type":               "approve_overpay",
		"observed_amount_bs": 1000.0,
	})
	require.Equal(t, string(tripsmodels.TripStatusSearching), getTripHTTP(t, customerID, tripID).Status)

	cancelled := cancelTripHTTP(t, customerID, tripID)
	require.Equal(t, string(tripsmodels.TripStatusCancelled), cancelled.Status)
	require.NotNil(t, cancelled.Payment)
	require.InDelta(t, mixedPMBs, *cancelled.Payment.CreditedBs, 0.001)

	movements := movementsByType(t, tripID)
	require.InDelta(t, -mixedBalanceBs, movements["balance_payment"], 0.001)
	require.InDelta(t, 455.0, movements["payment_overage"], 0.001)
	require.InDelta(t, mixedBalanceBs, movements["balance_payment_refund"], 0.001)
	require.InDelta(t, mixedPMBs, movements["order_rejected_refund"], 0.001)
	require.InDelta(t, mixedBalanceBs+1000.0, walletBalance(t, customerID), 0.001)

	_, raw := customerJSON(t, customerID, http.MethodPost, "/trips/"+tripID.String()+"/cancel", nil)
	require.Len(t, ledgerForReference(t, tripID), 4, string(raw))
}

func TestE2E_HTTP_MixedTripRejectedProofRefundsBalance(t *testing.T) {
	customerID := newCustomer(t)
	seedWallet(t, customerID, mixedBalanceBs)
	tripID := createMixedTripHTTP(t, customerID)

	staffResolveTrip(t, tripID, map[string]any{
		"type":   "reject_invalid",
		"reason": "la foto no es un comprobante",
	})

	require.Equal(t, string(tripsmodels.TripStatusCancelled), getTripHTTP(t, customerID, tripID).Status)
	movements := movementsByType(t, tripID)
	require.InDelta(t, -mixedBalanceBs, movements["balance_payment"], 0.001)
	require.InDelta(t, mixedBalanceBs, movements["balance_payment_refund"], 0.001)
	require.InDelta(t, mixedBalanceBs, walletBalance(t, customerID), 0.001)
}

func TestE2E_HTTP_CashTripCancelDoesNotCredit(t *testing.T) {
	customerID := newCustomer(t)
	quoteID := quoteStore.put(customerID, 5, bcvRate)
	status, raw := customerJSON(t, customerID, http.MethodPost, "/trips", map[string]any{
		"quote_id":       quoteID,
		"method_payment": "Efectivo",
	})
	require.Equal(t, http.StatusCreated, status, string(raw))
	var trip tripHTTP
	require.NoError(t, json.Unmarshal(raw, &trip))
	require.Equal(t, string(tripsmodels.TripStatusSearching), trip.Status)

	cancelled := cancelTripHTTP(t, customerID, trip.ID)
	require.Equal(t, string(tripsmodels.TripStatusCancelled), cancelled.Status)
	require.Nil(t, cancelled.Payment)
	require.Empty(t, ledgerForReference(t, trip.ID))
}

func createMixedTripHTTP(t *testing.T, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	quoteID := quoteStore.put(customerID, mixedFareUSD, mixedRate)
	status, raw := customerJSON(t, customerID, http.MethodPost, "/trips", map[string]any{
		"quote_id":            quoteID,
		"method_payment":      "PagoMovil",
		"balance_amount_bs":   mixedBalanceBs,
		"payment_proof_image": "proof",
	})
	require.Equal(t, http.StatusCreated, status, string(raw))
	var trip tripHTTP
	require.NoError(t, json.Unmarshal(raw, &trip))
	require.Equal(t, string(tripsmodels.TripStatusPaymentReview), trip.Status)
	return trip.ID
}

func seedWallet(t *testing.T, customerID uuid.UUID, amountBs float64) {
	t.Helper()
	_, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", amountBs, mixedRate)
	require.NoError(t, err)
}

func movementsByType(t *testing.T, referenceID uuid.UUID) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	for _, row := range ledgerForReference(t, referenceID) {
		out[row.MovementType] += row.Amount
	}
	return out
}

func walletBalance(t *testing.T, customerID uuid.UUID) float64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var amount float64
	err := sqlDB.QueryRowContext(ctx, `
		SELECT amount FROM finance.balances WHERE entity_id = $1 AND entity_type = 'customer'
	`, customerID).Scan(&amount)
	require.NoError(t, err)
	return amount
}
