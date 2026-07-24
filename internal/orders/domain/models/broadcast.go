package models

type OrderBroadcast struct {
	Type       string   `json:"type"`
	Order      Order    `json:"order"`
	SaleAmount *float64 `json:"sale_amount,omitempty"`
	Commission *float64 `json:"commission,omitempty"`
	NetAmount  *float64 `json:"net_amount,omitempty"`
}
