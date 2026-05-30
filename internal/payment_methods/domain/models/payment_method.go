package models

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PaymentMethod struct {
	bun.BaseModel `bun:"table:orders.payment_methods"`

	ID          uuid.UUID       `bun:"id,pk,type:uuid"    json:"id"`
	Type        string          `bun:"type"               json:"type"`
	Name        string          `bun:"name"               json:"name"`
	Description string          `bun:"description"        json:"description"`
	Fields      json.RawMessage `bun:"fields,type:jsonb"  json:"fields,omitempty"`
	QRUrl       *string         `bun:"qr_url"             json:"qr_url,omitempty"`
	IsActive    bool            `bun:"is_active"          json:"is_active"`
}
