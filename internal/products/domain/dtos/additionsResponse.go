package dtos

import "github.com/google/uuid"

type AdditionsResponse []AdditionResponse

type AdditionResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Ref  float64   `json:"ref"`
	Bs   float64   `json:"bs"`
}

func (a *AdditionsResponse) Add(addition AdditionResponse) {
	*a = append(*a, addition)
}
