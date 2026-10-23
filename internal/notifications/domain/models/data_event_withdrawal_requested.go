package models

import (
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type DataEventWithdrawalRequested struct {
	MessageType string
	RequestID   string
}

func (d *DataEventWithdrawalRequested) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.RequestID = asString(msg.Values["request_id"])
}

// WithdrawalRequest es el read model de la solicitud: el consumer consulta la
// OLTP directamente en vez de confiar solo en el payload del evento.
type WithdrawalRequest struct {
	bun.BaseModel `bun:"table:finance.withdrawal_requests"`

	ID         uuid.UUID `bun:"id,pk"`
	EntityID   uuid.UUID `bun:"entity_id"`
	EntityType string    `bun:"entity_type"`
	Amount     float64   `bun:"amount"`
	Status     string    `bun:"status"`
	PMPhone    *string   `bun:"pm_phone"`
	PMBankCode *string   `bun:"pm_bank_code"`

	Rider Customer `bun:"rel:belongs-to,join:entity_id=id"`
}
