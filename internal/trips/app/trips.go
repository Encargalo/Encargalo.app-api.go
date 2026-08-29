package app

import (
	"context"
	"strings"

	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type tripsApp struct {
	repo    ports.TripsRepo
	matcher ports.TripRiderMatcher
}

func NewTripsApp(repo ports.TripsRepo, matcher ports.TripRiderMatcher) ports.TripsApp {
	return &tripsApp{repo: repo, matcher: matcher}
}

func (a *tripsApp) RequestTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	trip := &models.Trip{
		CustomerID:       customerID,
		Status:           models.TripStatusRequested,
		PickupAddress:    strings.TrimSpace(req.PickupAddress),
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffAddress:   strings.TrimSpace(req.DropoffAddress),
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
	}

	if cell, err := geo.LatLngToCell(req.PickupLatitude, req.PickupLongitude, geo.DriverMatchResolution); err == nil {
		trip.PickupH3Cell = &cell
	}

	if err := a.repo.CreateTrip(ctx, trip); err != nil {
		return dtos.TripResponse{}, err
	}

	ok, err := a.repo.UpdateTripStatus(ctx, trip.ID, models.TripStatusRequested, models.TripStatusSearching)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if !ok {
		return dtos.TripResponse{}, errcustom.ErrUnexpectedError
	}

	trip.Status = models.TripStatusSearching
	a.matcher.OfferTrip(ctx, *trip)

	return trip.ToDTO(), nil
}

func (a *tripsApp) GetTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error) {
	trip, err := a.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if trip.CustomerID != requesterID {
		return dtos.TripResponse{}, errcustom.ErrTripForbidden
	}
	return trip.ToDTO(), nil
}

func (a *tripsApp) GetTripHistory(ctx context.Context, customerID uuid.UUID) (dtos.TripHistoryResponse, error) {
	trips, err := a.repo.GetTripHistoryByCustomer(ctx, customerID)
	if err != nil {
		return dtos.TripHistoryResponse{}, err
	}
	out := make([]dtos.TripResponse, len(trips))
	for i := range trips {
		out[i] = trips[i].ToDTO()
	}
	return dtos.TripHistoryResponse{Trips: out}, nil
}

func (a *tripsApp) CancelTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error) {
	trip, err := a.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if trip.CustomerID != requesterID {
		return dtos.TripResponse{}, errcustom.ErrTripForbidden
	}
	if !trip.IsCancellable() {
		return dtos.TripResponse{}, errcustom.ErrTripNotCancellable
	}

	ok, err := a.repo.UpdateTripStatus(ctx, tripID, trip.Status, models.TripStatusCancelled)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if !ok {
		return dtos.TripResponse{}, errcustom.ErrTripNotCancellable
	}

	trip.Status = models.TripStatusCancelled
	return trip.ToDTO(), nil
}
