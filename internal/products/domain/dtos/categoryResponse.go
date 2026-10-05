package dtos

import "github.com/google/uuid"

type CategoriesResponse []CategoryResponse

type CategoryResponse struct {
	ID                 uuid.UUID              `json:"id"`
	ShopID             uuid.UUID              `json:"shop_id"`
	Name               string                 `json:"name"`
	MacroCategoryID    *uuid.UUID             `json:"macro_category_id,omitempty"`
	MacroCategory      *MacroCategoryResponse `json:"macro_category,omitempty"`
	AvailabilityStatus string                 `json:"availability_status" example:"active"`
	CanOrder           bool                   `json:"can_order" example:"true"`

	Details   CategoryDetailsResponse `json:"details"`
	Items     ItemsResponse           `json:"items,omitempty"`
	Schedules []ScheduleResponse      `json:"schedules,omitempty"`
}

func (c *CategoriesResponse) Add(category CategoryResponse) {
	*c = append(*c, category)
}
