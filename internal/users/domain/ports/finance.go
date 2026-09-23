package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type RefundWSConn interface {
	Send(msg []byte)
	Close()
}

// StreamPublisher publica eventos al Redis Stream (mismo contrato que orders.RedisStream).
type StreamPublisher interface {
	PublisherEvent(ctx context.Context, data any) error
}

type CustomerFinanceRepo interface {
	GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.Balance, error)
	// SumBalancePaymentDebit returns the absolute Bs debited via balance_payment for a reference.
	SumBalancePaymentDebit(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string) (float64, error)
	RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.Transaction) (float64, error)
	ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error)
	GetTransaction(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	SetTransactionRefundRequestID(ctx context.Context, txID, refundRequestID uuid.UUID) error

	GetPagoMovil(ctx context.Context, customerID uuid.UUID) (*models.CustomerPagoMovil, error)
	UpsertPagoMovil(ctx context.Context, pm *models.CustomerPagoMovil) error

	CreateRefundRequest(ctx context.Context, req *models.BalanceRefundRequest) error
	GetRefundRequest(ctx context.Context, id uuid.UUID) (*models.BalanceRefundRequest, error)
	ListRefundRequestsByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.BalanceRefundRequest, error)
	ListActiveRefundRequests(ctx context.Context) ([]models.BalanceRefundRequest, error)
	HasActiveRefundForTransaction(ctx context.Context, transactionID uuid.UUID) (bool, error)
	// CreateRefundRequestWithDebit crea la solicitud pending y debita el balance
	// (movement balance_refund_hold) en la misma transacción.
	CreateRefundRequestWithDebit(ctx context.Context, req *models.BalanceRefundRequest) error
	// AcceptPending: pending→accepted y hold→processing (mismo movimiento; sin cambio de saldo).
	AcceptPending(ctx context.Context, id, staffID uuid.UUID) (bool, error)
	// RejectPendingWithRestore: pending→rejected y abona de nuevo el monto (balance_refund_restore).
	RejectPendingWithRestore(ctx context.Context, id, staffID uuid.UUID, reason string) (bool, error)
	// CompleteAccepted: accepted→completed con proof y processing→payout (mismo movimiento).
	CompleteAccepted(ctx context.Context, id uuid.UUID, proofURL string) (bool, error)

	// ListIncompleteForAlert returns pending/accepted requests due for a Discord alert.
	ListIncompleteForAlert(ctx context.Context, alertBefore time.Time) ([]models.BalanceRefundRequest, error)
	// MarkAlerted CAS-updates last_alerted_at if still incomplete and due.
	MarkAlerted(ctx context.Context, id uuid.UUID, alertedAt, alertBefore time.Time) (bool, error)
}

type StaffRefundHub interface {
	Register(ctx context.Context, staffID uuid.UUID, conn RefundWSConn) error
	Unregister(staffID uuid.UUID, conn RefundWSConn)
	Broadcast(req dto.RefundRequestBroadcast) error
}

// CustomerNotifier empuja mensajes al WS del cliente (orders/manager).
// Lo implementa AppManagerOrder; se inyecta con setter para evitar ciclo DI.
type CustomerNotifier interface {
	NotifyCustomer(customerID uuid.UUID, msg []byte) error
}

type CustomerFinanceApp interface {
	GetBalance(ctx context.Context, customerID uuid.UUID) (dto.CustomerBalanceResponse, error)
	GetBalanceBs(ctx context.Context, customerID uuid.UUID) (float64, error)
	// GetBalancePaymentDebitForReference returns Bs taken from balance for this reference (0 if none).
	GetBalancePaymentDebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string) (float64, error)
	// Credit* reciben montos ya en Bs (moneda nativa del ledger) y la tasa
	// BCV congelada del quote/review para grabarla en el movimiento (umbral
	// de devolución y proyección USD informativa). El caller resuelve la tasa.
	CreditPaymentOverage(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditPaymentPartial(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	// CreditOrderRejected acredita amountBs al balance; refundablePMBs es la
	// porción cash-refundable (Pago Móvil). bcvRate se congela en el movimiento.
	CreditOrderRejected(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, refundablePMBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	// DebitForReference/RefundBalancePayment operan directo en Bs: es lo que el
	// customer pidió usar de su balance, ya en su moneda nativa.
	DebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error
	RefundBalancePayment(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error

	GetPagoMovil(ctx context.Context, customerID uuid.UUID) (dto.PagoMovilResponse, error)
	UpsertPagoMovil(ctx context.Context, customerID uuid.UUID, req dto.UpsertPagoMovilRequest) (dto.PagoMovilResponse, error)

	CreateRefundRequest(ctx context.Context, customerID uuid.UUID, transactionID uuid.UUID) (dto.RefundRequestResponse, error)
	ListRefundRequests(ctx context.Context, customerID uuid.UUID) ([]dto.RefundRequestResponse, error)
	GetRefundRequest(ctx context.Context, customerID, requestID uuid.UUID) (dto.RefundRequestResponse, error)

	ListActiveForStaff(ctx context.Context) ([]dto.RefundRequestBroadcast, error)
	AcceptRefund(ctx context.Context, requestID, staffID uuid.UUID) (dto.RefundRequestResponse, error)
	RejectRefund(ctx context.Context, requestID, staffID uuid.UUID, reason string) (dto.RefundRequestResponse, error)
	CompleteRefund(ctx context.Context, requestID uuid.UUID, proofImage string, amountBs float64) (dto.RefundRequestResponse, error)

	// SetCustomerNotifier cablea el push al WS del cliente (evita ciclo DI con orders).
	SetCustomerNotifier(n CustomerNotifier)

	// NotifyIncompleteRefunds publica balance_refund_pending cada 5 min mientras no esté completed.
	NotifyIncompleteRefunds(ctx context.Context) error
}
