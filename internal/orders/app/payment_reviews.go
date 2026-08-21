package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

var referenceLast4Pattern = regexp.MustCompile(`^[0-9]{4}$`)

type paymentReviewsApp struct {
	repo    ports.PaymentReviewsRepo
	orders  ports.OrdersRepo
	finance ports.CustomerBalanceCreditor
	stream  ports.RedisStream
	hub     ports.StaffReviewHub
	mgr     ports.AppManagerOrder
}

func NewPaymentReviewsApp(
	repo ports.PaymentReviewsRepo,
	orders ports.OrdersRepo,
	finance ports.CustomerBalanceCreditor,
	stream ports.RedisStream,
	hub ports.StaffReviewHub,
	mgr ports.AppManagerOrder,
) ports.PaymentReviewsApp {
	return &paymentReviewsApp{repo, orders, finance, stream, hub, mgr}
}

func (a *paymentReviewsApp) CreateForOrder(ctx context.Context, order *models.Order, proofURL string, expectedBs, bcvRate float64) (*models.PaymentReview, error) {
	now := time.Now()
	review := &models.PaymentReview{
		ID:               uuid.New(),
		OrderID:          order.ID,
		ProofURL:         proofURL,
		ExpectedAmountBs: expectedBs,
		BcvRate:          bcvRate,
		Status:           models.ReviewPendingReview,
		CreatedAt:        now,
		LastAlertedAt:    &now,
	}
	if err := a.repo.Create(ctx, review); err != nil {
		return nil, err
	}

	event := models.DataEventPaymentProofPending{}
	event.BuildFrom(*order, *review, 0)
	if err := a.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("error producing payment_proof_pending", "error", err)
	}

	if a.hub != nil {
		_ = a.hub.BroadcastReview(a.BuildBroadcast(ctx, review, order))
	}

	return review, nil
}

func (a *paymentReviewsApp) Approve(ctx context.Context, reviewID, staffID uuid.UUID, referenceLast4 string) (models.StaffReviewResult, error) {
	last4, dup, err := a.prepareReferenceLast4(ctx, reviewID, staffID, referenceLast4)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if dup != nil {
		return *dup, nil
	}

	review, order, err := a.loadPending(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}

	update := &models.PaymentReview{
		Status:         models.ReviewApproved,
		StaffID:        &staffID,
		ReferenceLast4: &last4,
	}
	ok, err := a.repo.ResolvePending(ctx, review.ID, update)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if !ok {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}

	orderOut, err := a.releaseToShop(ctx, order)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	return models.StaffReviewResult{Order: orderOut, Action: "approve"}, nil
}

