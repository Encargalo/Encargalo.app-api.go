package dtos

import "github.com/google/uuid"

type MacroCategoriesResponse []MacroCategoryResponse

type MacroCategoryResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

func (m *MacroCategoriesResponse) Add(macroCategory MacroCategoryResponse) {
	*m = append(*m, macroCategory)
}
