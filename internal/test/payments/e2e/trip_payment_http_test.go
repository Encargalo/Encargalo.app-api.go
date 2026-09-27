package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsmodels "Encargalo.app-api.go/internal/trips/domain/models"
	usersdto "Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestE2E_HTTP_UnderpayCreditsObservedAndCancelsTrip(t *testing.T) {
	customerID := newCustomer(t)
	tripID := createPagoMovilTripHTTP(t, customerID)

	staffResolveTrip(t, tripID, map[string]any{
		"type":               "reject_underpaid",
		"observed_amount_bs": 150.0,
	})

	trip := getTripHTTP(t, customerID, tripID)
	require.Equal(t, string(tripsmodels.TripStatusCancelled), trip.Status)
	require.Nil(t, trip.Payment)

	rows := ledgerForReference(t, tripID)
	require.Len(t, rows, 1)
	require.Equal(t, "payment_partial_credit", rows[0].MovementType)
	require.InDelta(t, 150.0, rows[0].Amount, 0.001)
	require.Equal(t, "trip", *rows[0].ReferenceType)

	tx := balanceMovementHTTP(t, customerID, "payment_partial_credit")
	require.InDelta(t, 150.0, tx.AmountBs, 0.001)
	require.NotNil(t, tx.TripID)
	require.Equal(t, tripID.String(), *tx.TripID)
	require.Nil(t, tx.OrderID)
	require.True(t, tx.CanRequestRefund)
	require.NotNil(t, tx.RefundEligibleUntil)
	require.NotNil(t, tx.RefundablePMBs)
	require.InDelta(t, 150.0, *tx.RefundablePMBs, 0.001)
}

func TestE2E_HTTP_OverpayCreditsExcessAndStartsSearch(t *testing.T) {
	customerID := newCustomer(t)
	tripID := createPagoMovilTripHTTP(t, customerID)

	staffResolveTrip(t, tripID, map[string]any{
		"type":               "approve_overpay",
		"observed_amount_bs": 250.0,
	})

	trip := getTripHTTP(t, customerID, tripID)
	require.Equal(t, string(tripsmodels.TripStatusSearching), trip.Status)
	require.Nil(t, trip.Payment)

	rows := ledgerForReference(t, tripID)
	require.Len(t, rows, 1)
	require.Equal(t, "payment_overage", rows[0].MovementType)
	require.InDelta(t, 50.0, rows[0].Amount, 0.001)
	require.Equal(t, "trip", *rows[0].ReferenceType)

	tx := balanceMovementHTTP(t, customerID, "payment_overage")
	require.InDelta(t, 50.0, tx.AmountBs, 0.001)
	require.Equal(t, tripID.String(), *tx.TripID)
	require.True(t, tx.CanRequestRefund)
	require.NotNil(t, tx.RefundEligibleUntil)
}

func TestE2E_HTTP_ApproveThenCancelCreditsFare(t *testing.T) {
	customerID := newCustomer(t)
	tripID := createPagoMovilTripHTTP(t, customerID)

	staffResolveTrip(t, tripID, map[string]any{"type": "approve"})

	searching := getTripHTTP(t, customerID, tripID)
	require.Equal(t, string(tripsmodels.TripStatusSearching), searching.Status)
	require.Empty(t, ledgerForReference(t, tripID))

	cancelled := cancelTripHTTP(t, customerID, tripID)
	require.Equal(t, string(tripsmodels.TripStatusCancelled), cancelled.Status)
	require.NotNil(t, cancelled.Payment)
	require.NotNil(t, cancelled.Payment.TransactionID)
	require.InDelta(t, 200.0, *cancelled.Payment.CreditedBs, 0.001)
	require.True(t, cancelled.Payment.CanRequestRefund)

	rows := ledgerForReference(t, tripID)
	require.Len(t, rows, 1)
	require.Equal(t, "order_rejected_refund", rows[0].MovementType)
	require.InDelta(t, 200.0, rows[0].Amount, 0.001)
	require.Equal(t, "trip", *rows[0].ReferenceType)
	require.Equal(t, rows[0].ID.String(), cancelled.Payment.TransactionID.String())

	tx := balanceMovementHTTP(t, customerID, "order_rejected_refund")
	require.InDelta(t, 200.0, tx.AmountBs, 0.001)
	require.Equal(t, tripID.String(), *tx.TripID)
	require.True(t, tx.CanRequestRefund)

	registerPagoMovilHTTP(t, customerID)
	refund := refundHTTP(t, customerID, cancelled.Payment.TransactionID.String())
	require.NotNil(t, refund.TripID)
	require.Equal(t, tripID.String(), *refund.TripID)
	require.Nil(t, refund.OrderID)

	var refID uuid.UUID
	var refType string
	err := sqlDB.QueryRowContext(context.Background(), `
		SELECT reference_id, reference_type
		FROM finance.balance_refund_requests
		WHERE transaction_id = $1
	`, *cancelled.Payment.TransactionID).Scan(&refID, &refType)
	require.NoError(t, err)
	require.Equal(t, tripID, refID)
	require.Equal(t, "trip", refType)
}

