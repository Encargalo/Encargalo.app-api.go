package e2e_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	ordersapp "Encargalo.app-api.go/internal/orders/app"
	ordersmodels "Encargalo.app-api.go/internal/orders/domain/models"
	paymentsmodels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	usersdto "Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStaff_SnapshotAndExactApprove(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)

	conn := dialStaff(t)
	snapshot := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "payment_review_update", seed.Review.ID) && hasMessage(msgs, "connected", uuid.Nil)
	})
	item := findMessage(snapshot, "payment_review_update", seed.Review.ID)
	require.Equal(t, seed.Order.ID.String(), item["reference_id"])
	require.Equal(t, "order", item["reference_type"])
	require.Equal(t, shopID.String(), item["shop_id"])
	require.Equal(t, customerID.String(), item["customer_id"])
	require.Equal(t, "Ana Perez", item["customer_name"])
	require.Equal(t, "pending_review", item["status"])
	require.Equal(t, string(ordersmodels.StatusPaymentReview), item["order_status"])
	_, hasOrderID := item["order_id"]
	require.False(t, hasOrderID)

	last4 := freshLast4(t)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": last4,
	})
	done := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "action_ok", seed.Review.ID) && hasMessage(msgs, "payment_review_update", seed.Review.ID)
	})
	ok := findMessage(done, "action_ok", seed.Review.ID)
	require.Equal(t, "approve", ok["action"])
	require.Equal(t, seed.Order.ID.String(), ok["reference_id"])
	require.Equal(t, "order", ok["reference_type"])
	_, hasOrderID = ok["order_id"]
	require.False(t, hasOrderID)
	_, hasOrderStatus := ok["order_status"]
	require.False(t, hasOrderStatus)

	broadcast := findMessage(done, "payment_review_update", seed.Review.ID)
	require.Equal(t, "approved", broadcast["status"])
	require.Equal(t, string(ordersmodels.StatusPending), broadcast["order_status"])

	order := reloadOrder(t, seed.Order.ID)
	require.Equal(t, string(ordersmodels.StatusPending), order.Status)
	require.NotNil(t, order.PendingAt)
	pd := paymentDetails(t, order)
	require.Equal(t, paymentsmodels.PaymentReviewStatus("approved"), pd.Status)
	require.Nil(t, pd.TransactionID)

	review := reloadReview(t, seed.Review.ID)
	require.Equal(t, paymentsmodels.ReviewApproved, review.Status)
	require.NotNil(t, review.ReferenceLast4)
	require.Equal(t, last4, strings.TrimSpace(*review.ReferenceLast4))
	require.Empty(t, ledgerForReference(t, seed.Order.ID))
}

