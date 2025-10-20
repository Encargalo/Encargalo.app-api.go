package dtos

import "github.com/google/uuid"

type CategoriesResponse []CategoryResponse

type CategoryResponse struct {
	ID     uuid.UUID `json:"id"`
	ShopID uuid.UUID `json:"shop_id"`
	Name   string    `json:"name"`

	Items ItemsResponse `json:"items"`
}

func (c *CategoriesResponse) Add(category CategoryResponse) {
	*c = append(*c, category)
}
