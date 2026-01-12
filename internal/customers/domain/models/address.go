package models

import (
	"encoding/json"
	"fmt"
	"time"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Addresses []Address

type Address struct {
	bun.BaseModel `bun:"table:customers.address"`

	ID           uuid.UUID       `bun:"id,pk,type:uuid,default:gen_random_uuid()" json:"id"`
	CustomerID   uuid.UUID       `bun:"customer_id,type:uuid,notnull" json:"customer_id"`
	Label        string          `bun:"alias,notnull" json:"alias"`
	Address      string          `bun:"address,notnull" json:"address"`
	Latitude     float64         `bun:"latitude,notnull" json:"latitude"`
	Longitude    float64         `bun:"longitude,notnull" json:"longitude"`
	Type         string          `bun:"type,notnull" json:"type"`
	Instructions *string         `bun:"instructions,nullzero" json:"instructions,omitempty"`
	Extra        json.RawMessage `bun:"extra,type:jsonb,nullzero" json:"extra,omitempty"`
	CreatedAt    time.Time       `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt    time.Time       `bun:"updated_at,notnull,default:now()" json:"updated_at"`
}

func (a *Address) BuildToModel(customerID uuid.UUID, dtoAddr dto.Address) error {
	a.CustomerID = customerID
	a.Label = dtoAddr.Label
	a.Address = dtoAddr.Address
	a.Latitude = dtoAddr.Latitude
	a.Longitude = dtoAddr.Longitude
	a.Type = dtoAddr.Type
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()

	if dtoAddr.Instructions != "" {
		a.Instructions = &dtoAddr.Instructions
	}

	if dtoAddr.Extra != nil {
		b, err := json.Marshal(dtoAddr.Extra)
		if err != nil {
			return fmt.Errorf("marshal extra: %w", err)
		}
		a.Extra = json.RawMessage(b)
	}

	return nil
}
