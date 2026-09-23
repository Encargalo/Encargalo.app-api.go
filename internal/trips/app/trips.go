package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type tripsApp struct {
	repo      ports.TripsRepo
	matcher   ports.TripRiderMatcher
	bcv       bcvPorts.BcvApp
	quotes    ports.TripQuotesRepo
	upload    uploadimage.UploadImage
	reviews   paymentsPorts.PaymentReviewsApp
	hub       paymentsPorts.StaffReviewHub
	riders    ports.RiderPublicInfoLookup
	balance   paymentsPorts.CustomerBalanceCreditor
	customers ports.CustomerTripNotifier
}

func NewTripsApp(
	repo ports.TripsRepo,
	matcher ports.TripRiderMatcher,
	bcv bcvPorts.BcvApp,
	quotes ports.TripQuotesRepo,
	upload uploadimage.UploadImage,
	reviews paymentsPorts.PaymentReviewsApp,
	hub paymentsPorts.StaffReviewHub,
	riders ports.RiderPublicInfoLookup,
	balance paymentsPorts.CustomerBalanceCreditor,
	customers ports.CustomerTripNotifier,
) ports.TripsApp {
	return &tripsApp{
		repo:      repo,
		matcher:   matcher,
		bcv:       bcv,
		quotes:    quotes,
		upload:    upload,
		reviews:   reviews,
		hub:       hub,
		riders:    riders,
		balance:   balance,
		customers: customers,
	}
}

// RequestTrip nunca calcula el fare: usa el que EstimateFare ya congeló en la quote.
// El viaje nace en payment_review y no se oferta a riders hasta que el staff apruebe el comprobante.
func (a *tripsApp) RequestTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	if req.MethodPayment == models.MethodBalance {
		return a.requestTripWithBalance(ctx, customerID, req)
	}
	if req.MethodPayment != models.MethodPagoMovil || strings.TrimSpace(req.PaymentProofImage) == "" {
		return dtos.TripResponse{}, fmt.Errorf("%w: method_payment y payment_proof_image son requeridos", errcustom.ErrInvalidStatus)
	}

	proofUpload, err := a.upload.UploadImage(ctx, req.PaymentProofImage)
	if err != nil || proofUpload == nil || proofUpload.SecureURL == "" {
		return dtos.TripResponse{}, errcustom.ErrUploadingImage
	}
	proofURL := proofUpload.SecureURL

	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	expectedBs, ok := expectedAmountBs(&models.Trip{Fare: &quote.FareUSD, BcvRate: &quote.BcvRate})
	if !ok {
		return dtos.TripResponse{}, errcustom.ErrUnexpectedError
	}

	fare := quote.FareUSD
	rate := quote.BcvRate
	trip := &models.Trip{
		CustomerID:       customerID,
		Status:           models.TripStatusPaymentReview,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
		BcvRate:          &rate,
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

	review, err := a.reviews.CreateReview(ctx, models.ReferenceTypeTrip, trip.ID, proofURL, expectedBs, quote.BcvRate)
	if err != nil {
		if _, cancelErr := a.repo.UpdateTripStatus(ctx, trip.ID, models.TripStatusPaymentReview, models.TripStatusCancelled); cancelErr != nil {
			slog.Error("RequestTrip cancel after review failure", "tripID", trip.ID, "error", cancelErr)
		}
		return dtos.TripResponse{}, err
	}
	if a.hub != nil && review != nil {
		_ = a.hub.BroadcastReview(broadcastFromTrip(review, trip))
	}

	// Tasa de la quote, no la vigente: FareBs debe coincidir con lo ya mostrado.
	trip.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, trip)
	return trip.ToCustomerDTO(), nil
}

func (a *tripsApp) requestTripWithBalance(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	preview, err := a.quotes.Get(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	expectedBs, ok := expectedAmountBs(&models.Trip{Fare: &preview.FareUSD, BcvRate: &preview.BcvRate})
	if !ok {
		return dtos.TripResponse{}, errcustom.ErrUnexpectedError
	}
	if a.balance == nil {
		return dtos.TripResponse{}, errcustom.ErrUnexpectedError
	}
	available, err := a.balance.GetBalanceBs(ctx, customerID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if available+0.01 < expectedBs {
		return dtos.TripResponse{}, errcustom.ErrInsufficientBalance
	}
	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}

	fare := quote.FareUSD
	rate := quote.BcvRate
	trip := &models.Trip{
		CustomerID:       customerID,
		Status:           models.TripStatusSearching,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
		BcvRate:          &rate,
		BalancePaidBs:    &expectedBs,
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
	if err := a.balance.DebitForReference(ctx, customerID, trip.ID, models.ReferenceTypeTrip, expectedBs); err != nil {
		if _, cancelErr := a.repo.UpdateTripStatus(ctx, trip.ID, models.TripStatusSearching, models.TripStatusCancelled); cancelErr != nil {
			slog.Error("RequestTrip cancel after balance debit failure", "tripID", trip.ID, "error", cancelErr)
		}
		return dtos.TripResponse{}, err
	}
	if a.matcher != nil {
		a.matcher.OfferTrip(ctx, *trip)
	}
	trip.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, trip)
	return trip.ToCustomerDTO(), nil
}

func (a *tripsApp) GetTrip(ctx context.Context, tripID, requesterID uuid.UUID) (dtos.TripResponse, error) {
	trip, err := a.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if trip.CustomerID != requesterID {
		return dtos.TripResponse{}, errcustom.ErrTripForbidden
	}
	return a.toResponse(ctx, trip, a.bcvRate(ctx))
}

func (a *tripsApp) GetTripHistory(ctx context.Context, customerID uuid.UUID) (dtos.TripHistoryResponse, error) {
	trips, err := a.repo.GetTripHistoryByCustomer(ctx, customerID)
	if err != nil {
		return dtos.TripHistoryResponse{}, err
	}
	rate := a.bcvRate(ctx)
	out := make([]dtos.TripResponse, len(trips))
	for i := range trips {
		resp, err := a.toResponse(ctx, &trips[i], rate)
		if err != nil {
			return dtos.TripHistoryResponse{}, err
		}
		out[i] = resp
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
	if trip.BalancePaidBs != nil && *trip.BalancePaidBs > 0 && a.balance != nil {
		if refundErr := a.balance.RefundBalancePayment(ctx, trip.CustomerID, trip.ID, models.ReferenceTypeTrip, *trip.BalancePaidBs); refundErr != nil {
			slog.Error("CancelTrip refund balance", "tripID", trip.ID, "error", refundErr)
		}
	}
	resp, err := a.toResponse(ctx, trip, a.bcvRate(ctx))
	if err != nil {
		return dtos.TripResponse{}, err
	}
	a.notifyCustomer(ctx, trip)
	return resp, nil
}

func (a *tripsApp) toResponse(ctx context.Context, trip *models.Trip, rate float64) (dtos.TripResponse, error) {
	trip.ApplyBsFare(rate)
	resp := trip.ToCustomerDTO()
	if trip.RiderID == nil || a.riders == nil {
		return resp, nil
	}
	info, err := a.riders.GetByRiderID(ctx, *trip.RiderID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	resp.Rider = &info
	return resp, nil
}

func (a *tripsApp) notifyCustomer(ctx context.Context, trip *models.Trip) {
	if a.customers == nil || trip == nil {
		return
	}
	a.customers.NotifyTrip(ctx, *trip)
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
