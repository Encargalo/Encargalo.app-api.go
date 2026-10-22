package models

import "github.com/redis/go-redis/v9"

type DataEventOrderNoRider struct {
	MessageType string
	OrderID     string
}

func (d *DataEventOrderNoRider) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.OrderID = asString(msg.Values["order_id"])
}
