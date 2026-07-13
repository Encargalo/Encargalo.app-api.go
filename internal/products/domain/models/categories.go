package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Categories []Category
type Category struct {
	bun.BaseModel `bun:"table:products.categories" swaggerignore:"true"`

	ID                 uuid.UUID          `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ShopID             uuid.UUID          `bun:"shop_id" json:"shop_id"`
	Name               string             `bun:"name" json:"name"`
	MacroCategoryID    *uuid.UUID         `bun:"macro_category_id" json:"macro_category_id,omitempty"`
	AvailabilityStatus AvailabilityStatus `bun:"availability_status,notnull,default:'active'" json:"availability_status"`
	Details            CategoryDetails    `bun:"details,type:jsonb,notnull" json:"details"`

	Items         Items          `bun:"rel:has-many,join:id=category_id" json:"items"`
	MacroCategory *MacroCategory `bun:"rel:belongs-to,join:macro_category_id=id" json:"macro_category,omitempty"`
	Schedules     Schedules      `bun:"rel:has-many,join:id=category_id" json:"schedules,omitempty"`
}

func (c *Category) ToDomainDTO() dtos.CategoryResponse {
	return dtos.CategoryResponse{
		ID:                 c.ID,
		ShopID:             c.ShopID,
		Name:               c.Name,
		MacroCategoryID:    c.MacroCategoryID,
		AvailabilityStatus: string(c.AvailabilityStatus),
		CanOrder:           c.AvailabilityStatus.CanOrder(),
		Details:            c.Details.ToDomainDTO(),
		Items:              c.Items.ToDomainDTO(c.Details),
		Schedules:          c.Schedules.ToDomainDTO(),
	}
}

// ToOwnerCatalogDTO expone la categoría completa (details/flavors, horarios, macro)
// sin cargar items — pensado para el panel al crear/editar productos.
func (c *Category) ToOwnerCatalogDTO() dtos.CategoryResponse {
	resp := c.ToDomainDTO()
	resp.Items = nil
	return resp
}

func (c *Category) ToCustomerDTO() dtos.CategoryResponse {
	resp := c.ToDomainDTO()
	resp.Items = c.Items.ToCustomerDTO(c.Details)
	return resp
}

func (c *Categories) ToDomainDTO() dtos.CategoriesResponse {

	var categoriesResponse dtos.CategoriesResponse

	for _, v := range *c {
		categoriesResponse.Add(v.ToDomainDTO())
	}

	return categoriesResponse

}

func (c *Categories) ToOwnerCatalogDTO() dtos.CategoriesResponse {
	var categoriesResponse dtos.CategoriesResponse
	for _, v := range *c {
		categoriesResponse.Add(v.ToOwnerCatalogDTO())
	}
	return categoriesResponse
}

// ToSummaryDTO expone solo id y name, pensado para poblar selectores del
// negocio (ej. elegir en qué categorías ofrecer una adición) sin cargar
// items ni horarios.
func (c *Category) ToSummaryDTO() dtos.CategorySummaryResponse {
	return dtos.CategorySummaryResponse{
		ID:   c.ID,
		Name: c.Name,
	}
}

func (c *Categories) ToCustomerDTO() dtos.CategoriesResponse {
	var categoriesResponse dtos.CategoriesResponse
	for _, v := range *c {
		categoriesResponse.Add(v.ToCustomerDTO())
	}
	return categoriesResponse
}

func (c *Categories) ToSummaryDTO() dtos.CategorySummariesResponse {
	var summaries dtos.CategorySummariesResponse

	for _, v := range *c {
		summaries.Add(v.ToSummaryDTO())
	}

	return summaries
}
