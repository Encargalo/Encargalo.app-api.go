package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

type orderPaymentSubject struct {
	orders  ports.OrdersRepo
	reviews paymentsPorts.PaymentReviewsRepo
	finance paymentsPorts.CustomerBalanceCreditor
	mgr     ports.AppManagerOrder
}

func NewOrderPaymentSubject(
	orders ports.OrdersRepo,
	reviews paymentsPorts.PaymentReviewsRepo,
	finance paymentsPorts.CustomerBalanceCreditor,
	mgr ports.AppManagerOrder,
) paymentsPorts.PaymentSubject {
	return &orderPaymentSubject{orders: orders, reviews: reviews, finance: finance, mgr: mgr}
}

func (s *orderPaymentSubject) ExpectedAmountBs(ctx context.Context, referenceID uuid.UUID) (float64, error) {
	order, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return 0, err
	}
	return pagoMovilAmountBs(order, bcvRateFromPaymentDetails(order)), nil
}

func (s *orderPaymentSubject) CustomerID(ctx context.Context, referenceID uuid.UUID) (uuid.UUID, error) {
	order, err := s.orders.SearchOrdersByID(ctx, referenceID)
	if err != nil {
		return uuid.Nil, err
	}
	if order == nil {
		return uuid.Nil, errcustom.ErrNotFound
	}
	return order.CustomerID, nil
}

func (s *orderPaymentSubject) OnApproved(ctx context.Context, referenceID, reviewID uuid.UUID) error {
	return s.finishApproved(ctx, referenceID, reviewID, paymentsPorts.BalanceCredit{})
}

func (s *orderPaymentSubject) OnOverpaid(ctx context.Context, referenceID, reviewID uuid.UUID, _ float64, credit paymentsPorts.BalanceCredit) error {
	return s.finishApproved(ctx, referenceID, reviewID, credit)
}

func (s *orderPaymentSubject) finishApproved(ctx context.Context, referenceID, reviewID uuid.UUID, credit paymentsPorts.BalanceCredit) error {
	order, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	s.syncPaymentDetails(ctx, order, review, credit)
	_, err = s.releaseToShop(ctx, order)
	return err
}

func (s *orderPaymentSubject) OnRejected(ctx context.Context, referenceID, reviewID uuid.UUID, _ string, credit paymentsPorts.BalanceCredit) error {
	order, err := s.loadInPaymentReview(ctx, referenceID)
	if err != nil {
		return err
	}
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}

	balanceUsedBs := balanceAmountBs(order)
	if err := s.refundBalanceUsed(ctx, order, balanceUsedBs); err != nil {
		return err
	}
	if credit.TxID != uuid.Nil && balanceUsedBs > 0 {
		total := money.RoundBs(credit.RefundablePMBs + balanceUsedBs)
		review.CreditedBs = &total
		if err := s.reviews.UpdateCreditedBs(ctx, review.ID, total); err != nil {
			return err
		}
	} else if credit.TxID == uuid.Nil && balanceUsedBs > 0 {
		review.CreditedBs = &balanceUsedBs
	}

	s.syncPaymentDetails(ctx, order, review, credit)

	if err := s.orders.UpdateOrderStatus(ctx, order.ID, string(models.StatusRejected)); err != nil {
		return err
	}
	order.Status = string(models.StatusRejected)
	if s.mgr != nil {
		_ = s.mgr.BroadcastOrderToCustomer(*order)
	}
	return nil
}

func (s *orderPaymentSubject) EnrichStaffReview(ctx context.Context, review paymentsModels.PaymentReview) (paymentsModels.PaymentReviewBroadcast, error) {
	order, err := s.orders.SearchOrdersByID(ctx, review.ReferenceID)
	if err != nil {
		return paymentsModels.PaymentReviewBroadcast{}, err
	}
	if order == nil {
		return paymentsModels.PaymentReviewBroadcast{}, errcustom.ErrNotFound
	}
	return broadcastFromOrder(&review, order), nil
}

func (s *orderPaymentSubject) loadInPaymentReview(ctx context.Context, referenceID uuid.UUID) (*models.Order, error) {
	order, err := s.orders.SearchOrdersByID(ctx, referenceID)
	if err != nil {
		return nil, err
	}
	if order == nil || order.Status != string(models.StatusPaymentReview) {
		return nil, errcustom.ErrInvalidStatus
	}
	return order, nil
}

