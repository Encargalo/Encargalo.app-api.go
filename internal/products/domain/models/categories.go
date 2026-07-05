package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Categories []Category
type Category struct {
	bun.BaseModel `bun:"table:products.categories" swaggerignore:"true"`

	ID              uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ShopID          uuid.UUID  `bun:"shop_id" json:"shop_id"`
	Name            string     `bun:"name" json:"name"`
	MacroCategoryID *uuid.UUID `bun:"macro_category_id" json:"macro_category_id,omitempty"`

	Items         Items          `bun:"rel:has-many,join:id=category_id" json:"items"`
	MacroCategory *MacroCategory `bun:"rel:belongs-to,join:macro_category_id=id" json:"macro_category,omitempty"`
	Schedules     Schedules      `bun:"rel:has-many,join:id=category_id" json:"schedules,omitempty"`
}

func (c *Category) ToDomainDTO() dtos.CategoryResponse {
	return dtos.CategoryResponse{
		ID:        c.ID,
		ShopID:    c.ShopID,
		Name:      c.Name,
		Items:     c.Items.ToDomainDTO(),
		Schedules: c.Schedules.ToDomainDTO(),
	}
}

func (c *Categories) ToDomainDTO() dtos.CategoriesResponse {

	var categoriesResponse dtos.CategoriesResponse

	for _, v := range *c {
		categoriesResponse.Add(v.ToDomainDTO())
	}

	return categoriesResponse

}
