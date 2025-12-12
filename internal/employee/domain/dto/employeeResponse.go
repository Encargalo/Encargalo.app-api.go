package dto

import "github.com/google/uuid"

type EmployeeResponse struct {
	ID     uuid.UUID    `json:"id"`
	ShopID uuid.UUID    `json:"shop_id"`
	Name   string       `json:"name"`
	Email  string       `json:"email"`
	Phone  string       `json:"phone"`
	Role   string       `json:"role"`
	Shop   ShopResponse `json:"shop"`
}

type ShopResponse struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Opened bool      `json:"opened"`
}
