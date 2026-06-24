package models

type OrderStatus string

const (
	StatusPending       OrderStatus = "Pending"
	StatusAccepted      OrderStatus = "Accepted"
	StatusInPreparation OrderStatus = "In Preparation"
	StatusReady         OrderStatus = "Ready"
	StatusOnTheWay      OrderStatus = "On The Way"
	StatusCompleted     OrderStatus = "Completed"
	StatusCancelled     OrderStatus = "Cancelled"
	StatusRejected      OrderStatus = "Rejected"
)
