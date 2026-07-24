package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
)

// CatalogFlavor ubica un sabor del catálogo embebido en una categoría.
type CatalogFlavor struct {
	ID         uuid.UUID
	ShopID     uuid.UUID
	CategoryID uuid.UUID
	Entry      FlavorEntry
}

func (f *CatalogFlavor) ToDomainDTO() dtos.FlavorResponse {
	return dtos.FlavorResponse{
		ID:        f.Entry.ID,
		Name:      f.Entry.Name,
		Available: f.Entry.IsAvailable(),
	}
}

func BuildCatalogFlavorFromDTO(flavor dtos.RegisterFlavor) CatalogFlavor {
	entry := FlavorEntry{
		ID:        uuid.New(),
		Name:      flavor.Name,
		Available: new(true),
	}
	if flavor.Description != "" {
		entry.Description = flavor.Description
	}
	return CatalogFlavor{
		ShopID:     flavor.ShopID,
		CategoryID: flavor.CategoryID,
		Entry:      entry,
	}
}
