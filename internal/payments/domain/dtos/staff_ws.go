package dtos

import "github.com/google/uuid"

// StaffWSMessage son los mensajes entrantes del WS de staff.
type StaffWSMessage struct {
	Type             string    `json:"type"`
	ReviewID         uuid.UUID `json:"review_id"`
	ObservedAmountBs *float64  `json:"observed_amount_bs,omitempty"`
	ReferenceLast4   string    `json:"reference_last4,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}
