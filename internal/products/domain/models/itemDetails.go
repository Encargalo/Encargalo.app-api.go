package models

import "Encargalo.app-api.go/internal/products/domain/dtos"

type ItemDetails struct {
	Flavors               FlavorEntries   `json:"flavors"`
	MinSelection          int             `json:"min_selection"`
	MaxSelection          int             `json:"max_selection"`
	Additions             AdditionEntries `json:"additions"`
	MaxAdditionsSelection int             `json:"max_additions_selection"`
}

func DefaultItemDetails() ItemDetails {
	return ItemDetails{
		Flavors:               FlavorEntries{},
		MinSelection:          0,
		MaxSelection:          1,
		Additions:             AdditionEntries{},
		MaxAdditionsSelection: 0,
	}
}

func (d ItemDetails) ToDomainDTO() dtos.ItemDetailsResponse {
	return dtos.ItemDetailsResponse{
		Flavors:               d.Flavors.ToDomainDTO(),
		MinSelection:          d.MinSelection,
		MaxSelection:          d.MaxSelection,
		Additions:             d.Additions.ToDomainDTO(),
		MaxAdditionsSelection: d.MaxAdditionsSelection,
	}
}

func (d ItemDetails) ToFlavorSelection(category CategoryDetails) dtos.FlavorSelectionResponse {
	return dtos.FlavorSelectionResponse{
		SelectorType: category.SelectorType,
		MinSelection: d.MinSelection,
		MaxSelection: d.MaxSelection,
	}
}

func (d ItemDetails) ToAdditionSelection() dtos.AdditionSelectionResponse {
	return dtos.AdditionSelectionResponse{
		SelectorType: SelectorMultiSelect,
		MinSelection: 0,
		MaxSelection: MaxAdditionsSelection,
	}
}

func (d *ItemDetails) ApplyFixedAdditionsLimits() {
	if d.HasAdditions() {
		d.MaxAdditionsSelection = MaxAdditionsSelection
		return
	}
	d.MaxAdditionsSelection = 0
}

func (d ItemDetails) HasFlavors() bool {
	return len(d.Flavors) > 0
}

func (d ItemDetails) HasAdditions() bool {
	return len(d.Additions) > 0
}
