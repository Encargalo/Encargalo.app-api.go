package dtos

type PayableItem struct {
	EntityID   string  `json:"entity_id"`
	Name       string  `json:"name"`
	SurName    string  `json:"sur_name,omitempty"`
	Phone      string  `json:"phone,omitempty"`
	BalanceBs  float64 `json:"balance_bs"`
	BalanceUSD float64 `json:"balance_usd"`
	BcvRate    float64 `json:"bcv_rate,omitempty"`
	UpdatedAt  string  `json:"updated_at"`
}

type PayablesResponse struct {
	Items []PayableItem `json:"items"`
	Total int           `json:"total"`
}

type PayoutRequest struct {
	AmountBs   float64 `json:"amount_bs" validate:"required,gt=0"`
	Notes      string  `json:"notes"`
	ProofImage string  `json:"proof_image" validate:"required"`
}

type AdjustmentRequest struct {
	Direction string  `json:"direction" validate:"required,oneof=credit debit"`
	AmountBs  float64 `json:"amount_bs" validate:"required,gt=0"`
	Notes     string  `json:"notes" validate:"required"`
}

type ActionResponse struct {
	TransactionID string  `json:"transaction_id"`
	AmountBs      float64 `json:"amount_bs"`
	AmountUSD     float64 `json:"amount_usd"`
	BcvRate       float64 `json:"bcv_rate,omitempty"`
	BalanceBs     float64 `json:"balance_bs"`
	BalanceUSD    float64 `json:"balance_usd"`
	ProofURL      *string `json:"proof_url,omitempty"`
	Notes         string  `json:"notes,omitempty"`
	StaffID       string  `json:"staff_id"`
	CreatedAt     string  `json:"created_at"`
}
