package dtos

import "Encargalo.app-api.go/internal/riders/domain/models"

type BalanceResponse struct {
	BalanceBs  float64     `json:"balance_bs"`
	BalanceUSD float64     `json:"balance_usd"`
	BcvRate    float64     `json:"bcv_rate,omitempty"`
	Zone       models.Zone `json:"zone"`
	// WithdrawalMinBs es el mínimo de retiro vigente, ya convertido a Bs con
	// la tasa BCV actual (equivalente dinámico a WithdrawalMinUSD).
	WithdrawalMinBs float64 `json:"withdrawal_min_bs"`
	// ActiveWithdrawal distingue un balance en 0 por retiro en curso de un
	// balance en 0 por no haber ganado nada.
	ActiveWithdrawal *ActiveWithdrawalDTO `json:"active_withdrawal"`
	Transactions     []TransactionDTO     `json:"transactions"`
}

type ActiveWithdrawalDTO struct {
	ID        string  `json:"id"`
	AmountBs  float64 `json:"amount_bs"`
	AmountUSD float64 `json:"amount_usd"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

type TransactionDTO struct {
	ID            string   `json:"id"`
	OrderID       *string  `json:"order_id,omitempty"`
	TripID        *string  `json:"trip_id,omitempty"`
	AmountBs      float64  `json:"amount_bs"`
	AmountUSD     float64  `json:"amount_usd"`
	BcvRate       *float64 `json:"bcv_rate,omitempty"`
	MovementType  string   `json:"movement_type"`
	ServiceType   *string  `json:"service_type,omitempty"`
	PaymentMethod *string  `json:"payment_method,omitempty"`
	DistanceKm    *float64 `json:"distance_km,omitempty"`
	StaffID       *string  `json:"staff_id,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	ProofURL      *string  `json:"proof_url,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

type TransactionsResponse struct {
	Transactions []TransactionDTO `json:"transactions"`
	Total        int              `json:"total"`
	Page         int              `json:"page"`
	Limit        int              `json:"limit"`
}

// WithdrawalResponse es la solicitud creada, no un pago. El monto es lo que
// la plataforma debe al momento de solicitar; lo que se transfiera se congela
// recién cuando staff reclama la solicitud.
type WithdrawalResponse struct {
	ID        string  `json:"id"`
	AmountBs  float64 `json:"amount_bs"`
	AmountUSD float64 `json:"amount_usd"`
	BcvRate   float64 `json:"bcv_rate,omitempty"`
	Status    string  `json:"status"`
	PMPhone   string  `json:"pm_phone"`
	CreatedAt string  `json:"created_at"`
}
