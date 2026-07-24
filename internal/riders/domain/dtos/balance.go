package dtos

import "Encargalo.app-api.go/internal/riders/domain/models"

type BalanceResponse struct {
	Balance      float64          `json:"balance"`
	Zone         models.Zone      `json:"zone"`
	Transactions []TransactionDTO `json:"transactions"`
}

type TransactionDTO struct {
	ID            string   `json:"id"`
	OrderID       *string  `json:"order_id,omitempty"`
	Amount        float64  `json:"amount"`
	MovementType  string   `json:"movement_type"`
	PaymentMethod *string  `json:"payment_method,omitempty"`
	DistanceKm    *float64 `json:"distance_km,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

type TransactionsResponse struct {
	Transactions []TransactionDTO `json:"transactions"`
	Total        int              `json:"total"`
	Page         int              `json:"page"`
	Limit        int              `json:"limit"`
}

type WithdrawalResponse struct {
	AmountWithdrawn float64 `json:"amount_withdrawn"`
}
