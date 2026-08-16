package dtos

type ShopBalanceResponse struct {
	BalanceBs    float64              `json:"balance_bs"`
	BalanceUSD   float64              `json:"balance_usd"`
	BcvRate      float64              `json:"bcv_rate,omitempty"`
	Transactions []ShopTransactionDTO `json:"transactions"`
}

type ShopTransactionDTO struct {
	ID            string   `json:"id"`
	OrderID       *string  `json:"order_id,omitempty"`
	AmountBs      float64  `json:"amount_bs"`
	AmountUSD     float64  `json:"amount_usd"`
	BcvRate       *float64 `json:"bcv_rate,omitempty"`
	MovementType  string   `json:"movement_type"`
	PaymentMethod *string  `json:"payment_method,omitempty"`
	StaffID       *string  `json:"staff_id,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	ProofURL      *string  `json:"proof_url,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

type ShopTransactionsResponse struct {
	Transactions []ShopTransactionDTO `json:"transactions"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
}
