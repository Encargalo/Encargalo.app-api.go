package models

import "time"

type OrderStatus string

const (
	StatusPaymentReview OrderStatus = "Payment Review"
	StatusPending       OrderStatus = "Pending"
	StatusAccepted      OrderStatus = "Accepted"
	StatusInPreparation OrderStatus = "In Preparation"
	StatusReady         OrderStatus = "Ready"
	StatusOnTheWay      OrderStatus = "On The Way"
	StatusCompleted     OrderStatus = "Completed"
	StatusCancelled     OrderStatus = "Cancelled"
	StatusRejected      OrderStatus = "Rejected"
)

// PendingAcceptTimeout: tiempo máximo en Pending sin que el local acepte.
// Al cumplirse, auto-rechazo (mismo flujo que rechazo del local).
const PendingAcceptTimeout = 10 * time.Minute

// PendingUnassignedAlertMinutes: minutos (desde pending_at) en los que se
// publica order_pending_unassigned (aviso Discord). Cancelación en el 10.
var PendingUnassignedAlertMinutes = []int{4, 8}

// PendingUnassignedFirstAlertMinute / Max: para listados e índices.
const (
	PendingUnassignedFirstAlertMinute = 4
	PendingUnassignedMaxAlertMinute   = 8
)
