package models

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type DataProduceOrder struct {
	bun.BaseModel `bun:"table:products.items" swaggerignore:"true"`

	MessageType string    `bun:"type" json:"type"`
	ProductID   uuid.UUID `bun:"product_id" json:"product_id"`
	Quantity    int       `bun:"quantity" json:"quantity"`
}

func (d *DataProduceOrder) BuildModel(message redis.XMessage) error {
	d.MessageType = message.Values["type"].(string)
	d.ProductID = uuid.MustParse(message.Values["product_id"].(string))
	q, _ := strconv.Atoi(message.Values["quantity"].(string))
	d.Quantity = q
	return nil
}
