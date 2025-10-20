package dtos

import "github.com/google/uuid"

type AdditionsResponse []AdditionResponse

type AdditionResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Price int       `json:"price"`
}

func (a *AdditionsResponse) Add(addition AdditionResponse) {
	*a = append(*a, addition)
}
