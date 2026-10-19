package models

type Role string

const (
	Customer     Role = "Customer"
	OrderManager Role = "Order Manager"
	Owner        Role = "Owner"
	Rider        Role = "Rider"
	Staff        Role = "Staff"
)

// RiderPhoneNext es el paso que la app de riders muestra después del teléfono.
type RiderPhoneNext string

const (
	RiderPhoneNextSignIn     RiderPhoneNext = "sign_in"
	RiderPhoneNextOnboarding RiderPhoneNext = "onboarding"
	RiderPhoneNextRegister   RiderPhoneNext = "register"
)

type ContextKey string

const SessionKey ContextKey = "session"
