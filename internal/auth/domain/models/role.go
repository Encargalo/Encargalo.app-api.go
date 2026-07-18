package models

type Role string

const (
	Customer     Role = "Customer"
	OrderManager Role = "Order Manager"
	Owner        Role = "Owner"
	Rider        Role = "Rider"
	Staff        Role = "Staff"
)

type ContextKey string

const SessionKey ContextKey = "session"
