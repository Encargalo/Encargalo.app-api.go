package app

import (
	"context"

	"Encargalo.app-api.go/internal/orders/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
)

// OrderStaffSnapshot builds the staff queue slice for payment reviews of orders.
type OrderStaffSnapshot func(ctx context.Context) ([]paymentsModels.PaymentReviewBroadcast, error)

func NewOrderStaffSnapshot(orders ports.OrdersRepo, reviews paymentsPorts.PaymentReviewsRepo) OrderStaffSnapshot {
	return func(ctx context.Context) ([]paymentsModels.PaymentReviewBroadcast, error) {
		list, err := reviews.ListActiveForStaff(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]paymentsModels.PaymentReviewBroadcast, 0, len(list))
		for i := range list {
			review := list[i]
			if review.ReferenceType != paymentsModels.ReferenceTypeOrder {
				continue
			}
			order, err := orders.SearchOrdersByID(ctx, review.ReferenceID)
			if err != nil || order == nil {
				continue
			}
			out = append(out, broadcastFromOrder(&review, order))
		}
		return out, nil
	}
}
