package deliveryfee

import (
	"context"
	"log/slog"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
)

// ApplyBsToOrder fetches the current BCV rate and sets DeliveryFeeBs and TotalPriceBs on the order.
// If the rate cannot be fetched the fields are left at zero and a warning is logged.
func ApplyBsToOrder(ctx context.Context, bcv bcvPorts.BcvApp, order *ordersModels.Order) {
	rate, err := bcv.GetRate(ctx)
	if err != nil {
		slog.Warn("could not fetch BCV rate for order Bs prices", "error", err)
		return
	}
	order.ApplyBsPrice(rate.Rate)
}
