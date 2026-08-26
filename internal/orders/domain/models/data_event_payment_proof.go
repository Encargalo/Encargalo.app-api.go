package models

type DataEventPaymentProofPending struct {
	MessageType      string  `json:"type"`
	OrderID          string  `json:"order_id"`
	ReviewID         string  `json:"review_id"`
	ShopID           string  `json:"shop_id"`
	CustomerID       string  `json:"customer_id"`
	ExpectedAmountBs float64 `json:"expected_amount_bs"`
	ProofURL         string  `json:"proof_url"`
	TotalUSD         float64 `json:"total_usd"`
	MinutesPending   int     `json:"minutes_pending"`
}

func (d *DataEventPaymentProofPending) BuildFrom(order Order, review PaymentReview, minutesPending int) {
	d.MessageType = "payment_proof_pending"
	d.OrderID = order.ID.String()
	d.ReviewID = review.ID.String()
	d.ShopID = order.ShopID.String()
	d.CustomerID = order.CustomerID.String()
	d.ExpectedAmountBs = review.ExpectedAmountBs
	d.ProofURL = review.ProofURL
	d.TotalUSD = order.TotalPrice
	d.MinutesPending = minutesPending
}
