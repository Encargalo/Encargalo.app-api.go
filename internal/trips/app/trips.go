package app

import (
	"Encargalo.app-api.go/internal/pkg/id"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	paymentsPorts "Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
)

type tripsApp struct {
	repo        ports.TripsRepo
	matcher     ports.TripRiderMatcher
	bcv         bcvPorts.BcvApp
	quotes      ports.TripQuotesRepo
	upload      uploadimage.UploadImage
	reviews     paymentsPorts.PaymentReviewsApp
	hub         paymentsPorts.StaffReviewHub
	riders      ports.RiderPublicInfoLookup
	balance     paymentsPorts.CustomerBalanceCreditor
	customers   ports.CustomerTripNotifier
	nearby      ports.NearbyRidersLookup
	riderNotify ports.TripRiderNotifier
	events      ports.TripEventPublisher
	fareQuotes  ports.FareQuotesRepo
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
	nearby ports.NearbyRidersLookup,
	riderNotify ports.TripRiderNotifier,
	events ports.TripEventPublisher,
	fareQuotes ports.FareQuotesRepo,
) ports.TripsApp {
	return &tripsApp{
		repo:        repo,
		matcher:     matcher,
		bcv:         bcv,
		quotes:      quotes,
		upload:      upload,
		reviews:     reviews,
		hub:         hub,
		riders:      riders,
		balance:     balance,
		customers:   customers,
		nearby:      nearby,
		riderNotify: riderNotify,
		events:      events,
		fareQuotes:  fareQuotes,
	}
}

// RequestTrip es idempotente por req.ID: si el cliente ya creó el viaje con ese id
// (la respuesta se perdió y reintenta), lo devuelve sin volver a cobrar ni a ofertar.
func (a *tripsApp) RequestTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	if req.ID == uuid.Nil {
		return a.createTrip(ctx, customerID, req)
	}
	existing, err := a.existingTrip(ctx, customerID, req.ID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if existing != nil {
		return *existing, nil
	}

	resp, err := a.createTrip(ctx, customerID, req)
	if !errors.Is(err, errcustom.ErrQuoteNotFound) && !errors.Is(err, errcustom.ErrTripAlreadyExists) {
		return resp, err
	}
	// Un intento simultáneo con el mismo id pudo consumir la cotización o insertar primero.
	existing, lookupErr := a.existingTrip(ctx, customerID, req.ID)
	if lookupErr != nil || existing == nil {
		return resp, err
	}
	return *existing, nil
}

