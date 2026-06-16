package models

type OrderStatus string

const (
	StatusAccepted            OrderStatus = "Accepted"
	StatusWaitingPaymentProof OrderStatus = "Waiting For Payment Proof"
	StatusInPreparation       OrderStatus = "In Preparation"
	StatusReady               OrderStatus = "Ready"
	StatusOnTheWay            OrderStatus = "On The Way"
	StatusCompleted           OrderStatus = "Completed"
	StatusCancelled           OrderStatus = "Cancelled"
	StatusRejected            OrderStatus = "Rejected"
)
