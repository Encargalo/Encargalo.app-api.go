package app

import (
	"context"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
)

// ShipmentStaffSnapshot arma la porción de la cola de staff que corresponde a envíos.
type ShipmentStaffSnapshot func(ctx context.Context) ([]paymentsModels.PaymentReviewBroadcast, error)

func NewShipmentStaffSnapshot(shipments ports.ShipmentsRepo, reviews paymentsPorts.PaymentReviewsRepo) ShipmentStaffSnapshot {
	return func(ctx context.Context) ([]paymentsModels.PaymentReviewBroadcast, error) {
		list, err := reviews.ListActiveForStaff(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]paymentsModels.PaymentReviewBroadcast, 0, len(list))
		for i := range list {
			review := list[i]
			if review.ReferenceType != models.ReferenceTypeShipment {
				continue
			}
			shipment, err := shipments.GetShipmentByID(ctx, review.ReferenceID)
			if err != nil || shipment == nil {
				continue
			}
			if shipment.Status != models.ShipmentStatusPaymentReview {
				continue
			}
			out = append(out, broadcastFromShipment(&review, shipment))
		}
		return out, nil
	}
}
