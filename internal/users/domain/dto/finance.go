package dto

type CustomerBalanceResponse struct {
	Balance      float64                   `json:"balance"`
	Transactions []CustomerTransactionItem `json:"transactions"`
}

type CustomerTransactionItem struct {
	ID           string  `json:"id"`
	Amount       float64 `json:"amount"`
	MovementType string  `json:"movement_type"`
	OrderID      *string `json:"order_id,omitempty"`
	CreatedAt    string  `json:"created_at"`
}
