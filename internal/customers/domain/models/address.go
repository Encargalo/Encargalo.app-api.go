package models

import (
	"time"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Addresses []Address

type Address struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	CustomerID   string             `bson:"customer_id"`
	Alias        string             `bson:"alias"`
	Address      string             `bson:"address"`
	Reference    string             `bson:"reference"`
	Latitude     float64            `bson:"latitude"`
	Longitude    float64            `bson:"longitude"`
	Type         string             `bson:"type"`
	Instructions string             `bson:"instructions,omitempty"`
	Extra        interface{}        `bson:"extra,omitempty"`
	CreatedAt    time.Time          `bson:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at"`
}

func (a *Address) BuildToModel(customerID uuid.UUID, dtoAddr dto.Address) {
	a.ID = primitive.NewObjectID()
	a.CustomerID = customerID.String()
	a.Alias = dtoAddr.Label
	a.Address = dtoAddr.Address
	a.Reference = dtoAddr.Instructions
	a.Latitude = dtoAddr.Latitude
	a.Longitude = dtoAddr.Longitude
	a.Type = dtoAddr.Type
	a.Instructions = dtoAddr.Instructions
	a.Extra = dtoAddr.Extra
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
}
