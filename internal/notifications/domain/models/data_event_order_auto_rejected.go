package models

import "github.com/redis/go-redis/v9"

type DataEventOrderAutoRejected struct {
	MessageType    string
	OrderID        string
	ShopID         string
	CustomerID     string
	MinutesPending int
}

func (d *DataEventOrderAutoRejected) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.OrderID = asString(msg.Values["order_id"])
	d.ShopID = asString(msg.Values["shop_id"])
	d.CustomerID = asString(msg.Values["customer_id"])
	d.MinutesPending = asInt(msg.Values["minutes_pending"])
}
