package models

import "Encargalo.app-api.go/internal/products/domain/dtos"

func BoolPtr(v bool) *bool {
	b := v
	return &b
}

func (d ItemDetails) MeetsFlavorMinimum() bool {
	if !d.HasFlavors() {
		return true
	}
	return d.Flavors.CountAvailable() >= d.MinSelection
}

func (d ItemDetails) ToCustomerDTO() dtos.ItemDetailsResponse {
	return dtos.ItemDetailsResponse{
		Flavors:               d.Flavors.ToCustomerDTO(),
		MinSelection:          d.MinSelection,
		MaxSelection:          d.MaxSelection,
		Additions:             d.Additions.ToCustomerDTO(),
		MaxAdditionsSelection: d.MaxAdditionsSelection,
	}
}

func EffectiveItemAvailability(stored AvailabilityStatus, details ItemDetails) AvailabilityStatus {
	if stored == AvailabilityHidden {
		return AvailabilityHidden
	}
	if !details.MeetsFlavorMinimum() {
		return AvailabilityOutOfStock
	}
	return stored
}

func EffectiveItemCanOrder(stored AvailabilityStatus, details ItemDetails) bool {
	return EffectiveItemAvailability(stored, details).CanOrder()
}

func SyncIsAvailableFlag(status AvailabilityStatus) bool {
	return status.CanOrder()
}
