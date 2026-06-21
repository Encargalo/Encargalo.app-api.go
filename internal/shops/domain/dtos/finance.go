package dtos

type ShopBalanceResponse struct {
	Balance      float64              `json:"balance"`
	Transactions []ShopTransactionDTO `json:"transactions"`
}

type ShopTransactionDTO struct {
	ID            string  `json:"id"`
	OrderID       *string `json:"order_id,omitempty"`
	Amount        float64 `json:"amount"`
	MovementType  string  `json:"movement_type"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type ShopTransactionsResponse struct {
	Transactions []ShopTransactionDTO `json:"transactions"`
	Total        int                  `json:"total"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
}
