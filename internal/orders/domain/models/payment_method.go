package models

type PaymentMethod string

const (
	MethodCashBs    PaymentMethod = "EfectivoBs"
	MethodCashUSD   PaymentMethod = "EfectivoDolar"
	MethodPagoMovil PaymentMethod = "PagoMovil"
)
