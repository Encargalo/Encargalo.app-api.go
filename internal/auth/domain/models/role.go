package models

type Role string

const (
	Customer          Role = "Customer"
	CustomerMinuscula Role = "customer"
	OrderManager      Role = "Order Manager"
	Owner             Role = "Owner"
)
