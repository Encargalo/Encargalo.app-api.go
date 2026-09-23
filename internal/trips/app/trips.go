package app

import (
	"context"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type tripsApp struct {
	repo    ports.TripsRepo
	matcher ports.TripRiderMatcher
	bcv     bcvPorts.BcvApp
	quotes  ports.TripQuotesRepo
}

func NewTripsApp(repo ports.TripsRepo, matcher ports.TripRiderMatcher, bcv bcvPorts.BcvApp, quotes ports.TripQuotesRepo) ports.TripsApp {
	return &tripsApp{repo: repo, matcher: matcher, bcv: bcv, quotes: quotes}
}

// RequestTrip nunca calcula el fare: usa el que EstimateFare ya congeló en la quote.
func (a *tripsApp) RequestTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}

	fare := quote.FareUSD
	trip := &models.Trip{
		CustomerID:       customerID,
		Status:           models.TripStatusRequested,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
	}

	if cell, err := geo.LatLngToCell(quote.PickupLatitude, quote.PickupLongitude, geo.DriverMatchResolution); err == nil {
		trip.PickupH3Cell = &cell
	}
	if cell, err := geo.LatLngToCell(quote.PickupLatitude, quote.PickupLongitude, geo.ZoneAnalyticsResolution); err == nil {
		trip.PickupH3CellZone = &cell
	}
	if cell, err := geo.LatLngToCell(quote.DropoffLatitude, quote.DropoffLongitude, geo.ZoneAnalyticsResolution); err == nil {
		trip.DropoffH3CellZone = &cell
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

	// Tasa de la quote, no la vigente: FareBs debe coincidir con lo ya mostrado.
	trip.ApplyBsFare(quote.BcvRate)
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
	trip.ApplyBsFare(a.bcvRate(ctx))
	return trip.ToDTO(), nil
}

func (a *tripsApp) GetTripHistory(ctx context.Context, customerID uuid.UUID) (dtos.TripHistoryResponse, error) {
	trips, err := a.repo.GetTripHistoryByCustomer(ctx, customerID)
	if err != nil {
		return dtos.TripHistoryResponse{}, err
	}
	rate := a.bcvRate(ctx)
	out := make([]dtos.TripResponse, len(trips))
	for i := range trips {
		trips[i].ApplyBsFare(rate)
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
	trip.ApplyBsFare(a.bcvRate(ctx))
	return trip.ToDTO(), nil
}

// EstimateFare calcula el fare una sola vez y lo congela en una TripQuote.
func (a *tripsApp) EstimateFare(ctx context.Context, req dtos.FareEstimateRequest) (dtos.FareEstimateResponse, error) {
	estimate, err := a.fareEstimate(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	if err != nil {
		return dtos.FareEstimateResponse{}, err
	}
	rate := a.bcvRate(ctx)
	if rate <= 0 {
		return dtos.FareEstimateResponse{}, errcustom.ErrBcvRateUnavailable
	}

	now := time.Now()
	quote := &models.TripQuote{
		QuoteID:          uuid.New(),
		CustomerID:       req.CustomerID,
		PickupAddress:    req.PickupAddress,
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffAddress:   req.DropoffAddress,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		DistanceMeters:   estimate.DistanceMeters,
		DurationMin:      estimate.DurationMin,
		FareUSD:          estimate.Fee,
		FareBs:           money.RoundBs(estimate.Fee * rate),
		BcvRate:          rate,
		CreatedAt:        now,
		ExpiresAt:        now.Add(models.TripQuoteTTL),
	}
	if err := a.quotes.Save(ctx, quote, models.TripQuoteTTL); err != nil {
		return dtos.FareEstimateResponse{}, err
	}

	return dtos.FareEstimateResponse{
		QuoteID:        quote.QuoteID,
		DistanceMeters: quote.DistanceMeters,
		DurationMin:    quote.DurationMin,
		FareUSD:        quote.FareUSD,
		FareBs:         quote.FareBs,
		ExpiresAt:      quote.ExpiresAt,
	}, nil
}

// fareEstimate evita gastar cuota de Google Maps fuera de producción.
func (a *tripsApp) fareEstimate(pickupLat, pickupLon, dropoffLat, dropoffLon float64) (deliveryfee.Estimate, error) {
	if !config.Get().IsProduction() {
		distanceMeters := geo.HaversineKm(pickupLat, pickupLon, dropoffLat, dropoffLon) * 1000
		return deliveryfee.Estimate{
			DistanceMeters: distanceMeters,
			DurationMin:    0,
			Fee:            deliveryfee.DevPlaceholderFee,
		}, nil
	}
	return deliveryfee.CalculateWithDuration(pickupLat, pickupLon, dropoffLat, dropoffLon)
}

func (a *tripsApp) bcvRate(ctx context.Context) float64 {
	rate, err := a.bcv.GetRate(ctx)
	if err != nil || rate == nil {
		return 0
	}
	return rate.Rate
}
