package models

import "Encargalo.app-api.go/internal/products/domain/dtos"

type CategoryDetails struct {
	Flavors      FlavorEntries `json:"flavors"`
	SelectorType string        `json:"selector_type"`
}

func DefaultCategoryDetails() CategoryDetails {
	return CategoryDetails{
		Flavors:      FlavorEntries{},
		SelectorType: SelectorNone,
	}
}

func (d CategoryDetails) ToDomainDTO() dtos.CategoryDetailsResponse {
	return dtos.CategoryDetailsResponse{
		Flavors:      d.Flavors.ToDomainDTO(),
		SelectorType: d.SelectorType,
	}
}

func (d CategoryDetails) CatalogFlavors() FlavorEntries {
	return d.Flavors
}
