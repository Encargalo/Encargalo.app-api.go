package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PaymentReviewStatus string

const (
	ReviewPendingReview     PaymentReviewStatus = "pending_review"
	ReviewApproved          PaymentReviewStatus = "approved"
	ReviewOverpaid          PaymentReviewStatus = "overpaid"
	ReviewRejectedUnderpaid PaymentReviewStatus = "rejected_underpaid"
	ReviewRejectedInvalid   PaymentReviewStatus = "rejected_invalid"
	// ReviewRefundedShopReject: el local rechazó la orden tras pago aprobado;
	// el total de la orden se acreditó al balance del cliente.
	ReviewRefundedShopReject PaymentReviewStatus = "refunded_shop_reject"
)

type PaymentReview struct {
	bun.BaseModel `bun:"table:orders.payment_reviews"`

	ID               uuid.UUID           `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	OrderID          uuid.UUID           `bun:"order_id,type:uuid,notnull" json:"order_id"`
	ProofURL         string              `bun:"proof_url,notnull" json:"proof_url"`
	ExpectedAmountBs float64             `bun:"expected_amount_bs,notnull" json:"expected_amount_bs"`
	ObservedAmountBs *float64            `bun:"observed_amount_bs" json:"observed_amount_bs,omitempty"`
	DifferenceBs     *float64            `bun:"difference_bs" json:"difference_bs,omitempty"`
	BcvRate          float64             `bun:"bcv_rate,notnull" json:"bcv_rate"`
	Status           PaymentReviewStatus `bun:"status,notnull" json:"status"`
	RejectionReason  *string             `bun:"rejection_reason" json:"rejection_reason,omitempty"`
	StaffID          *uuid.UUID          `bun:"staff_id,type:uuid" json:"staff_id,omitempty"`
	CreditedBs       *float64            `bun:"credited_bs" json:"credited_bs,omitempty"`
	// ReferenceLast4: últimos 4 dígitos del comprobante capturados por staff.
	// Solo se persiste en approve / overpay / underpaid (unicidad parcial en BD).
	ReferenceLast4 *string    `bun:"reference_last4" json:"reference_last4,omitempty"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	ResolvedAt     *time.Time `bun:"resolved_at" json:"resolved_at,omitempty"`
}

// StaffReviewResult es el resultado de una acción del staff sobre un review.
// Action refleja lo efectivamente aplicado (puede ser reject_invalid si hubo
// auto-rechazo por reference_last4 duplicado).
type StaffReviewResult struct {
	Order           Order  `json:"order"`
	Action          string `json:"action"`
	DuplicateReject bool   `json:"duplicate_reject,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}