func createPagoMovilTripHTTP(t *testing.T, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	quoteID := quoteStore.put(customerID, 5, bcvRate)
	status, raw := customerJSON(t, customerID, http.MethodPost, "/trips", map[string]any{
		"quote_id":            quoteID,
		"method_payment":      "PagoMovil",
		"payment_proof_image": "proof",
	})
	require.Equal(t, http.StatusCreated, status)
	var trip tripHTTP
	require.NoError(t, json.Unmarshal(raw, &trip))
	require.Equal(t, string(tripsmodels.TripStatusPaymentReview), trip.Status)
	require.NotNil(t, trip.FareUSD)
	require.InDelta(t, 5, *trip.FareUSD, 0.001)
	return trip.ID
}

func getTripHTTP(t *testing.T, customerID, tripID uuid.UUID) tripHTTP {
	t.Helper()
	status, raw := customerJSON(t, customerID, http.MethodGet, "/trips/"+tripID.String(), nil)
	require.Equal(t, http.StatusOK, status)
	var trip tripHTTP
	require.NoError(t, json.Unmarshal(raw, &trip))
	return trip
}

func cancelTripHTTP(t *testing.T, customerID, tripID uuid.UUID) tripHTTP {
	t.Helper()
	status, raw := customerJSON(t, customerID, http.MethodPost, "/trips/"+tripID.String()+"/cancel", nil)
	require.Equal(t, http.StatusOK, status, string(raw))
	var trip tripHTTP
	require.NoError(t, json.Unmarshal(raw, &trip))
	return trip
}

func balanceMovementHTTP(t *testing.T, customerID uuid.UUID, movement string) usersdto.CustomerTransactionItem {
	t.Helper()
	status, raw := customerJSON(t, customerID, http.MethodGet, "/users/balance", nil)
	require.Equal(t, http.StatusOK, status)
	var balance usersdto.CustomerBalanceResponse
	require.NoError(t, json.Unmarshal(raw, &balance))
	for _, tx := range balance.Transactions {
		if tx.MovementType == movement {
			return tx
		}
	}
	t.Fatalf("no está %s en el balance: %#v", movement, balance.Transactions)
	return usersdto.CustomerTransactionItem{}
}

func registerPagoMovilHTTP(t *testing.T, customerID uuid.UUID) {
	t.Helper()
	status, raw := customerJSON(t, customerID, http.MethodPut, "/users/pago-movil", map[string]any{
		"phone":       nextPhone(),
		"bank_code":   "0102",
		"national_id": "V" + nextPhone()[4:],
	})
	require.Equal(t, http.StatusOK, status, string(raw))
}

func refundHTTP(t *testing.T, customerID uuid.UUID, transactionID string) usersdto.RefundRequestResponse {
	t.Helper()
	status, raw := customerJSON(t, customerID, http.MethodPost, "/users/balance/refund-requests", map[string]any{
		"transaction_id": transactionID,
	})
	require.Equal(t, http.StatusCreated, status, string(raw))
	var refund usersdto.RefundRequestResponse
	require.NoError(t, json.Unmarshal(raw, &refund))
	return refund
}

