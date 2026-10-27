package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Encargalo.app-api.go/api/router/groups"
	authmw "Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	financemodels "Encargalo.app-api.go/internal/finance/domain/models"
	mockshops "Encargalo.app-api.go/internal/mocks/shops/domain/ports"
	ordersapp "Encargalo.app-api.go/internal/orders/app"
	ordersmodels "Encargalo.app-api.go/internal/orders/domain/models"
	ordersports "Encargalo.app-api.go/internal/orders/domain/ports"
	paymentsmodels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentshandler "Encargalo.app-api.go/internal/payments/handler"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	usersdto "Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStaff_AmountBoundariesStayPending(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	conn := dialStaff(t)
	readConnected(t, conn)

	sendStaff(t, conn, map[string]any{
		"type":               "approve_overpay",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": seed.Expected,
		"reference_last4":    freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	sendStaff(t, conn, map[string]any{
		"type":               "approve_overpay",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": seed.Expected - 1,
		"reference_last4":    freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	sendStaff(t, conn, map[string]any{
		"type":               "reject_underpaid",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": seed.Expected,
		"reference_last4":    freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	sendStaff(t, conn, map[string]any{
		"type":               "reject_underpaid",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": seed.Expected + 1,
		"reference_last4":    freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	require.Equal(t, paymentsmodels.ReviewPendingReview, reloadReview(t, seed.Review.ID).Status)
	require.Equal(t, string(ordersmodels.StatusPaymentReview), reloadOrder(t, seed.Order.ID).Status)
	require.Empty(t, ledgerForReference(t, seed.Order.ID))
}

func TestStaff_DuplicateLast4OnOverpayAndUnderpaidDoesNotCredit(t *testing.T) {
	last4 := consumeLast4(t)

	over := seedPendingOrder(t, newCustomer(t), 0, 400)
	conn := dialStaff(t)
	readConnected(t, conn)
	sendStaff(t, conn, map[string]any{
		"type":               "approve_overpay",
		"review_id":          over.Review.ID,
		"observed_amount_bs": 500.0,
		"reference_last4":    last4,
	})
	expectDuplicateReject(t, conn, over.Review.ID)
	requireRejectedWithoutCredit(t, over, "payment_overage")

	under := seedPendingOrder(t, newCustomer(t), 0, 400)
	sendStaff(t, conn, map[string]any{
		"type":               "reject_underpaid",
		"review_id":          under.Review.ID,
		"observed_amount_bs": 200.0,
		"reference_last4":    last4,
	})
	expectDuplicateReject(t, conn, under.Review.ID)
	requireRejectedWithoutCredit(t, under, "payment_partial_credit")
}

func TestStaff_BadPayloadsDoNotResolve(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	conn := dialStaff(t)
	readConnected(t, conn)

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{`)))
	expectWSError(t, conn, "payload inválido")

	sendStaff(t, conn, map[string]any{"type": "approve_later", "review_id": seed.Review.ID})
	expectWSError(t, conn, "tipo de mensaje desconocido")

	sendStaff(t, conn, map[string]any{"type": "approve", "review_id": seed.Review.ID})
	expectWSError(t, conn, "reference_last4 requerido")

	sendStaff(t, conn, map[string]any{
		"type":            "approve_overpay",
		"review_id":       seed.Review.ID,
		"reference_last4": freshLast4(t),
	})
	expectWSError(t, conn, "observed_amount_bs requerido")

	sendStaff(t, conn, map[string]any{
		"type":               "reject_underpaid",
		"review_id":          seed.Review.ID,
		"observed_amount_bs": 100.0,
	})
	expectWSError(t, conn, "reference_last4 requerido")

	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": "abcd",
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	sendStaff(t, conn, map[string]any{
		"type":      "reject_invalid",
		"review_id": seed.Review.ID,
		"reason":    "   ",
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	require.Equal(t, paymentsmodels.ReviewPendingReview, reloadReview(t, seed.Review.ID).Status)
}

func TestStaff_RepeatAndUnknownReviewDoNotDoubleApply(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	conn := dialStaff(t)
	readConnected(t, conn)
	last4 := freshLast4(t)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": last4,
	})
	readUntil(t, conn, 20*time.Second, func(msgs []map[string]any) bool {
		ok := findMessage(msgs, "action_ok", seed.Review.ID)
		return ok != nil && ok["action"] == "approve"
	})

	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())
	require.Equal(t, paymentsmodels.ReviewApproved, reloadReview(t, seed.Review.ID).Status)
	require.Equal(t, string(ordersmodels.StatusPending), reloadOrder(t, seed.Order.ID).Status)
	require.Empty(t, ledgerForReference(t, seed.Order.ID))

	missing := uuid.New()
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       missing,
		"reference_last4": freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrNotFound.Error())
	_, err := reviewsRepo.GetByID(context.Background(), missing)
	require.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestStaff_OrderOutsidePaymentReviewFailsAfterResolvingTheReview(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	setOrderColumn(t, seed.Order.ID, "status", string(ordersmodels.StatusPending))

	conn := dialStaff(t)
	readConnected(t, conn)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": freshLast4(t),
	})
	expectWSError(t, conn, errcustom.ErrInvalidStatus.Error())

	// El CAS de la review ocurre antes de que el subject valide el estado de la orden.
	require.Equal(t, paymentsmodels.ReviewApproved, reloadReview(t, seed.Review.ID).Status)
	require.Equal(t, string(ordersmodels.StatusPending), reloadOrder(t, seed.Order.ID).Status)
	require.Empty(t, ledgerForReference(t, seed.Order.ID))
}

func TestStaff_OtherConnectionSeesTheResolution(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	first := dialStaff(t)
	second := dialStaff(t)
	readConnected(t, first)
	readConnected(t, second)

	sendStaff(t, first, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": freshLast4(t),
	})
	readUntil(t, second, 20*time.Second, func(msgs []map[string]any) bool {
		return hasReviewStatus(msgs, seed.Review.ID, "approved")
	})
}

func TestStaff_ExactApproveKeepsBalanceDebit(t *testing.T) {
	customerID := newCustomer(t)
	_, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", 80, bcvRate)
	require.NoError(t, err)
	seed := seedPendingOrder(t, customerID, 80, 320)
	require.NoError(t, reviewsApp.DebitForReference(context.Background(), customerID, seed.Order.ID, "order", 80))

	conn := dialStaff(t)
	readConnected(t, conn)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": freshLast4(t),
	})
	readUntil(t, conn, 20*time.Second, func(msgs []map[string]any) bool {
		ok := findMessage(msgs, "action_ok", seed.Review.ID)
		return ok != nil && ok["action"] == "approve"
	})

	require.Equal(t, string(ordersmodels.StatusPending), reloadOrder(t, seed.Order.ID).Status)
	var refunded bool
	for _, row := range ledgerForReference(t, seed.Order.ID) {
		require.NotEqual(t, "balance_payment_refund", row.MovementType)
		if row.MovementType == "balance_payment" {
			refunded = true
			require.InDelta(t, -80, row.Amount, 0.001)
		}
	}
	require.True(t, refunded)
	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.InDelta(t, 0, balance.BalanceBs, 0.001)
}

func TestCreateReview_RejectsEachIncompleteField(t *testing.T) {
	_, err := reviewsPort.CreateReview(context.Background(), paymentsmodels.ReferenceTypeOrder, uuid.Nil, proofURL, 10, bcvRate)
	require.ErrorIs(t, err, errcustom.ErrInvalidStatus)
	_, err = reviewsPort.CreateReview(context.Background(), paymentsmodels.ReferenceTypeOrder, uuid.New(), "  ", 10, bcvRate)
	require.ErrorIs(t, err, errcustom.ErrInvalidStatus)
	_, err = reviewsPort.CreateReview(context.Background(), paymentsmodels.ReferenceTypeOrder, uuid.New(), proofURL, 0, bcvRate)
	require.ErrorIs(t, err, errcustom.ErrInvalidStatus)
}

func TestOrders_ShopRejectCreditsOrderReference(t *testing.T) {
	customerID := newCustomer(t)
	_, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", 80, bcvRate)
	require.NoError(t, err)
	seed := seedPendingOrder(t, customerID, 0, 400)
	require.NoError(t, reviewsApp.DebitForReference(context.Background(), customerID, seed.Order.ID, "order", 80))

	conn := dialStaff(t)
	readConnected(t, conn)
	sendStaff(t, conn, map[string]any{
		"type":            "approve",
		"review_id":       seed.Review.ID,
		"reference_last4": freshLast4(t),
	})
	readUntil(t, conn, 20*time.Second, func(msgs []map[string]any) bool {
		ok := findMessage(msgs, "action_ok", seed.Review.ID)
		return ok != nil && ok["action"] == "approve"
	})

	app := newShopOrderApp(t)
	rejected, err := app.UpdateOrderStatus(context.Background(), seed.Order.ID, string(ordersmodels.StatusRejected), shopID, true, shopID)
	require.NoError(t, err)
	require.Equal(t, string(ordersmodels.StatusRejected), rejected.Status)

	rows := ledgerForReference(t, seed.Order.ID)
	var credited float64
	var refundable *float64
	for _, row := range rows {
		if row.MovementType != "order_rejected_refund" {
			continue
		}
		credited = row.Amount
		require.Equal(t, "order", *row.ReferenceType)
		require.Nil(t, row.OrderID)
		refundable = refundablePM(t, row.ID)
	}
	require.InDelta(t, 400, credited, 0.001)
	require.NotNil(t, refundable)
	require.InDelta(t, 320, *refundable, 0.001)

	pd := paymentDetails(t, reloadOrder(t, seed.Order.ID))
	require.Equal(t, paymentsmodels.ReviewRefundedShopReject, pd.Status)
	require.NotNil(t, pd.TransactionID)

	balance, err := reviewsApp.GetBalance(context.Background(), customerID)
	require.NoError(t, err)
	require.InDelta(t, 400, balance.BalanceBs, 0.001)
	var sawOrder bool
	for _, tx := range balance.Transactions {
		if tx.MovementType == "order_rejected_refund" {
			require.NotNil(t, tx.OrderID)
			require.Equal(t, seed.Order.ID.String(), *tx.OrderID)
			sawOrder = true
		}
	}
	require.True(t, sawOrder)
}

func TestOrders_ShopRejectIgnoresCashAndWrongCaller(t *testing.T) {
	customerID := newCustomer(t)
	seed := seedPendingOrder(t, customerID, 0, 400)
	setOrderColumn(t, seed.Order.ID, "status", string(ordersmodels.StatusPending))
	app := newShopOrderApp(t)

	_, err := app.UpdateOrderStatus(context.Background(), seed.Order.ID, string(ordersmodels.StatusRejected), customerID, false, uuid.Nil)
	require.ErrorIs(t, err, errcustom.ErrUnautorized)

	_, err = app.UpdateOrderStatus(context.Background(), seed.Order.ID, string(ordersmodels.StatusRejected), uuid.New(), true, uuid.New())
	require.ErrorIs(t, err, errcustom.ErrUnautorized)
	require.Equal(t, string(ordersmodels.StatusPending), reloadOrder(t, seed.Order.ID).Status)
	require.Empty(t, ledgerForReference(t, seed.Order.ID))

	setOrderColumn(t, seed.Order.ID, "method_payment", "EfectivoBs")
	rejected, err := app.UpdateOrderStatus(context.Background(), seed.Order.ID, string(ordersmodels.StatusRejected), shopID, true, shopID)
	require.NoError(t, err)
	require.Equal(t, string(ordersmodels.StatusRejected), rejected.Status)
	require.Empty(t, ledgerForReference(t, seed.Order.ID))
}

func TestFinance_RefundRequestErrors(t *testing.T) {
	customerID := newCustomer(t)
	txID, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", 10, bcvRate)
	require.NoError(t, err)

	_, err = reviewsApp.CreateRefundRequest(context.Background(), customerID, txID)
	require.ErrorIs(t, err, errcustom.ErrPagoMovilRequired)

	_, err = reviewsApp.UpsertPagoMovil(context.Background(), customerID, pagoMovil())
	require.NoError(t, err)
	_, err = reviewsApp.CreateRefundRequest(context.Background(), customerID, txID)
	require.ErrorIs(t, err, errcustom.ErrRefundNotEligible)

	debitRef := uuid.New()
	require.NoError(t, reviewsApp.DebitForReference(context.Background(), customerID, debitRef, "order", 5))
	var debitID uuid.UUID
	err = sqlDB.QueryRowContext(context.Background(), `
		SELECT id FROM finance.transactions
		WHERE reference_id = $1 AND movement_type = 'balance_payment'
	`, debitRef).Scan(&debitID)
	require.NoError(t, err)
	_, err = reviewsApp.CreateRefundRequest(context.Background(), customerID, debitID)
	require.ErrorIs(t, err, errcustom.ErrRefundNotEligible)

	eligibleID, _, err := reviewsApp.CreditPaymentOverage(context.Background(), customerID, uuid.New(), "order", 80, bcvRate)
	require.NoError(t, err)
	other := newCustomer(t)
	_, err = reviewsApp.UpsertPagoMovil(context.Background(), other, usersdto.UpsertPagoMovilRequest{
		Phone: nextPhone(), BankCode: "0102", NationalID: "V87654321",
	})
	require.NoError(t, err)
	_, err = reviewsApp.CreateRefundRequest(context.Background(), other, eligibleID)
	require.ErrorIs(t, err, errcustom.ErrNotFound)

	first, err := reviewsApp.CreateRefundRequest(context.Background(), customerID, eligibleID)
	require.NoError(t, err)
	require.NotNil(t, first.OrderID)
	_, err = reviewsApp.CreateRefundRequest(context.Background(), customerID, eligibleID)
	require.ErrorIs(t, err, errcustom.ErrRefundAlreadyRequested)
}

func TestFinance_StaffPayoutModelWritesReferenceColumns(t *testing.T) {
	refID := uuid.New()
	refType := "order"
	row := &financemodels.Transaction{
		ID:            uuid.New(),
		EntityID:      uuid.New(),
		EntityType:    "shop",
		ReferenceID:   &refID,
		ReferenceType: &refType,
		Amount:        15,
		Currency:      "VES",
		MovementType:  string(financemodels.MovementTypeStaffPayout),
	}
	_, err := db.NewInsert().Model(row).Exec(context.Background())
	require.NoError(t, err)

	var got ledgerRow
	err = db.NewSelect().Model(&got).Where("id = ?", row.ID).Scan(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got.ReferenceID)
	require.Equal(t, refID, *got.ReferenceID)
	require.NotNil(t, got.ReferenceType)
	require.Equal(t, "order", *got.ReferenceType)
	require.Nil(t, got.OrderID)
}

func TestRoutes_RealAuthGuardsStaffReviews(t *testing.T) {
	secret := config.Get().InternalGateway.Secret
	mw := authmw.NewAuthMidlleware(sharedmw.NewGatewaySecretAuth(*config.Get()))
	e := echoNewPayments(mw)
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)

	_, resp, err := dialAuth(server, nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	_ = resp.Body.Close()

	_, resp, err = dialAuth(server, http.Header{
		sharedmw.GatewaySecretHeader: []string{secret},
	})
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	_ = resp.Body.Close()

	_, resp, err = dialAuth(server, http.Header{
		sharedmw.GatewaySecretHeader: []string{secret},
		sharedmw.SessionHeader:       []string{`{`},
	})
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	_ = resp.Body.Close()

	_, resp, err = dialAuth(server, authHeaders(t, secret, authmodels.Customer))
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	_ = resp.Body.Close()

	conn, resp, err := dialAuth(server, authHeaders(t, secret, authmodels.Staff))
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	_ = resp.Body.Close()
	t.Cleanup(func() { _ = conn.Close() })
	readConnected(t, conn)
}

func echoNewPayments(mw authmw.AuthMiddleware) *echo.Echo {
	e := echo.New()
	groups.NewPaymentsGroup(mw, paymentshandler.NewStaffReviewsHandler(hub, reviewsPort)).Resource(e)
	return e
}

func dialAuth(server *httptest.Server, headers http.Header) (*websocket.Conn, *http.Response, error) {
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/payments/staff/reviews"
	return websocket.DefaultDialer.Dial(wsURL, headers)
}

func authHeaders(t *testing.T, secret string, role authmodels.Role) http.Header {
	t.Helper()
	raw, err := json.Marshal(authmodels.ActiveSession{UserID: staffID, Roles: []authmodels.Role{role}})
	require.NoError(t, err)
	return http.Header{
		sharedmw.GatewaySecretHeader: []string{secret},
		sharedmw.SessionHeader:       []string{string(raw)},
	}
}

func readConnected(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	readUntil(t, conn, 20*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "connected", uuid.Nil)
	})
}

func expectWSError(t *testing.T, conn *websocket.Conn, message string) {
	t.Helper()
	done := readUntil(t, conn, 20*time.Second, func(msgs []map[string]any) bool {
		return findMessage(msgs, "error", uuid.Nil) != nil
	})
	require.Contains(t, fmt.Sprint(findMessage(done, "error", uuid.Nil)["message"]), message)
}

func expectDuplicateReject(t *testing.T, conn *websocket.Conn, reviewID uuid.UUID) {
	t.Helper()
	done := readUntil(t, conn, 20*time.Second, func(msgs []map[string]any) bool {
		return hasMessage(msgs, "action_ok", reviewID)
	})
	action := findMessage(done, "action_ok", reviewID)
	require.Equal(t, "reject_invalid", action["action"])
	require.Equal(t, true, action["duplicate_reject"])
}

func requireRejectedWithoutCredit(t *testing.T, seed seededOrder, forbidden string) {
	t.Helper()
	require.Equal(t, string(ordersmodels.StatusRejected), reloadOrder(t, seed.Order.ID).Status)
	review := reloadReview(t, seed.Review.ID)
	require.Equal(t, paymentsmodels.ReviewRejectedInvalid, review.Status)
	require.Nil(t, review.ReferenceLast4)
	for _, row := range ledgerForReference(t, seed.Order.ID) {
		require.NotEqual(t, forbidden, row.MovementType)
	}
}

func consumeLast4(t *testing.T) string {
	t.Helper()
	resolved, err := reviewsPort.CreateReview(context.Background(), "trip", uuid.New(), proofURL, 100, bcvRate)
	require.NoError(t, err)
	last4 := freshLast4(t)
	ok, err := reviewsRepo.ResolvePending(context.Background(), resolved.ID, &paymentsmodels.PaymentReview{
		Status:         paymentsmodels.ReviewApproved,
		StaffID:        &staffID,
		ReferenceLast4: &last4,
	})
	require.NoError(t, err)
	require.True(t, ok)
	return last4
}

func hasReviewStatus(msgs []map[string]any, reviewID uuid.UUID, status string) bool {
	for _, msg := range msgs {
		if msg["type"] == "payment_review_update" && msg["review_id"] == reviewID.String() && msg["status"] == status {
			return true
		}
	}
	return false
}

func setOrderColumn(t *testing.T, id uuid.UUID, column, value string) {
	t.Helper()
	_, err := db.NewUpdate().Table("orders.orders").Set(column+" = ?", value).Where("id = ?", id).Exec(context.Background())
	require.NoError(t, err)
}

func refundablePM(t *testing.T, txID uuid.UUID) *float64 {
	t.Helper()
	var amount *float64
	err := sqlDB.QueryRowContext(context.Background(), `
		SELECT refundable_pm_bs FROM finance.transactions WHERE id = $1
	`, txID).Scan(&amount)
	require.NoError(t, err)
	return amount
}

func newShopOrderApp(t *testing.T) ordersports.OrdersApp {
	t.Helper()
	shops := mockshops.NewShopsRepo(t)
	shops.On("TryCloseOpenBatch", mock.Anything, shopID).Return(nil).Maybe()
	return ordersapp.NewOrderApp(
		ordersRepo, shops, nil, nil, nil, quietRider{}, nil, nil, fixedBCV{},
		reviewsRepo, hub, reviewsApp, nil, nil, nil,
	)
}

type quietRider struct{}

func (quietRider) BroadcastNewOrder(context.Context, ordersmodels.Order) {}

func (quietRider) BroadcastOrderUpdate(context.Context, ordersmodels.Order) {}

type fixedBCV struct{}

func (fixedBCV) Start(context.Context) {}

func (fixedBCV) GetRate(context.Context) (*bcvmodels.ExchangeRate, error) {
	return &bcvmodels.ExchangeRate{Rate: bcvRate}, nil
}
