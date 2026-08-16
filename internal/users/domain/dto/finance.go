package dto

type CustomerBalanceResponse struct {
	Balance      float64                   `json:"balance"`
	Transactions []CustomerTransactionItem `json:"transactions"`
}

type CustomerTransactionItem struct {
	ID                  string   `json:"id"`
	Amount              float64  `json:"amount"`
	MovementType        string   `json:"movement_type"`
	OrderID             *string  `json:"order_id,omitempty"`
	CreatedAt           string   `json:"created_at"`
	RefundablePMBs      *float64 `json:"refundable_pm_bs,omitempty"`
	CanRequestRefund    bool     `json:"can_request_refund,omitempty"`
	RefundEligibleUntil *string  `json:"refund_eligible_until,omitempty"`
	RefundRequestID     *string  `json:"refund_request_id,omitempty"`
}

type UpsertPagoMovilRequest struct {
	Phone      string `json:"phone" validate:"required"`
	BankCode   string `json:"bank_code" validate:"required"`
	NationalID string `json:"national_id" validate:"required"`
}

type PagoMovilResponse struct {
	Phone      string `json:"phone"`
	BankCode   string `json:"bank_code"`
	NationalID string `json:"national_id"`
	UpdatedAt  string `json:"updated_at"`
}

type CreateRefundRequest struct {
	TransactionID string `json:"transaction_id" validate:"required,uuid"`
}

type CompleteRefundRequest struct {
	AmountBs   float64 `json:"amount_bs" validate:"required,gt=0"`
	ProofImage string  `json:"proof_image" validate:"required"`
}

type RefundRequestResponse struct {
	ID              string  `json:"id"`
	TransactionID   string  `json:"transaction_id"`
	OrderID         *string `json:"order_id,omitempty"`
	AmountBs        float64 `json:"amount_bs"`
	Status          string  `json:"status"`
	PMPhone         string  `json:"pm_phone"`
	PMBankCode      string  `json:"pm_bank_code"`
	PMNationalID    string  `json:"pm_national_id"`
	ProofURL        *string `json:"proof_url,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
	AcceptedAt      *string `json:"accepted_at,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
	ResolvedAt      *string `json:"resolved_at,omitempty"`
}

type RefundRequestBroadcast struct {
	Type          string  `json:"type"`
	ID            string  `json:"id"`
	CustomerID    string  `json:"customer_id"`
	TransactionID string  `json:"transaction_id"`
	OrderID       *string `json:"order_id,omitempty"`
	AmountBs      float64 `json:"amount_bs"`
	// AmountUSD/BcvRate: equivalente informativo, ver RefundRequestResponse.
	AmountUSD       float64 `json:"amount_usd,omitempty"`
	BcvRate         float64 `json:"bcv_rate,omitempty"`
	Status          string  `json:"status"`
	PMPhone         string  `json:"pm_phone"`
	PMBankCode      string  `json:"pm_bank_code"`
	PMNationalID    string  `json:"pm_national_id"`
	ProofURL        *string `json:"proof_url,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
}