func (a *paymentReviewsApp) ApproveOverpay(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error) {
	last4, dup, err := a.prepareReferenceLast4(ctx, reviewID, staffID, referenceLast4)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if dup != nil {
		return *dup, nil
	}

	review, order, err := a.loadPending(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if observedBs <= review.ExpectedAmountBs {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}

	diff := observedBs - review.ExpectedAmountBs
	creditedBs := money.RoundBs(diff)

	update := &models.PaymentReview{
		Status:           models.ReviewOverpaid,
		StaffID:          &staffID,
		ObservedAmountBs: &observedBs,
		DifferenceBs:     &diff,
		CreditedBs:       &creditedBs,
		ReferenceLast4:   &last4,
	}
	ok, err := a.repo.ResolvePending(ctx, review.ID, update)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if !ok {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}

	txID, createdAt, err := a.finance.CreditPaymentOverage(ctx, order.CustomerID, order.ID, creditedBs, review.BcvRate)
	if err != nil {
		slog.Error("ApproveOverpay credit failed", "orderID", order.ID, "error", err)
		return models.StaffReviewResult{}, err
	}

	review.Status = models.ReviewOverpaid
	review.ObservedAmountBs = &observedBs
	review.DifferenceBs = &diff
	review.CreditedBs = &creditedBs
	review.ReferenceLast4 = &last4
	a.syncPaymentDetails(ctx, order, review, txID, creditedBs, createdAt)

	orderOut, err := a.releaseToShop(ctx, order)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	return models.StaffReviewResult{Order: orderOut, Action: "approve_overpay"}, nil
}

// RejectUnderpaid resuelve un comprobante con monto insuficiente: acredita en
// Bs lo que efectivamente se pagó (observedBs) al balance del cliente y rechaza
// la orden de inmediato.
// No hay espera ni comprobante adicional para completar la diferencia: si el
// cliente quiere reintentar, debe armar una nueva orden (el front decide si
// le ofrece pagar con su balance + un nuevo comprobante).
func (a *paymentReviewsApp) RejectUnderpaid(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error) {
	last4, dup, err := a.prepareReferenceLast4(ctx, reviewID, staffID, referenceLast4)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if dup != nil {
		return *dup, nil
	}

	review, order, err := a.loadPending(ctx, reviewID)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if observedBs >= review.ExpectedAmountBs {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}

	diff := review.ExpectedAmountBs - observedBs
	pmCreditedBs := money.RoundBs(observedBs)
	balanceUsedBs := balanceAmountBs(order)
	totalCreditedBs := money.RoundBs(pmCreditedBs + balanceUsedBs)

	update := &models.PaymentReview{
		Status:           models.ReviewRejectedUnderpaid,
		StaffID:          &staffID,
		ObservedAmountBs: &observedBs,
		DifferenceBs:     &diff,
		CreditedBs:       &totalCreditedBs,
		ReferenceLast4:   &last4,
	}
	ok, err := a.repo.ResolvePending(ctx, review.ID, update)
	if err != nil {
		return models.StaffReviewResult{}, err
	}
	if !ok {
		return models.StaffReviewResult{}, errcustom.ErrInvalidStatus
	}

	txID, createdAt, err := a.finance.CreditPaymentPartial(ctx, order.CustomerID, order.ID, pmCreditedBs, review.BcvRate)
	if err != nil {
		slog.Error("RejectUnderpaid credit failed", "orderID", order.ID, "error", err)
		return models.StaffReviewResult{}, err
	}
	if err := a.refundBalanceUsed(ctx, order, balanceUsedBs); err != nil {
		return models.StaffReviewResult{}, err
	}

	if err := a.orders.UpdateOrderStatus(ctx, order.ID, string(models.StatusRejected)); err != nil {
		return models.StaffReviewResult{}, err
	}
	order.Status = string(models.StatusRejected)

	review.Status = models.ReviewRejectedUnderpaid
	review.ObservedAmountBs = &observedBs
	review.DifferenceBs = &diff
	review.CreditedBs = &totalCreditedBs
	review.StaffID = &staffID
	review.ReferenceLast4 = &last4
	// Solo el crédito PM es cash-refundable (no el refund del balance usado).
	a.syncPaymentDetails(ctx, order, review, txID, pmCreditedBs, createdAt)

	if a.mgr != nil {
		_ = a.mgr.BroadcastOrderToCustomer(*order)
	}

	return models.StaffReviewResult{Order: *order, Action: "reject_underpaid"}, nil
}

func (a *paymentReviewsApp) RejectInvalid(ctx context.Context, reviewID, staffID uuid.UUID, reason string) (models.Order, error) {
	if reason == "" {
		return models.Order{}, errcustom.ErrInvalidStatus
	}
	review, order, err := a.loadPending(ctx, reviewID)
	if err != nil {
		return models.Order{}, err
	}

	update := &models.PaymentReview{
		Status:          models.ReviewRejectedInvalid,
		StaffID:         &staffID,
		RejectionReason: &reason,
	}
	ok, err := a.repo.ResolvePending(ctx, review.ID, update)
	if err != nil {
		return models.Order{}, err
	}
	if !ok {
		return models.Order{}, errcustom.ErrInvalidStatus
	}

	balanceUsedBs := balanceAmountBs(order)
	if err := a.refundBalanceUsed(ctx, order, balanceUsedBs); err != nil {
		return models.Order{}, err
	}

	if err := a.orders.UpdateOrderStatus(ctx, order.ID, string(models.StatusRejected)); err != nil {
		return models.Order{}, err
	}
	order.Status = string(models.StatusRejected)

	review.Status = models.ReviewRejectedInvalid
	review.RejectionReason = &reason
	if balanceUsedBs > 0 {
		review.CreditedBs = &balanceUsedBs
	}
	a.syncPaymentDetails(ctx, order, review, uuid.Nil, 0, time.Time{})

	if a.mgr != nil {
		_ = a.mgr.BroadcastOrderToCustomer(*order)
	}

	return *order, nil
}

func (a *paymentReviewsApp) ListActiveForStaff(ctx context.Context) ([]models.PaymentReviewBroadcast, error) {
	reviews, err := a.repo.ListActiveForStaff(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.PaymentReviewBroadcast, 0, len(reviews))
	for i := range reviews {
		order, err := a.orders.SearchOrdersByID(ctx, reviews[i].OrderID)
		if err != nil {
			continue
		}
		out = append(out, a.BuildBroadcast(ctx, &reviews[i], order))
	}
	return out, nil
}

func (a *paymentReviewsApp) NotifyPendingPaymentProofs(ctx context.Context) error {
	if a.stream == nil {
		return nil
	}
	now := time.Now()
	alertBefore := now.Add(-models.PaymentProofAlertInterval)
	list, err := a.repo.ListPendingForAlert(ctx, alertBefore)
	if err != nil {
		return err
	}
	for i := range list {
		review := list[i]
		ok, err := a.repo.MarkAlerted(ctx, review.ID, now, alertBefore)
		if err != nil {
			slog.Error("NotifyPendingPaymentProofs MarkAlerted failed", "reviewID", review.ID, "error", err)
			continue
		}
		if !ok {
			continue
		}
		order, err := a.orders.SearchOrdersByID(ctx, review.OrderID)
		if err != nil || order == nil {
			slog.Error("NotifyPendingPaymentProofs order lookup failed", "reviewID", review.ID, "orderID", review.OrderID, "error", err)
			continue
		}
		minutes := max(int(now.Sub(review.CreatedAt)/time.Minute), 0)
		event := models.DataEventPaymentProofPending{}
		event.BuildFrom(*order, review, minutes)
		if err := a.stream.PublisherEvent(ctx, event); err != nil {
			slog.Error("NotifyPendingPaymentProofs publish failed", "reviewID", review.ID, "error", err)
			continue
		}
		slog.Info("payment_proof_pending published",
			"reviewID", review.ID,
			"orderID", review.OrderID,
			"minutesPending", minutes,
		)
	}
	return nil
}

// syncPaymentDetails congela el estado actual de la revisión (montos,
// diferencia, motivo de rechazo, monto acreditado) dentro de order.Details
// para que el cliente lo reciba en el mismo order_update por WS y quede
// persistido para el snapshot al reconectar (SearchActiveOrdersByCriteria).
// Si creditTxID != Nil, añade transaction_id / can_request_refund / ventana.
func (a *paymentReviewsApp) syncPaymentDetails(
	ctx context.Context,
	order *models.Order,
	review *models.PaymentReview,
	creditTxID uuid.UUID,
	refundablePMBs float64,
	creditCreatedAt time.Time,
) {
	var details models.Details
	if len(order.Details) > 0 {
		_ = json.Unmarshal(order.Details, &details)
	}
	if details.PaymentDetails == nil {
		details.PaymentDetails = &models.PaymentDetails{}
	}
	details.PaymentDetails.PaymentProofURL = review.ProofURL
	details.PaymentDetails.Status = review.Status
	details.PaymentDetails.ExpectedAmountBs = review.ExpectedAmountBs
	details.PaymentDetails.ObservedAmountBs = review.ObservedAmountBs
	details.PaymentDetails.DifferenceBs = review.DifferenceBs
	details.PaymentDetails.RejectionReason = review.RejectionReason
	details.PaymentDetails.CreditedBs = review.CreditedBs

	creditedBs := 0.0
	if review.CreditedBs != nil {
		creditedBs = *review.CreditedBs
	}
	applyBalanceCreditToPaymentDetails(details.PaymentDetails, creditTxID, creditedBs, refundablePMBs, review.BcvRate, creditCreatedAt)

	data, err := json.Marshal(details)
	if err != nil {
		slog.Error("syncPaymentDetails marshal failed", "orderID", order.ID, "error", err)
		return
	}
	order.Details = data
	if err := a.orders.UpdateOrderDetails(ctx, order.ID, data); err != nil {
		slog.Error("syncPaymentDetails persist failed", "orderID", order.ID, "error", err)
	}
}

func (a *paymentReviewsApp) BuildBroadcast(ctx context.Context, review *models.PaymentReview, order *models.Order) models.PaymentReviewBroadcast {
	b := models.PaymentReviewBroadcast{
		Type:              "payment_review_update",
		ReviewID:          review.ID,
		OrderID:           order.ID,
		ShopID:            order.ShopID,
		CustomerID:        order.CustomerID,
		ProofURL:          review.ProofURL,
		ExpectedAmountBs:  review.ExpectedAmountBs,
		ObservedAmountBs:  review.ObservedAmountBs,
		DifferenceBs:      review.DifferenceBs,
		Status:            review.Status,
		RejectionReason:   review.RejectionReason,
		CreditedBs:        review.CreditedBs,
		BalanceAmountBs:   balanceAmountBs(order),
		PagoMovilAmountBs: pagoMovilAmountBs(order, review.BcvRate),
		TotalUSD:          order.TotalPrice,
		OrderStatus:       order.Status,
		CreatedAt:         review.CreatedAt,
	}

	var details models.Details
	if len(order.Details) > 0 {
		if err := json.Unmarshal(order.Details, &details); err == nil {
			b.CustomerName = details.CustomerDetails.Name
			if details.CustomerDetails.LastName != "" {
				b.CustomerName = details.CustomerDetails.Name + " " + details.CustomerDetails.LastName
			}
			b.CustomerPhone = details.CustomerDetails.Phone
		}
	}
	_ = ctx
	return b
}

func balanceAmountBs(order *models.Order) float64 {
	var details models.Details
	if len(order.Details) == 0 {
		return 0
	}
	if err := json.Unmarshal(order.Details, &details); err != nil || details.PaymentDetails == nil {
		return 0
	}
	return details.PaymentDetails.BalanceAmountBs
}

func bcvRateFromPaymentDetails(order *models.Order) float64 {
	if order == nil || len(order.Details) == 0 {
		return 0
	}
	var details models.Details
	if err := json.Unmarshal(order.Details, &details); err != nil || details.PaymentDetails == nil {
		return 0
	}
	return details.PaymentDetails.BcvRate
}

// pagoMovilAmountBs es la porción pagada por Pago Móvil (cash-refundable),
// en Bs. Nunca debe incluir lo pagado con balance.
func pagoMovilAmountBs(order *models.Order, bcvRate float64) float64 {
	balanceUsedBs, pmFromDetails, hasPD := paymentSplitFromDetails(order, bcvRate)
	totalBs := 0.0
	if bcvRate > 0 {
		totalBs = money.RoundBs(order.TotalPrice * bcvRate)
	}
	return cashRefundablePMBs(totalBs, balanceUsedBs, pmFromDetails, hasPD)
}

// paymentSplitFromDetails devuelve el balance usado y la porción Pago Móvil
// en Bs, tal cual quedaron en order.Details al crear la orden. Pedidos
// viejos con solo pago_movil_amount_usd se convierten con rate.
func paymentSplitFromDetails(order *models.Order, rate float64) (balanceUsedBs, pmFromDetailsBs float64, hasPaymentDetails bool) {
	if order == nil || len(order.Details) == 0 {
		return 0, 0, false
	}
	var details models.Details
	if err := json.Unmarshal(order.Details, &details); err != nil || details.PaymentDetails == nil {
		return 0, 0, false
	}
	pmBs := details.PaymentDetails.PagoMovilAmountBs
	if pmBs <= 0 && details.PaymentDetails.PagoMovilAmountUSD > 0 && rate > 0 {
		pmBs = money.RoundBs(details.PaymentDetails.PagoMovilAmountUSD * rate)
	}
	return details.PaymentDetails.BalanceAmountBs, pmBs, true
}

// cashRefundablePMBs calcula el monto elegible para devolución en efectivo.
// Si hubo pago con balance, el cash-refundable es total−balance (no el total entero).
func cashRefundablePMBs(orderTotalBs, balanceUsedBs, pmFromDetailsBs float64, hasPaymentDetails bool) float64 {
	orderTotalBs = money.RoundBs(orderTotalBs)
	balanceUsedBs = money.RoundBs(balanceUsedBs)
	pmFromDetailsBs = money.RoundBs(pmFromDetailsBs)

	if balanceUsedBs > 0 {
		derived := money.RoundBs(orderTotalBs - balanceUsedBs)
		if derived < 0 {
			return 0
		}
		// Si details trae un PM que incluye el balance (p.ej. igual al total), ignorarlo.
		if pmFromDetailsBs > 0 && pmFromDetailsBs <= derived+0.01 {
			return pmFromDetailsBs
		}
		return derived
	}
	if pmFromDetailsBs > 0 {
		return pmFromDetailsBs
	}
	if !hasPaymentDetails {
		return orderTotalBs
	}
	return orderTotalBs
}

func (a *paymentReviewsApp) refundBalanceUsed(ctx context.Context, order *models.Order, amount float64) error {
	if amount <= 0 || a.finance == nil {
		return nil
	}
	if err := a.finance.RefundBalancePayment(ctx, order.CustomerID, order.ID, amount); err != nil {
		slog.Error("refundBalanceUsed failed", "orderID", order.ID, "amount", amount, "error", err)
		return err
	}
	return nil
}

// prepareReferenceLast4 valida formato y, si el last4 ya fue consumido,
// auto-rechaza la review como rejected_invalid (sin persistir el last4).
func (a *paymentReviewsApp) prepareReferenceLast4(
	ctx context.Context,
	reviewID, staffID uuid.UUID,
	referenceLast4 string,
) (last4 string, duplicate *models.StaffReviewResult, err error) {
	last4 = strings.TrimSpace(referenceLast4)
	if !referenceLast4Pattern.MatchString(last4) {
		return "", nil, errcustom.ErrInvalidStatus
	}

	exists, err := a.repo.ExistsResolvedReferenceLast4(ctx, last4)
	if err != nil {
		return "", nil, err
	}
	if !exists {
		return last4, nil, nil
	}

	reason := fmt.Sprintf("comprobante duplicado (referencia %s)", last4)
	order, err := a.RejectInvalid(ctx, reviewID, staffID, reason)
	if err != nil {
		return "", nil, err
	}
	return last4, &models.StaffReviewResult{
		Order:           order,
		Action:          "reject_invalid",
		DuplicateReject: true,
		RejectionReason: reason,
	}, nil
}

func (a *paymentReviewsApp) loadPending(ctx context.Context, reviewID uuid.UUID) (*models.PaymentReview, *models.Order, error) {
	review, err := a.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, nil, err
	}
	if review.Status != models.ReviewPendingReview {
		return nil, nil, errcustom.ErrInvalidStatus
	}
	order, err := a.orders.SearchOrdersByID(ctx, review.OrderID)
	if err != nil {
		return nil, nil, err
	}
	if order.Status != string(models.StatusPaymentReview) {
		return nil, nil, errcustom.ErrInvalidStatus
	}
	return review, order, nil
}

func (a *paymentReviewsApp) releaseToShop(ctx context.Context, order *models.Order) (models.Order, error) {
	if err := a.orders.UpdateOrderStatus(ctx, order.ID, string(models.StatusPending)); err != nil {
		return models.Order{}, err
	}
	order.Status = string(models.StatusPending)

	now := time.Now()
	if err := a.orders.MarkPendingAt(ctx, order.ID, now); err != nil {
		slog.Error("releaseToShop MarkPendingAt failed", "orderID", order.ID, "error", err)
		return models.Order{}, err
	}
	order.PendingAt = &now

	if a.mgr != nil {
		if err := a.mgr.BroadcastOrder(*order); err != nil {
			slog.Error("releaseToShop broadcast failed", "orderID", order.ID, "error", err)
		}
	}
	return *order, nil
}
