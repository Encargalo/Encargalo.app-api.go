package app

import (
	orderModels "Encargalo.app-api.go/internal/orders/domain/models"
	productModels "Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

func validateOrderCatalogAvailability(items productModels.Items, order *orderModels.Order) error {
	byID := make(map[uuid.UUID]productModels.Item, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	for _, line := range order.ItemsOrders {
		item, ok := byID[line.ItemID]
		if !ok {
			continue
		}

		if item.Category != nil {
			status := item.Category.AvailabilityStatus
			if !status.IsVisibleToCustomer() || !status.CanOrder() {
				return errcustom.ErrProductUnavailable
			}
		}

		if !productModels.EffectiveItemCanOrder(item.AvailabilityStatus, item.Details) {
			return errcustom.ErrProductUnavailable
		}

		if line.Additions == nil {
			continue
		}

		additionIDs := line.GetAdditionsID()
		if len(additionIDs) > productModels.MaxAdditionsSelection {
			return errcustom.ErrProductUnavailable
		}

		for _, addID := range additionIDs {
			entry, ok := item.Details.Additions.FindByID(addID)
			if !ok || !entry.IsAvailable() {
				return errcustom.ErrProductUnavailable
			}
		}
	}

	return nil
}