func staffResolveTrip(t *testing.T, tripID uuid.UUID, action map[string]any) {
	t.Helper()
	conn := dialStaff(t)
	snapshot := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return findTripReview(msgs, tripID) != nil
	})
	review := findTripReview(snapshot, tripID)
	require.Equal(t, "pending_review", review["status"])
	require.Equal(t, "trip", review["reference_type"])

	action["review_id"] = review["review_id"]
	action["reference_last4"] = freshLast4(t)
	sendStaff(t, conn, action)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		ok := findMessage(msgs, "action_ok", uuid.MustParse(review["review_id"].(string)))
		return ok != nil && ok["action"] == action["type"]
	})
}

func findTripReview(msgs []map[string]any, tripID uuid.UUID) map[string]any {
	for _, msg := range msgs {
		if msg["type"] == "payment_review_update" && msg["reference_id"] == tripID.String() && msg["reference_type"] == "trip" {
			return msg
		}
	}
	return nil
}

func customerJSON(t *testing.T, customerID uuid.UUID, method, path string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, httpServer.URL+path, reader)
	require.NoError(t, err)
	req.Header.Set(testCustomerHeader, customerID.String())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, raw
}

type tripHTTP struct {
	ID      uuid.UUID    `json:"id"`
	Status  string       `json:"status"`
	FareUSD *float64     `json:"fare_usd"`
	Payment *tripPayHTTP `json:"payment"`
}

type tripPayHTTP struct {
	TransactionID    *uuid.UUID `json:"transaction_id"`
	CreditedBs       *float64   `json:"credited_bs"`
	CanRequestRefund bool       `json:"can_request_refund"`
}

type memQuotes struct {
	mu    sync.Mutex
	items map[string]*tripsmodels.TripQuote
}

func newMemQuotes() *memQuotes {
	return &memQuotes{items: map[string]*tripsmodels.TripQuote{}}
}

func (m *memQuotes) put(customerID uuid.UUID, fareUSD, rate float64) uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	now := time.Now()
	quote := &tripsmodels.TripQuote{
		QuoteID:          id,
		CustomerID:       customerID,
		PickupAddress:    "Origen",
		PickupLatitude:   10.5,
		PickupLongitude:  -66.9,
		DropoffAddress:   "Destino",
		DropoffLatitude:  10.51,
		DropoffLongitude: -66.91,
		FareUSD:          fareUSD,
		BcvRate:          rate,
		CreatedAt:        now,
		ExpiresAt:        now.Add(tripsmodels.TripQuoteTTL),
	}
	m.items[customerID.String()+id.String()] = quote
	return id
}

func (m *memQuotes) Save(_ context.Context, quote *tripsmodels.TripQuote, _ time.Duration) error {
	if quote == nil {
		return errcustom.ErrUnexpectedError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *quote
	m.items[quote.CustomerID.String()+quote.QuoteID.String()] = &cp
	return nil
}

func (m *memQuotes) Get(_ context.Context, customerID, quoteID uuid.UUID) (*tripsmodels.TripQuote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	quote, ok := m.items[customerID.String()+quoteID.String()]
	if !ok || quote.CustomerID != customerID {
		return nil, errcustom.ErrQuoteNotFound
	}
	cp := *quote
	return &cp, nil
}

func (m *memQuotes) GetAndConsume(ctx context.Context, customerID, quoteID uuid.UUID) (*tripsmodels.TripQuote, error) {
	quote, err := m.Get(ctx, customerID, quoteID)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	delete(m.items, customerID.String()+quoteID.String())
	m.mu.Unlock()
	return quote, nil
}

type fakeUpload struct{}

func (fakeUpload) UploadImage(context.Context, string) (*uploader.UploadResult, error) {
	return &uploader.UploadResult{SecureURL: proofURL}, nil
}

func (fakeUpload) DeleteImage(context.Context, string) error { return nil }
