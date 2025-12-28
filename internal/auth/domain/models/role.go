package models

type Role string

const (
	Customer     Role = "customer"
	OrderManager Role = "order_manager"
	Admin        Role = "admin"
)