func TestStaff_ApproveOverpayCreditsBalanceAndKeepsOrderIDOnCustomerDTO(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)

	conn := dialStaff(t)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
	observed := 500.0
	sendStaff(t, conn, map[string]any{
		"type":               "approve_overpay",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": observed,
		"reference_last4":    freshLast4(t),
	})
	done := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "action_ok", seed.Review.ID)
	})
	ok := findMessage(done, "action_ok", seed.Review.ID)
	require.Equal(t, "approve_overpay", ok["action"])

	order := reloadOrder(t, seed.Order.ID)
	require.Equal(t, string(ordersmodels.StatusPending), order.Status)
	pd := paymentDetails(t, order)
	require.Equal(t, paymentsmodels.ReviewOverpaid, pd.Status)
	require.NotNil(t, pd.TransactionID)
	require.NotNil(t, pd.CreditedBs)
	require.InDelta(t, 100, *pd.CreditedBs, 0.001)
	require.True(t, pd.CanRequestRefund)
	require.NotNil(t, pd.RefundEligibleUntil)

	rows := ledgerForReference(t, seed.Order.ID)
	require.Len(t, rows, 1)
	require.Equal(t, "payment_overage", rows[0].MovementType)
	require.InDelta(t, 100, rows[0].Amount, 0.001)
	require.NotNil(t, rows[0].ReferenceType)
	require.Equal(t, "order", *rows[0].ReferenceType)
	require.Nil(t, rows[0].OrderID)

	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.InDelta(t, 100, balance.BalanceBs, 0.001)
	var overage *string
	for i := range balance.Transactions {
		if balance.Transactions[i].MovementType == "payment_overage" {
			overage = balance.Transactions[i].OrderID
			require.True(t, balance.Transactions[i].CanRequestRefund)
		}
	}
	require.NotNil(t, overage)
	require.Equal(t, seed.Order.ID.String(), *overage)

	_, err = reviewsApp.UpsertPagoMovil(context.Background(), customerID, pagoMovil())
	require.NoError(t, err)
	refund, err := reviewsApp.CreateRefundRequest(context.Background(), customerID, *pd.TransactionID)
	require.NoError(t, err)
	require.NotNil(t, refund.OrderID)
	require.Equal(t, seed.Order.ID.String(), *refund.OrderID)

	var refID uuid.UUID
	var refType string
	err = sqlDB.QueryRowContext(context.Background(), `
		SELECT reference_id, reference_type
		FROM finance.balance_refund_requests
		WHERE transaction_id = $1
	`, *pd.TransactionID).Scan(&refID, &refType)
	require.NoError(t, err)
	require.Equal(t, seed.Order.ID, refID)
	require.Equal(t, "order", refType)
}

func TestStaff_RejectUnderpaidRefundsBalanceUsed(t *testing.T) {
	customerID := newCustomer(t)
	// 80 Bs de balance para poder debitarlos al crear la orden.
	_, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", 80, bcvRate)
	require.NoError(t, err)
	seed := seedPendingOrder(t, customerID, 80, 320)
	require.NoError(t, reviewsApp.DebitForReference(context.Background(), customerID, seed.Order.ID, "order", 80))

	debits := ledgerForReference(t, seed.Order.ID)
	require.Len(t, debits, 1)
	require.Equal(t, "balance_payment", debits[0].MovementType)
	require.InDelta(t, -80, debits[0].Amount, 0.001)
	require.Equal(t, "order", *debits[0].ReferenceType)
	require.Nil(t, debits[0].OrderID)

	conn := dialStaff(t)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
	sendStaff(t, conn, map[string]any{
		"type":               "reject_underpaid",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": 200.0,
		"reference_last4":    freshLast4(t),
	})
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		ok := findMessage(msgs, "action_ok", seed.Review.ID)
		return ok != nil && ok["action"] == "reject_underpaid"
	})

	order := reloadOrder(t, seed.Order.ID)
	require.Equal(t, string(ordersmodels.StatusRejected), order.Status)
	require.Nil(t, order.PendingAt)
	pd := paymentDetails(t, order)
	require.Equal(t, paymentsmodels.ReviewRejectedUnderpaid, pd.Status)
	require.NotNil(t, pd.TransactionID)
	require.NotNil(t, pd.CreditedBs)
	require.InDelta(t, 280, *pd.CreditedBs, 0.001)

	review := reloadReview(t, seed.Review.ID)
	require.NotNil(t, review.CreditedBs)
	require.InDelta(t, 280, *review.CreditedBs, 0.001)

	rows := ledgerForReference(t, seed.Order.ID)
	var partial, refund float64
	for _, row := range rows {
		require.Equal(t, "order", *row.ReferenceType)
		require.Nil(t, row.OrderID)
		switch row.MovementType {
		case "payment_partial_credit":
			partial = row.Amount
		case "balance_payment_refund":
			refund = row.Amount
		}
	}
	require.InDelta(t, 200, partial, 0.001)
	require.InDelta(t, 80, refund, 0.001)

	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.InDelta(t, 280, balance.BalanceBs, 0.001)
}