func (s *orderPaymentSubject) releaseToShop(ctx context.Context, order *models.Order) (models.Order, error) {
	if err := s.orders.UpdateOrderStatus(ctx, order.ID, string(models.StatusPending)); err != nil {
		return models.Order{}, err
	}
	order.Status = string(models.StatusPending)

	now := time.Now()
	if err := s.orders.MarkPendingAt(ctx, order.ID, now); err != nil {
		slog.Error("releaseToShop MarkPendingAt failed", "orderID", order.ID, "error", err)
		return models.Order{}, err
	}
	order.PendingAt = &now

	if s.mgr != nil {
		if err := s.mgr.BroadcastOrder(*order); err != nil {
			slog.Error("releaseToShop broadcast failed", "orderID", order.ID, "error", err)
		}
	}
	return *order, nil
}

func (s *orderPaymentSubject) refundBalanceUsed(ctx context.Context, order *models.Order, amount float64) error {
	if amount <= 0 || s.finance == nil {
		return nil
	}
	if err := s.finance.RefundBalancePayment(ctx, order.CustomerID, order.ID, paymentsModels.ReferenceTypeOrder, amount); err != nil {
		slog.Error("refundBalanceUsed failed", "orderID", order.ID, "amount", amount, "error", err)
		return err
	}
	return nil
}

// syncPaymentDetails congela el estado de la revisión dentro de order.Details
// para el order_update del cliente. Si credit.TxID != Nil, añade transaction_id
// y la ventana de devolución en efectivo.
func (s *orderPaymentSubject) syncPaymentDetails(
	ctx context.Context,
	order *models.Order,
	review *paymentsModels.PaymentReview,
	credit paymentsPorts.BalanceCredit,
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
	applyBalanceCreditToPaymentDetails(details.PaymentDetails, credit.TxID, creditedBs, credit.RefundablePMBs, review.BcvRate, credit.CreatedAt)

	data, err := json.Marshal(details)
	if err != nil {
		slog.Error("syncPaymentDetails marshal failed", "orderID", order.ID, "error", err)
		return
	}
	order.Details = data
	if err := s.orders.UpdateOrderDetails(ctx, order.ID, data); err != nil {
		slog.Error("syncPaymentDetails persist failed", "orderID", order.ID, "error", err)
	}
}

func broadcastFromOrder(review *paymentsModels.PaymentReview, order *models.Order) paymentsModels.PaymentReviewBroadcast {
	b := paymentsModels.PaymentReviewBroadcast{
		Type:              "payment_review_update",
		ReviewID:          review.ID,
		ReferenceID:       order.ID,
		ReferenceType:     paymentsModels.ReferenceTypeOrder,
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
	return b
}

func balanceAmountBs(order *models.Order) float64 {
	var details models.Details
	if order == nil || len(order.Details) == 0 {
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

func pagoMovilAmountBs(order *models.Order, bcvRate float64) float64 {
	balanceUsedBs, pmFromDetails, hasPD := paymentSplitFromDetails(order, bcvRate)
	totalBs := 0.0
	if bcvRate > 0 {
		totalBs = money.RoundBs(order.TotalPrice * bcvRate)
	}
	return cashRefundablePMBs(totalBs, balanceUsedBs, pmFromDetails, hasPD)
}

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

func cashRefundablePMBs(orderTotalBs, balanceUsedBs, pmFromDetailsBs float64, hasPaymentDetails bool) float64 {
	orderTotalBs = money.RoundBs(orderTotalBs)
	balanceUsedBs = money.RoundBs(balanceUsedBs)
	pmFromDetailsBs = money.RoundBs(pmFromDetailsBs)

	if balanceUsedBs > 0 {
		derived := money.RoundBs(orderTotalBs - balanceUsedBs)
		if derived < 0 {
			return 0
		}
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

var _ paymentsPorts.PaymentSubject = (*orderPaymentSubject)(nil)
var _ paymentsPorts.StaffReviewEnricher = (*orderPaymentSubject)(nil)
