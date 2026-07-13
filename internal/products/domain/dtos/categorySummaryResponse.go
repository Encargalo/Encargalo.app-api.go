package dtos

import "github.com/google/uuid"

type CategorySummariesResponse []CategorySummaryResponse

type CategorySummaryResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (c *CategorySummariesResponse) Add(category CategorySummaryResponse) {
	*c = append(*c, category)
}
