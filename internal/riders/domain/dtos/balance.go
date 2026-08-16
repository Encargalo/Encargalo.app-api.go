package dtos

import "Encargalo.app-api.go/internal/riders/domain/models"

type BalanceResponse struct {
	BalanceBs  float64     `json:"balance_bs"`
	BalanceUSD float64     `json:"balance_usd"`
	BcvRate    float64     `json:"bcv_rate,omitempty"`
	Zone       models.Zone `json:"zone"`
	// WithdrawalMinBs es el mínimo de retiro vigente, ya convertido a Bs con
	// la tasa BCV actual (equivalente dinámico a WithdrawalMinUSD).
	WithdrawalMinBs float64          `json:"withdrawal_min_bs"`
	Transactions    []TransactionDTO `json:"transactions"`
}

type TransactionDTO struct {
	ID            string   `json:"id"`
	OrderID       *string  `json:"order_id,omitempty"`
	AmountBs      float64  `json:"amount_bs"`
	AmountUSD     float64  `json:"amount_usd"`
	BcvRate       *float64 `json:"bcv_rate,omitempty"`
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
