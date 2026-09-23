package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CustomerBalanceCreditor credits and debits the customer's Bs balance.
// Implemented by users/app.customerFinanceApp and adapted in DI.
// Credit* receive the amount already in Bs plus the frozen BCV rate.
// Debit and Refund operate directly in Bs.
type CustomerBalanceCreditor interface {
	GetBalanceBs(ctx context.Context, customerID uuid.UUID) (float64, error)
	GetBalancePaymentDebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string) (float64, error)
	CreditPaymentOverage(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditPaymentPartial(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	CreditOrderRejected(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, refundablePMBs, bcvRate float64) (txID uuid.UUID, createdAt time.Time, err error)
	DebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error
	RefundBalancePayment(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error
}
