package dtos

type FlavorSelectionResponse struct {
	SelectorType string `json:"selector_type" example:"single_select"`
	MinSelection int    `json:"min_selection" example:"1"`
	MaxSelection int    `json:"max_selection" example:"1"`
}

type ItemDetailsResponse struct {
	Flavors               FlavorsResponse   `json:"flavors,omitempty"`
	MinSelection          int               `json:"min_selection,omitempty" example:"1"`
	MaxSelection          int               `json:"max_selection,omitempty" example:"2"`
	Additions             AdditionsResponse `json:"additions,omitempty"`
	MaxAdditionsSelection int               `json:"max_additions_selection,omitempty" example:"5"`
}

type AdditionSelectionResponse struct {
	SelectorType string `json:"selector_type" example:"multi_select"`
	MinSelection int    `json:"min_selection" example:"0"`
	MaxSelection int    `json:"max_selection" example:"5"`
}

type CategoryDetailsResponse struct {
	Flavors      FlavorsResponse `json:"flavors,omitempty"`
	SelectorType string          `json:"selector_type" example:"multi_select"`
}
