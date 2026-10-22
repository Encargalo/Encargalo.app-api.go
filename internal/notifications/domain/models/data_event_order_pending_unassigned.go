package models

import (
	"strconv"

	"github.com/redis/go-redis/v9"
)

type DataEventOrderPendingUnassigned struct {
	MessageType    string `json:"type"`
	OrderID        string `json:"order_id"`
	ShopID         string `json:"shop_id"`
	CustomerID     string `json:"customer_id"`
	MinutesPending int    `json:"minutes_pending"`
}

func (d *DataEventOrderPendingUnassigned) BuildFromXMessage(msg redis.XMessage) {
	d.MessageType = asString(msg.Values["type"])
	d.OrderID = asString(msg.Values["order_id"])
	d.ShopID = asString(msg.Values["shop_id"])
	d.CustomerID = asString(msg.Values["customer_id"])
	d.MinutesPending = asInt(msg.Values["minutes_pending"])
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func asInt(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case string:
		n, _ := strconv.Atoi(t)
		return n
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}
