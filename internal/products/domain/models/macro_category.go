package models

import (
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MacroCategories []MacroCategory

type MacroCategory struct {
	bun.BaseModel `bun:"table:products.macro_categories" swaggerignore:"true"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name      string    `bun:"name" json:"name"`
	Slug      string    `bun:"slug" json:"slug"`
	CreatedAt time.Time `bun:"created_at,default:now()" json:"-"`
}

func (m *MacroCategory) ToDomainDTO() dtos.MacroCategoryResponse {
	return dtos.MacroCategoryResponse{
		ID:   m.ID,
		Name: m.Name,
		Slug: m.Slug,
	}
}

func (m *MacroCategories) ToDomainDTO() dtos.MacroCategoriesResponse {
	var macroCategories dtos.MacroCategoriesResponse

	for _, v := range *m {
		macroCategories.Add(v.ToDomainDTO())
	}

	return macroCategories
}
