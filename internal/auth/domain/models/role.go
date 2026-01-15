package models

type Role string

const (
	Customer     Role = "Customer"
	OrderManager Role = "Order Manager"
	Owner        Role = "Owner"
)
