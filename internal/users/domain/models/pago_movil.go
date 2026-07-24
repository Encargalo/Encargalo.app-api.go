package models

import (
	"time"

	"github.com/google/uuid"
)

// CustomerPagoMovil es el Pago Móvil destino del cliente (almacenado en
// users.accounts.pago_movil como JSONB).
type CustomerPagoMovil struct {
	CustomerID uuid.UUID `json:"-"`
	Phone      string    `json:"phone"`
	BankCode   string    `json:"bank_code"`
	NationalID string    `json:"national_id"`
	// UpdatedAt viene de users.accounts.updated_at (no va en el JSONB).
	UpdatedAt time.Time `json:"-"`
}
