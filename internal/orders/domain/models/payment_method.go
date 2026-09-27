package models

type PaymentMethod string

const (
	MethodPagoMovil PaymentMethod = "PagoMovil"
	MethodEfectivo  PaymentMethod = "Efectivo"
)
