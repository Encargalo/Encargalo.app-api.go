package models

import (
	"time"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Addresses []Address

type Address struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CustomerID   string             `bson:"customer_id" json:"customer_id"`
	Label        string             `bson:"alias" json:"alias"`
	Address      string             `bson:"address" json:"address"`
	Latitude     float64            `bson:"latitude" json:"latitude"`
	Longitude    float64            `bson:"longitude" json:"longitude"`
	Type         string             `bson:"type" json:"type"`
	Instructions string             `bson:"instructions,omitempty" json:"instructions,omitempty"`
	Extra        interface{}        `bson:"extra,omitempty" json:"extra,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

func (a *Address) BuildToModel(customerID uuid.UUID, dtoAddr dto.Address) {
	a.ID = primitive.NewObjectID()
	a.CustomerID = customerID.String()
	a.Label = dtoAddr.Label
	a.Address = dtoAddr.Address
	a.Latitude = dtoAddr.Latitude
	a.Longitude = dtoAddr.Longitude
	a.Instructions = dtoAddr.Instructions
	a.Type = dtoAddr.Type
	a.Extra = dtoAddr.Extra
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
}