func TestStaff_RejectInvalidRefundsBalanceAndDoesNotStoreLast4(t *testing.T) {
	customerID := newCustomer(t)
	_, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", 80, bcvRate)
	require.NoError(t, err)
	seed := seedPendingOrder(t, customerID, 80, 320)
	require.NoError(t, reviewsApp.DebitForReference(context.Background(), customerID, seed.Order.ID, "order", 80))

	conn := dialStaff(t)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
	sendStaff(t, conn, map[string]any{
		"type":      "reject_invalid",
		"review_id": seed.Review.ID,
		"reason":    "la foto no es un comprobante",
	})
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		ok := findMessage(msgs, "action_ok", seed.Review.ID)
		return ok != nil && ok["action"] == "reject_invalid"
	})

	order := reloadOrder(t, seed.Order.ID)
	require.Equal(t, string(ordersmodels.StatusRejected), order.Status)
	pd := paymentDetails(t, order)
	require.Equal(t, paymentsmodels.ReviewRejectedInvalid, pd.Status)
	require.Nil(t, pd.TransactionID)
	require.NotNil(t, pd.CreditedBs)
	require.InDelta(t, 80, *pd.CreditedBs, 0.001)

	review := reloadReview(t, seed.Review.ID)
	require.Equal(t, paymentsmodels.ReviewRejectedInvalid, review.Status)
	require.Nil(t, review.ReferenceLast4)
	require.Nil(t, review.CreditedBs)

	var refunded bool
	for _, row := range ledgerForReference(t, seed.Order.ID) {
		if row.MovementType == "balance_payment_refund" {
			refunded = true
			require.InDelta(t, 80, row.Amount, 0.001)
		}
		require.NotEqual(t, "payment_partial_credit", row.MovementType)
		require.NotEqual(t, "payment_overage", row.MovementType)
	}
	require.True(t, refunded)
}

func TestStaff_DuplicateLast4IsGlobalAndDoesNotConsumeTheReference(t *testing.T) {
	tripID := uuid.New()
	resolved, err := reviewsPort.CreateReview(context.Background(), "trip", tripID, proofURL, 100, bcvRate)
	require.NoError(t, err)
	last4 := freshLast4(t)
	ok, err := reviewsRepo.ResolvePending(context.Background(), resolved.ID, &paymentsmodels.PaymentReview{
		Status:         paymentsmodels.ReviewApproved,
		StaffID:        &staffID,
		ReferenceLast4: &last4,
	})
	require.NoError(t, err)
	require.True(t, ok)

	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	conn := dialStaff(t)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": last4,
	})
	done := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "action_ok", seed.Review.ID)
	})
	action := findMessage(done, "action_ok", seed.Review.ID)
	require.Equal(t, "reject_invalid", action["action"])
	require.Equal(t, true, action["duplicate_reject"])
	require.Contains(t, action["rejection_reason"], last4)

	order := reloadOrder(t, seed.Order.ID)
	require.Equal(t, string(ordersmodels.StatusRejected), order.Status)
	review := reloadReview(t, seed.Review.ID)
	require.Equal(t, paymentsmodels.ReviewRejectedInvalid, review.Status)
	require.Nil(t, review.ReferenceLast4)
	require.Empty(t, ledgerForReference(t, seed.Order.ID))
}

func TestStaff_UniqueIndexRejectsASecondConsumedLast4(t *testing.T) {
	firstCustomer := newCustomer(t)
	first := seedPendingOrder(t, firstCustomer, 0, 400)
	conn := dialStaff(t)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
	consumed := freshLast4(t)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       first.Review.ID,
		"reference_last4": consumed,
	})
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "action_ok", first.Review.ID)
	})

	secondCustomer := newCustomer(t)
	second := seedPendingOrder(t, secondCustomer, 0, 400)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       second.Review.ID,
		"reference_last4": freshLast4(t),
	})
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "action_ok", second.Review.ID)
	})

	_, err := sqlDB.ExecContext(context.Background(), `
		UPDATE payments.reviews SET reference_last4 = $2 WHERE id = $1
	`, second.Review.ID, consumed)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "duplicate")
}

