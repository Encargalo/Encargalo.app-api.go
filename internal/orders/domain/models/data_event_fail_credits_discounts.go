package models

import (
	"fmt"
)

type EventFailCreditDiscount struct {
	MessageType string `json:"type"`
	StatusCode  string `json:"status_code"`
	Message     string `json:"message"`
	ShopID      string `json:"shop_id"`
	Value       string `json:"value"`
}

func (f *EventFailCreditDiscount) BuildFailCreditDiscount(statusCode, message, shopID string, value float64) {
	f.MessageType = "Fail Post Credits"
	f.StatusCode = statusCode
	f.Message = message
	f.ShopID = shopID
	f.Value = fmt.Sprintf("%f", value)
}
