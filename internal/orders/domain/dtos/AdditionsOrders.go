package dtos

import "github.com/google/uuid"

type CreateAdditionsOrders struct {
	AdditionID uuid.UUID `json:"addition_id"`
	Name       string    `json:"name"`
}