func TestStaff_UnknownReferenceTypeDoesNotConsumeTheReview(t *testing.T) {
	review, err := reviewsPort.CreateReview(context.Background(), "legacy", uuid.New(), proofURL, 100, bcvRate)
	require.NoError(t, err)

	conn := dialStaff(t)
	msgs := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
	require.False(t, hasMessage(msgs, "payment_review_update", review.ID))

	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       review.ID,
		"reference_last4": freshLast4(t),
	})
	done := readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "error", uuid.Nil)
	})
	require.Equal(t, "error", findMessage(done, "error", uuid.Nil)["type"])

	still := reloadReview(t, review.ID)
	require.Equal(t, paymentsmodels.ReviewPendingReview, still.Status)
	require.Nil(t, still.ReferenceLast4)
}

func TestStaff_InvalidPayloadDoesNotResolve(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	conn := dialStaff(t)
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})

	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": "12",
	})
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "error", uuid.Nil)
	})
	require.Equal(t, paymentsmodels.ReviewPendingReview, reloadReview(t, seed.Review.ID).Status)

	sendStaff(t, conn, map[string]any{
		"type":      "reject_invalid",
		"review_id": seed.Review.ID,
	})
	readUntil(t, conn, 5*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "error", uuid.Nil)
	})
	require.Equal(t, paymentsmodels.ReviewPendingReview, reloadReview(t, seed.Review.ID).Status)
	require.Equal(t, string(ordersmodels.StatusPaymentReview), reloadOrder(t, seed.Order.ID).Status)
}

func TestCreateReview_RejectsIncompleteInput(t *testing.T) {
	_, err := reviewsPort.CreateReview(context.Background(), "", uuid.New(), proofURL, 10, bcvRate)
	require.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestAlertWorker_OnlyRepublishesOrderProofs(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	trip, err := reviewsPort.CreateReview(context.Background(), "trip", uuid.New(), proofURL, 50, bcvRate)
	require.NoError(t, err)

	past := time.Now().Add(-4 * time.Minute)
	_, err = sqlDB.ExecContext(context.Background(), `
		UPDATE payments.reviews
		SET created_at = $1, last_alerted_at = $1
		WHERE id IN ($2, $3)
	`, past, seed.Review.ID, trip.ID)
	require.NoError(t, err)

	stream := &recordStream{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ordersapp.NewPaymentProofAlertWorker(reviewsRepo, ordersRepo, stream).Start(ctx)

	require.Eventually(t, func() bool {
		stream.mu.Lock()
		defer stream.mu.Unlock()
		for _, event := range stream.events {
			if ev, ok := event.(ordersmodels.DataEventPaymentProofPending); ok && ev.OrderID == seed.Order.ID.String() {
				return ev.MessageType == "payment_proof_pending" && ev.ReviewID == seed.Review.ID.String()
			}
		}
		return false
	}, 3*time.Second, 50*time.Millisecond)

	stream.mu.Lock()
	defer stream.mu.Unlock()
	var sawSeed bool
	for _, event := range stream.events {
		ev, ok := event.(ordersmodels.DataEventPaymentProofPending)
		require.True(t, ok)
		require.NotEqual(t, trip.ID.String(), ev.ReviewID)
		require.NotEmpty(t, ev.OrderID)
		if ev.OrderID == seed.Order.ID.String() {
			sawSeed = true
			require.Equal(t, seed.Review.ID.String(), ev.ReviewID)
		}
	}
	require.True(t, sawSeed)
}

func pagoMovil() usersdto.UpsertPagoMovilRequest {
	return usersdto.UpsertPagoMovilRequest{
		Phone:      "04141234567",
		BankCode:   "0102",
		NationalID: "V12345678",
	}
}

type recordStream struct {
	mu     sync.Mutex
	events []any
}

func (r *recordStream) PublisherEvent(_ context.Context, data any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, data)
	return nil
}
