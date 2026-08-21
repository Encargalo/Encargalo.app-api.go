package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"github.com/google/uuid"
)

type PaymentReviewsRepo interface {
	Create(ctx context.Context, review *models.PaymentReview) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PaymentReview, error)
	ListActiveForStaff(ctx context.Context) ([]models.PaymentReview, error)
	// ResolvePending atomically updates a pending_review row. Returns false if already resolved.
	ResolvePending(ctx context.Context, id uuid.UUID, update *models.PaymentReview) (bool, error)
	// ExistsResolvedReferenceLast4 reports whether last4 is already used by an
	// approved / overpaid / rejected_underpaid review.
	ExistsResolvedReferenceLast4(ctx context.Context, last4 string) (bool, error)
	ListPendingForAlert(ctx context.Context, alertBefore time.Time) ([]models.PaymentReview, error)
	MarkAlerted(ctx context.Context, id uuid.UUID, alertedAt, alertBefore time.Time) (bool, error)
}

// CustomerBalanceCreditor is implemented by users finance to credit/debit the
// customer's Bs balance. Credit* reciben el monto ya en Bs + la tasa BCV
// congelada (review.BcvRate / quote.BcvRate) para grabarla en el movimiento;
// Debit/Refund operan directo en Bs (ya es lo que el customer pidió usar).
type CustomerBalanceCreditor interface {
	GetBalanceBs(ctx context.Context, customerID uuid.UUID) (float64, error)
	// GetBalancePaymentDebitForOrder returns the Bs debited from balance for this order
	// (absolute value of balance_payment movements). 0 if none.
	GetBalancePaymentDebitForOrder(ctx context.Context, customerID, orderID uuid.UUID) (float64, error)
	CreditPaymentOverage(ctx context.Context, customerID, orderID uuid.UUID, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditPaymentPartial(ctx context.Context, customerID, orderID uuid.UUID, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditOrderRejected(ctx context.Context, customerID, orderID uuid.UUID, amountBs, refundablePMBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	DebitForOrder(ctx context.Context, customerID, orderID uuid.UUID, amountBs float64) error
	RefundBalancePayment(ctx context.Context, customerID, orderID uuid.UUID, amountBs float64) error
}

// StaffReviewHub manages staff WebSocket connections for payment reviews.
type StaffReviewHub interface {
	Register(ctx context.Context, staffID uuid.UUID, conn ManagerConn) error
	Unregister(staffID uuid.UUID, conn ManagerConn)
	BroadcastReview(review models.PaymentReviewBroadcast) error
}

type PaymentReviewsApp interface {
	CreateForOrder(ctx context.Context, order *models.Order, proofURL string, expectedBs, bcvRate float64) (*models.PaymentReview, error)
	// Approve / ApproveOverpay / RejectUnderpaid requieren referenceLast4 (4 dígitos).
	// Si ya existe en otra review consumida, auto-rechazan como rejected_invalid
	// y el resultado trae Action=reject_invalid + DuplicateReject=true.
	Approve(ctx context.Context, reviewID, staffID uuid.UUID, referenceLast4 string) (models.StaffReviewResult, error)
	ApproveOverpay(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error)
	// RejectUnderpaid rechaza la orden por pago incompleto: acredita en Bs lo
	// efectivamente pagado (observedBs) al balance del cliente y cancela la
	// orden de inmediato. No hay espera ni comprobante adicional; si el
	// cliente quiere reintentar, arma una nueva orden (puede pagar con su
	// balance + un nuevo comprobante, eso lo decide el front).
	RejectUnderpaid(ctx context.Context, reviewID, staffID uuid.UUID, observedBs float64, referenceLast4 string) (models.StaffReviewResult, error)
	RejectInvalid(ctx context.Context, reviewID, staffID uuid.UUID, reason string) (models.Order, error)
	ListActiveForStaff(ctx context.Context) ([]models.PaymentReviewBroadcast, error)
	BuildBroadcast(ctx context.Context, review *models.PaymentReview, order *models.Order) models.PaymentReviewBroadcast
	// NotifyPendingPaymentProofs republica payment_proof_pending cada
	// PaymentProofAlertInterval mientras status sea pending_review.
	NotifyPendingPaymentProofs(ctx context.Context) error
}
