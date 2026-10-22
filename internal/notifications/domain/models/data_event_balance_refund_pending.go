package models

import (
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

type DataEventBalanceRefundPending struct {
	MessageType    string
	RequestID      string
	MinutesPending int
}

func (d *DataEventBalanceRefundPending) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.RequestID = asString(msg.Values["request_id"])
	d.MinutesPending = asInt(msg.Values["minutes_pending"])
}

type BalanceRefundRequest struct {
	bun.BaseModel `bun:"table:finance.balance_refund_requests"`

	ID         uuid.UUID  `bun:"id,pk"`
	CustomerID uuid.UUID  `bun:"customer_id"`
	OrderID    *uuid.UUID `bun:"order_id"`
	AmountBs   float64    `bun:"amount_bs"`
	Status     string     `bun:"status"`

	Customer Customer `bun:"rel:belongs-to,join:customer_id=id"`
}

type OrderStatusRow struct {
	bun.BaseModel `bun:"table:orders.orders"`

	ID     uuid.UUID `bun:"id,pk"`
	Status string    `bun:"status"`
}
