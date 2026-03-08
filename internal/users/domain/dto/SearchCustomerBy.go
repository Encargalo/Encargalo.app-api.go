package dto

import "github.com/google/uuid"

type SearchCustomerBy struct {
	ID    uuid.UUID `query:"customer_id"`
	Phone string    `query:"phone"`
}
