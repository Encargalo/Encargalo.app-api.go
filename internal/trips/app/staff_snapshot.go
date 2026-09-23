package app

import (
	"context"

	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
)

// TripStaffSnapshot arma la porción de la cola de staff que corresponde a viajes.
type TripStaffSnapshot func(ctx context.Context) ([]paymentsModels.PaymentReviewBroadcast, error)

func NewTripStaffSnapshot(trips ports.TripsRepo, reviews paymentsPorts.PaymentReviewsRepo) TripStaffSnapshot {
	return func(ctx context.Context) ([]paymentsModels.PaymentReviewBroadcast, error) {
		list, err := reviews.ListActiveForStaff(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]paymentsModels.PaymentReviewBroadcast, 0, len(list))
		for i := range list {
			review := list[i]
			if review.ReferenceType != models.ReferenceTypeTrip {
				continue
			}
			trip, err := trips.GetTripByID(ctx, review.ReferenceID)
			if err != nil || trip == nil {
				continue
			}
			out = append(out, broadcastFromTrip(&review, trip))
		}
		return out, nil
	}
}