// existingTrip devuelve nil si no hay viaje con ese id. Un id de otro cliente no expone su viaje.
func (a *tripsApp) existingTrip(ctx context.Context, customerID, tripID uuid.UUID) (*dtos.TripResponse, error) {
	trip, err := a.repo.GetTripByID(ctx, tripID)
	if errors.Is(err, errcustom.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if trip.CustomerID != customerID {
		return nil, errcustom.ErrTripAlreadyExists
	}
	// Tasa congelada en el viaje, para responder el mismo fare_bs que la primera vez.
	rate := 0.0
	if trip.BcvRate != nil {
		rate = *trip.BcvRate
	}
	if rate <= 0 {
		rate = a.bcvRate(ctx)
	}
	resp, err := a.toResponse(ctx, trip, rate)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// createTrip nunca calcula el fare: usa el que EstimateFare ya congeló en la quote.
// Con Pago Móvil el viaje nace en payment_review y no se oferta hasta que el staff apruebe.
func (a *tripsApp) createTrip(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	if req.BalanceAmountBs > 0 || req.CashAmountBs > 0 {
		return a.requestTripSplit(ctx, customerID, req)
	}
	if req.MethodPayment == models.MethodBalance {
		return a.requestTripWithBalance(ctx, customerID, req)
	}
	if req.MethodPayment == models.MethodEfectivo {
		return a.requestTripCash(ctx, customerID, req)
	}
	if req.MethodPayment != models.MethodPagoMovil || strings.TrimSpace(req.PaymentProofImage) == "" {
		return dtos.TripResponse{}, fmt.Errorf("%w: method_payment y payment_proof_image son requeridos", errcustom.ErrInvalidStatus)
	}

	if _, err := a.quotes.Get(ctx, customerID, req.QuoteID); err != nil {
		return dtos.TripResponse{}, err
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
		ID:               req.ID,
		QuoteID:          &req.QuoteID,
		CustomerID:       customerID,
		Status:           models.TripStatusPaymentReview,
		VehicleType:      quote.VehicleType,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
		DiscountUSD:      quote.DiscountUSD,
		BcvRate:          &rate,
		MethodPayment:    models.MethodPagoMovil,
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
	a.publishPaymentProofPending(ctx, trip.ID, review)

	// Tasa de la quote, no la vigente: FareBs debe coincidir con lo ya mostrado.
	trip.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, trip)
	a.publishTripEvent(ctx, models.EventTripCreated, trip)
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
		ID:               req.ID,
		QuoteID:          &req.QuoteID,
		CustomerID:       customerID,
		Status:           models.TripStatusSearching,
		VehicleType:      quote.VehicleType,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
		DiscountUSD:      quote.DiscountUSD,
		BcvRate:          &rate,
		BalancePaidBs:    &expectedBs,
		MethodPayment:    models.MethodBalance,
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
	a.publishTripEvent(ctx, models.EventTripCreated, trip)
	return trip.ToCustomerDTO(), nil
}

// requestTripCash no cobra a la plataforma: el pasajero paga la tarifa al rider
// al terminar, y el viaje sale a buscar conductor de inmediato.
func (a *tripsApp) requestTripCash(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}

	fare := quote.FareUSD
	rate := quote.BcvRate
	trip := &models.Trip{
		ID:               req.ID,
		QuoteID:          &req.QuoteID,
		CustomerID:       customerID,
		Status:           models.TripStatusSearching,
		VehicleType:      quote.VehicleType,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
		DiscountUSD:      quote.DiscountUSD,
		BcvRate:          &rate,
		MethodPayment:    models.MethodEfectivo,
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
	if a.matcher != nil {
		a.matcher.OfferTrip(ctx, *trip)
	}
	trip.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, trip)
	a.publishTripEvent(ctx, models.EventTripCreated, trip)
	return trip.ToCustomerDTO(), nil
}

// requestTripSplit parte la tarifa en dos métodos. Si hay Pago Móvil, el viaje
// espera la revisión. Si no, sale a searching y el rider cobra solo el efectivo.
func (a *tripsApp) requestTripSplit(ctx context.Context, customerID uuid.UUID, req dtos.CreateTripRequest) (dtos.TripResponse, error) {
	preview, err := a.quotes.Get(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	if preview.BcvRate <= 0 || preview.FareUSD <= 0 {
		return dtos.TripResponse{}, errcustom.ErrUnexpectedError
	}
	totalBs := money.RoundBs(preview.FareUSD * preview.BcvRate)
	split, err := paymentsModels.ResolveTenders(totalBs, money.RoundBs(req.BalanceAmountBs), money.RoundBs(req.CashAmountBs), req.MethodPayment)
	if err != nil {
		return dtos.TripResponse{}, fmt.Errorf("%w: %s", errcustom.ErrInvalidStatus, err.Error())
	}
	if !split.IsPair() {
		return dtos.TripResponse{}, fmt.Errorf("%w: %s", errcustom.ErrInvalidStatus, paymentsModels.ErrTooManyTenders.Error())
	}
	if split.PagoMovilBs > 0.01 && strings.TrimSpace(req.PaymentProofImage) == "" {
		return dtos.TripResponse{}, fmt.Errorf("%w: payment_proof_image es requerido", errcustom.ErrInvalidStatus)
	}
	if split.BalanceBs > 0.01 {
		if a.balance == nil {
			return dtos.TripResponse{}, errcustom.ErrUnexpectedError
		}
		available, err := a.balance.GetBalanceBs(ctx, customerID)
		if err != nil {
			return dtos.TripResponse{}, err
		}
		if available+0.01 < split.BalanceBs {
			return dtos.TripResponse{}, errcustom.ErrInsufficientBalance
		}
	}

	var proofURL string
	if split.PagoMovilBs > 0.01 {
		proofUpload, err := a.upload.UploadImage(ctx, req.PaymentProofImage)
		if err != nil || proofUpload == nil || proofUpload.SecureURL == "" {
			return dtos.TripResponse{}, errcustom.ErrUploadingImage
		}
		proofURL = proofUpload.SecureURL
	}

	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.TripResponse{}, err
	}
	fare := quote.FareUSD
	rate := quote.BcvRate
	status := models.TripStatusSearching
	if split.PagoMovilBs > 0.01 {
		status = models.TripStatusPaymentReview
	}
	trip := &models.Trip{
		ID:               req.ID,
		QuoteID:          &req.QuoteID,
		CustomerID:       customerID,
		Status:           status,
		VehicleType:      quote.VehicleType,
		PickupAddress:    quote.PickupAddress,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffAddress:   quote.DropoffAddress,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		Fare:             &fare,
		DiscountUSD:      quote.DiscountUSD,
		BcvRate:          &rate,
		MethodPayment:    req.MethodPayment,
	}
	if split.BalanceBs > 0.01 {
		balanceBs := money.RoundBs(split.BalanceBs)
		trip.BalancePaidBs = &balanceBs
	}
	if split.CashBs > 0.01 {
		cashBs := money.RoundBs(split.CashBs)
		trip.CashAmountBs = &cashBs
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
	if trip.BalancePaidBs != nil {
		if err := a.balance.DebitForReference(ctx, customerID, trip.ID, models.ReferenceTypeTrip, *trip.BalancePaidBs); err != nil {
			if _, cancelErr := a.repo.UpdateTripStatus(ctx, trip.ID, status, models.TripStatusCancelled); cancelErr != nil {
				slog.Error("RequestTrip cancel after balance debit failure", "tripID", trip.ID, "error", cancelErr)
			}
			return dtos.TripResponse{}, err
		}
	}
	if split.PagoMovilBs > 0.01 {
		review, err := a.reviews.CreateReview(ctx, models.ReferenceTypeTrip, trip.ID, proofURL, money.RoundBs(split.PagoMovilBs), quote.BcvRate)
		if err != nil {
			if _, cancelErr := a.repo.UpdateTripStatus(ctx, trip.ID, status, models.TripStatusCancelled); cancelErr != nil {
				slog.Error("RequestTrip cancel after review failure", "tripID", trip.ID, "error", cancelErr)
			}
			if trip.BalancePaidBs != nil && *trip.BalancePaidBs > 0 && a.balance != nil {
				if refundErr := a.balance.RefundBalancePayment(ctx, customerID, trip.ID, models.ReferenceTypeTrip, *trip.BalancePaidBs); refundErr != nil {
					slog.Error("RequestTrip refund after review failure", "tripID", trip.ID, "error", refundErr)
				}
			}
			return dtos.TripResponse{}, err
		}
		if a.hub != nil && review != nil {
			_ = a.hub.BroadcastReview(broadcastFromTrip(review, trip))
		}
		a.publishPaymentProofPending(ctx, trip.ID, review)
	}
	if split.PagoMovilBs <= 0.01 && a.matcher != nil {
		a.matcher.OfferTrip(ctx, *trip)
	}
	trip.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, trip)
	a.publishTripEvent(ctx, models.EventTripCreated, trip)
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
	if trip.Status == models.TripStatusPaymentReview {
		return dtos.TripResponse{}, errcustom.ErrTripPaymentReviewPending
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

	verifiedPagoMovil := trip.HasVerifiedPagoMovil()
	trip.Status = models.TripStatusCancelled
	saveCancelReason(ctx, trip, models.CancelReasonCustomerCancelled, a.repo.SaveTripCancelReason)
	if verifiedPagoMovil {
		CreditUnservedPagoMovil(ctx, a.balance, trip, a.repo.SaveTripPayment)
	}
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
	a.notifyRidersCancelled(ctx, trip)
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

func (a *tripsApp) NearbyRiders(ctx context.Context, lat, lng float64) (dtos.NearbyRidersResponse, error) {
	if !geo.InVenezuela(lat, lng) {
		return dtos.NearbyRidersResponse{}, errcustom.ErrInvalidCoordinates
	}
	empty := dtos.NearbyRidersResponse{Riders: []dtos.NearbyRider{}}
	if a.nearby == nil {
		return empty, nil
	}
	riders, err := a.nearby.NearbyRiders(ctx, lat, lng)
	if err != nil {
		return dtos.NearbyRidersResponse{}, err
	}
	if riders == nil {
		riders = []dtos.NearbyRider{}
	}
	return dtos.NearbyRidersResponse{Riders: riders}, nil
}

func (a *tripsApp) notifyCustomer(ctx context.Context, trip *models.Trip) {
	if a.customers == nil || trip == nil {
		return
	}
	a.customers.NotifyTrip(ctx, *trip)
}

func (a *tripsApp) publishTripEvent(ctx context.Context, messageType string, trip *models.Trip) {
	if a.events == nil {
		return
	}
	event := models.DataEventTrip{}
	event.BuildFromTrip(messageType, *trip)
	if err := a.events.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing trip event", "type", messageType, "tripID", trip.ID, "error", err)
	}
}

func (a *tripsApp) publishPaymentProofPending(ctx context.Context, tripID uuid.UUID, review *paymentsModels.PaymentReview) {
	if a.events == nil || review == nil {
		return
	}
	event := models.DataEventTripPaymentProofPending{}
	event.BuildFrom(tripID, review.ID, review.ExpectedAmountBs, review.ProofURL, 0)
	if err := a.events.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing trip payment_proof_pending", "tripID", tripID, "error", err)
	}
}

func (a *tripsApp) notifyRidersCancelled(ctx context.Context, trip *models.Trip) {
	if a.riderNotify == nil || trip == nil {
		return
	}
	a.riderNotify.NotifyTripCancelled(ctx, *trip)
}

// EstimateFare calcula el fare una sola vez y lo congela en una TripQuote.
func (a *tripsApp) EstimateFare(ctx context.Context, req dtos.FareEstimateRequest) (dtos.FareEstimateResponse, error) {
	// El viaje se crea a partir de la cotización, así que este es el único punto donde se puede exigir que ambos extremos estén en el área de servicio.
	if !geo.InVenezuela(req.PickupLatitude, req.PickupLongitude) || !geo.InVenezuela(req.DropoffLatitude, req.DropoffLongitude) {
		return dtos.FareEstimateResponse{}, errcustom.ErrInvalidCoordinates
	}

	// El servicio se resuelve antes de gastar una llamada a Google Maps.
	vehicleType := req.VehicleType
	if vehicleType == "" {
		vehicleType = models.DefaultTripService
	}
	if !models.TripServiceAvailable(vehicleType) {
		return dtos.FareEstimateResponse{}, errcustom.ErrTripServiceUnavailable
	}

	estimate, err := a.fareEstimate(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	if err != nil {
		return dtos.FareEstimateResponse{}, err
	}
	rate := a.bcvRate(ctx)
	if rate <= 0 {
		return dtos.FareEstimateResponse{}, errcustom.ErrBcvRateUnavailable
	}

	discount := a.firstTripDiscount(ctx, req.CustomerID, estimate.Fee)
	fare := money.RoundUSD(estimate.Fee - discount)

	now := time.Now()
	quote := &models.TripQuote{
		QuoteID:          id.New(),
		CustomerID:       req.CustomerID,
		PickupAddress:    req.PickupAddress,
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffAddress:   req.DropoffAddress,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		DistanceMeters:   estimate.DistanceMeters,
		DurationMin:      estimate.DurationMin,
		FareUSD:          fare,
		DiscountUSD:      discount,
		FareBs:           money.RoundBs(fare * rate),
		BcvRate:          rate,
		FareBand:         string(estimate.Band),
		VehicleType:      vehicleType,
		CreatedAt:        now,
		ExpiresAt:        now.Add(models.TripQuoteTTL),
	}
	if err := a.quotes.Save(ctx, quote, models.TripQuoteTTL); err != nil {
		return dtos.FareEstimateResponse{}, err
	}
	a.recordFareQuote(ctx, quote)

	resp := dtos.FareEstimateResponse{
		QuoteID:        quote.QuoteID,
		DistanceMeters: quote.DistanceMeters,
		DurationMin:    quote.DurationMin,
		FareUSD:        quote.FareUSD,
		FareBs:         quote.FareBs,
		BcvRate:        quote.BcvRate,
		VehicleType:    quote.VehicleType,
		ExpiresAt:      quote.ExpiresAt,
	}
	if discount > 0 {
		resp.FareOriginalUSD = money.RoundUSD(estimate.Fee)
		resp.DiscountUSD = discount
	}
	return resp, nil
}

// Services mapea el catálogo del dominio al DTO: dtos no puede importar models.
func (a *tripsApp) Services() dtos.TripServicesResponse {
	out := make([]dtos.TripServiceResponse, 0, len(models.TripServices))
	for _, s := range models.TripServices {
		out = append(out, dtos.TripServiceResponse{
			Type:      s.Type,
			Name:      s.Name,
			Capacity:  s.Capacity,
			Available: s.Available,
		})
	}
	return dtos.TripServicesResponse{Services: out}
}

// firstTripDiscount aplica solo si el cliente no tiene viajes fuera de cancelled/no_driver.
// Si la consulta falla se cotiza sin descuento: perder la promo es mejor que no cotizar.
func (a *tripsApp) firstTripDiscount(ctx context.Context, customerID uuid.UUID, grossUSD float64) float64 {
	hasTrip, err := a.repo.HasBillableTrip(ctx, customerID)
	if err != nil {
		slog.Error("firstTripDiscount lookup failed", "customerID", customerID, "error", err)
		return 0
	}
	if hasTrip {
		return 0
	}
	return models.FirstTripDiscountUSD(grossUSD)
}

// recordFareQuote guarda la cotización en OLTP para medir conversión. Es un
// registro de analítica: un fallo acá se loguea pero no rompe la cotización.
func (a *tripsApp) recordFareQuote(ctx context.Context, quote *models.TripQuote) {
	if a.fareQuotes == nil {
		return
	}

	record := &models.FareQuote{
		QuoteID:          quote.QuoteID,
		CustomerID:       quote.CustomerID,
		PickupLatitude:   quote.PickupLatitude,
		PickupLongitude:  quote.PickupLongitude,
		DropoffLatitude:  quote.DropoffLatitude,
		DropoffLongitude: quote.DropoffLongitude,
		DistanceMeters:   quote.DistanceMeters,
		DurationMin:      quote.DurationMin,
		FareUSD:          quote.FareUSD,
		FareBs:           quote.FareBs,
		DiscountUSD:      quote.DiscountUSD,
		BcvRate:          quote.BcvRate,
		FareBand:         quote.FareBand,
		VehicleType:      quote.VehicleType,
		CreatedAt:        quote.CreatedAt,
	}
	if cell, err := geo.LatLngToCell(quote.PickupLatitude, quote.PickupLongitude, geo.ZoneAnalyticsResolution); err == nil {
		record.PickupH3CellZone = &cell
	}
	if cell, err := geo.LatLngToCell(quote.DropoffLatitude, quote.DropoffLongitude, geo.ZoneAnalyticsResolution); err == nil {
		record.DropoffH3CellZone = &cell
	}

	if err := a.fareQuotes.Insert(ctx, record); err != nil {
		slog.Error("recordFareQuote failed", "quoteID", quote.QuoteID, "error", err)
		return
	}
	a.publishFareQuoteCreated(ctx, quote.QuoteID)
}

func (a *tripsApp) publishFareQuoteCreated(ctx context.Context, quoteID uuid.UUID) {
	if a.events == nil {
		return
	}
	event := models.DataEventFareQuoteCreated{}
	event.BuildFromQuoteID(quoteID.String())
	if err := a.events.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing fare_quote_created event", "quoteID", quoteID, "error", err)
	}
}

// fareEstimate evita gastar cuota de Google Maps fuera de producción.
func (a *tripsApp) fareEstimate(pickupLat, pickupLon, dropoffLat, dropoffLon float64) (tripfare.Estimate, error) {
	if !config.Get().IsProduction() {
		distanceMeters := geo.HaversineKm(pickupLat, pickupLon, dropoffLat, dropoffLon) * 1000
		return tripfare.Estimate{
			DistanceMeters: distanceMeters,
			DurationMin:    0,
			Fee:            tripfare.DevPlaceholderFee,
			Band:           tripfare.BandAt(bizclock.Now()),
		}, nil
	}
	return tripfare.CalculateWithDuration(pickupLat, pickupLon, dropoffLat, dropoffLon)
}

func (a *tripsApp) bcvRate(ctx context.Context) float64 {
	rate, err := a.bcv.GetRate(ctx)
	if err != nil || rate == nil {
		return 0
	}
	return rate.Rate
}
