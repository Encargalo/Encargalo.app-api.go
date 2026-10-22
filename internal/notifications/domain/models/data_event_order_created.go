package models

import "github.com/redis/go-redis/v9"

type DataEventOrderCreated struct {
	MessageType string `json:"type"`
	OrderID     string `json:"order_id"`
	ShopID      string `json:"shop_id"`
	CustomerID  string `json:"customer_id"`
	Total       string `json:"total"`
}

func (data *DataEventOrderCreated) BuidlToModelEvent(message redis.XMessage) {
	data.MessageType = message.Values["type"].(string)
	data.OrderID = message.Values["order_id"].(string)
	data.ShopID = message.Values["shop_id"].(string)
	data.CustomerID = message.Values["customer_id"].(string)
	data.Total = message.Values["total"].(string)
}
