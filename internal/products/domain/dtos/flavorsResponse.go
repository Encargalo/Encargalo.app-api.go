package dtos

import "github.com/google/uuid"

type FlavorsResponse []FlavorResponse

type FlavorResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (f *FlavorsResponse) Add(flavor FlavorResponse) {
	*f = append(*f, flavor)
}
