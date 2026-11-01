package app

import (
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
	"Encargalo.app-api.go/internal/pkg/code"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/shipmentfare"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
)

type shipmentsApp struct {
	repo        ports.ShipmentsRepo
	matcher     ports.ShipmentRiderMatcher
	bcv         bcvPorts.BcvApp
	quotes      ports.ShipmentQuotesRepo
	upload      uploadimage.UploadImage
	reviews     paymentsPorts.PaymentReviewsApp
	hub         paymentsPorts.StaffReviewHub
	riders      ports.RiderPublicInfoLookup
	balance     paymentsPorts.CustomerBalanceCreditor
	customers   ports.CustomerShipmentNotifier
	riderNotify ports.ShipmentRiderNotifier
	events      ports.ShipmentEventPublisher
	fareQuotes  ports.FareQuotesRepo
	contacts    ports.CustomerContactLookup
}

func NewShipmentsApp(
	repo ports.ShipmentsRepo,
	matcher ports.ShipmentRiderMatcher,
	bcv bcvPorts.BcvApp,
	quotes ports.ShipmentQuotesRepo,
	upload uploadimage.UploadImage,
	reviews paymentsPorts.PaymentReviewsApp,
	hub paymentsPorts.StaffReviewHub,
	riders ports.RiderPublicInfoLookup,
	balance paymentsPorts.CustomerBalanceCreditor,
	customers ports.CustomerShipmentNotifier,
	riderNotify ports.ShipmentRiderNotifier,
	events ports.ShipmentEventPublisher,
	fareQuotes ports.FareQuotesRepo,
	contacts ports.CustomerContactLookup,
) ports.ShipmentsApp {
	return &shipmentsApp{
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
		riderNotify: riderNotify,
		events:      events,
		fareQuotes:  fareQuotes,
		contacts:    contacts,
	}
}

// RequestShipment es idempotente por req.ID: si el cliente ya creó el envío con ese id
// (la respuesta se perdió y reintenta), lo devuelve sin volver a cobrar ni a ofertar.
func (a *shipmentsApp) RequestShipment(ctx context.Context, customerID uuid.UUID, req dtos.CreateShipmentRequest) (dtos.ShipmentResponse, error) {
	if req.ID == uuid.Nil {
		return a.createShipment(ctx, customerID, req)
	}
	existing, err := a.existingShipment(ctx, customerID, req.ID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	if existing != nil {
		return *existing, nil
	}

	resp, err := a.createShipment(ctx, customerID, req)
	if !errors.Is(err, errcustom.ErrQuoteNotFound) && !errors.Is(err, errcustom.ErrShipmentAlreadyExists) {
		return resp, err
	}
	existing, lookupErr := a.existingShipment(ctx, customerID, req.ID)
	if lookupErr != nil || existing == nil {
		return resp, err
	}
	return *existing, nil
}

// existingShipment devuelve nil si no hay envío con ese id. Un id de otro cliente no expone su envío.
func (a *shipmentsApp) existingShipment(ctx context.Context, customerID, shipmentID uuid.UUID) (*dtos.ShipmentResponse, error) {
	shipment, err := a.repo.GetShipmentByID(ctx, shipmentID)
	if errors.Is(err, errcustom.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if shipment.CustomerID != customerID {
		return nil, errcustom.ErrShipmentAlreadyExists
	}
	rate := 0.0
	if shipment.BcvRate != nil {
		rate = *shipment.BcvRate
	}
	if rate <= 0 {
		rate = a.bcvRate(ctx)
	}
	resp, err := a.toResponse(ctx, shipment, rate)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// createShipment nunca calcula el fare: usa el que EstimateFare ya congeló en
// la quote. Con Pago Móvil el envío nace en payment_review y no se oferta
// hasta que el staff apruebe.
func (a *shipmentsApp) createShipment(ctx context.Context, customerID uuid.UUID, req dtos.CreateShipmentRequest) (dtos.ShipmentResponse, error) {
	if req.MethodPayment == models.MethodBalance {
		return a.requestShipmentBalance(ctx, customerID, req)
	}
	if req.MethodPayment != models.MethodPagoMovil || strings.TrimSpace(req.PaymentProofImage) == "" {
		return dtos.ShipmentResponse{}, fmt.Errorf("%w: method_payment y payment_proof_image son requeridos", errcustom.ErrInvalidStatus)
	}

	if _, err := a.quotes.Get(ctx, customerID, req.QuoteID); err != nil {
		return dtos.ShipmentResponse{}, err
	}
	proofUpload, err := a.upload.UploadImage(ctx, req.PaymentProofImage)
	if err != nil || proofUpload == nil || proofUpload.SecureURL == "" {
		return dtos.ShipmentResponse{}, errcustom.ErrUploadingImage
	}
	proofURL := proofUpload.SecureURL

	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	expectedBs, ok := expectedAmountBs(&models.Shipment{Fare: &quote.FareUSD, BcvRate: &quote.BcvRate})
	if !ok {
		return dtos.ShipmentResponse{}, errcustom.ErrUnexpectedError
	}

	shipment, err := a.buildShipment(ctx, customerID, req, quote, models.ShipmentStatusPaymentReview)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	shipment.MethodPayment = models.MethodPagoMovil

	if err := a.repo.CreateShipment(ctx, shipment); err != nil {
		return dtos.ShipmentResponse{}, err
	}

	review, err := a.reviews.CreateReview(ctx, models.ReferenceTypeShipment, shipment.ID, proofURL, expectedBs, quote.BcvRate)
	if err != nil {
		if _, cancelErr := a.repo.UpdateShipmentStatus(ctx, shipment.ID, models.ShipmentStatusPaymentReview, models.ShipmentStatusCancelled); cancelErr != nil {
			slog.Error("RequestShipment cancel after review failure", "shipmentID", shipment.ID, "error", cancelErr)
		}
		return dtos.ShipmentResponse{}, err
	}
	if a.hub != nil && review != nil {
		_ = a.hub.BroadcastReview(broadcastFromShipment(review, shipment))
	}
	a.publishPaymentProofPending(ctx, shipment.ID, review)

	shipment.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, shipment)
	a.publishShipmentEvent(ctx, models.EventShipmentCreated, shipment)
	return shipment.ToCustomerDTO(), nil
}

func (a *shipmentsApp) requestShipmentBalance(ctx context.Context, customerID uuid.UUID, req dtos.CreateShipmentRequest) (dtos.ShipmentResponse, error) {
	preview, err := a.quotes.Get(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	expectedBs, ok := expectedAmountBs(&models.Shipment{Fare: &preview.FareUSD, BcvRate: &preview.BcvRate})
	if !ok {
		return dtos.ShipmentResponse{}, errcustom.ErrUnexpectedError
	}
	if a.balance == nil {
		return dtos.ShipmentResponse{}, errcustom.ErrUnexpectedError
	}
	available, err := a.balance.GetBalanceBs(ctx, customerID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	if available+0.01 < expectedBs {
		return dtos.ShipmentResponse{}, errcustom.ErrInsufficientBalance
	}
	quote, err := a.quotes.GetAndConsume(ctx, customerID, req.QuoteID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}

	shipment, err := a.buildShipment(ctx, customerID, req, quote, models.ShipmentStatusSearching)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	shipment.MethodPayment = models.MethodBalance
	shipment.BalancePaidBs = &expectedBs

	if err := a.repo.CreateShipment(ctx, shipment); err != nil {
		return dtos.ShipmentResponse{}, err
	}
	if err := a.balance.DebitForReference(ctx, customerID, shipment.ID, models.ReferenceTypeShipment, expectedBs); err != nil {
		if _, cancelErr := a.repo.UpdateShipmentStatus(ctx, shipment.ID, models.ShipmentStatusSearching, models.ShipmentStatusCancelled); cancelErr != nil {
			slog.Error("RequestShipment cancel after balance debit failure", "shipmentID", shipment.ID, "error", cancelErr)
		}
		return dtos.ShipmentResponse{}, err
	}
	if a.matcher != nil {
		a.matcher.OfferShipment(ctx, *shipment)
	}
	shipment.ApplyBsFare(quote.BcvRate)
	a.notifyCustomer(ctx, shipment)
	a.publishShipmentEvent(ctx, models.EventShipmentCreated, shipment)
	return shipment.ToCustomerDTO(), nil
}

// buildShipment arma el registro desde la cotización congelada y resuelve los
// contactos (sender/recipient), generando ambos códigos de verificación antes
// de persistir: a diferencia de trips, shipments los expone siempre al cliente.
func (a *shipmentsApp) buildShipment(ctx context.Context, customerID uuid.UUID, req dtos.CreateShipmentRequest, quote *models.ShipmentQuote, status models.ShipmentStatus) (*models.Shipment, error) {
	senderName, senderPhone, err := a.resolveContact(ctx, customerID, req.Sender)
	if err != nil {
		return nil, err
	}
	recipientName, recipientPhone, err := a.resolveContact(ctx, customerID, req.Recipient)
	if err != nil {
		return nil, err
	}
	pickupCode, err := code.GenerateFourDigits()
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}
	dropoffCode, err := code.GenerateFourDigits()
	if err != nil {
		return nil, errcustom.Unexpected(err)
	}
	packagePhotoURL, err := a.uploadPackagePhoto(ctx, req.PackagePhoto)
	if err != nil {
		return nil, err
	}

	fare := quote.FareUSD
	rate := quote.BcvRate
	shipment := &models.Shipment{
		// id.Ensure conserva el id de idempotencia del cliente si vino, o genera
		// uno nuevo (UUIDv7): ya no hay DEFAULT en la BD que lo resuelva por nosotros.
		ID:                  id.Ensure(req.ID),
		QuoteID:             &req.QuoteID,
		CustomerID:          customerID,
		Status:              status,
		PickupAddress:       quote.PickupAddress,
		PickupLatitude:      quote.PickupLatitude,
		PickupLongitude:     quote.PickupLongitude,
		DropoffAddress:      quote.DropoffAddress,
		DropoffLatitude:     quote.DropoffLatitude,
		DropoffLongitude:    quote.DropoffLongitude,
		SenderIsCustomer:    req.Sender.IsCustomer,
		SenderName:          senderName,
		SenderPhone:         senderPhone,
		RecipientIsCustomer: req.Recipient.IsCustomer,
		RecipientName:       recipientName,
		RecipientPhone:      recipientPhone,
		PackageDescription:  req.PackageDescription,
		PackagePhotoURL:     packagePhotoURL,
		Fare:                &fare,
		BcvRate:             &rate,
		PickupCode:          pickupCode,
		DropoffCode:         dropoffCode,
	}
	if cell, err := geo.LatLngToCell(quote.PickupLatitude, quote.PickupLongitude, geo.ZoneAnalyticsResolution); err == nil {
		shipment.PickupH3CellZone = &cell
	}
	if cell, err := geo.LatLngToCell(quote.DropoffLatitude, quote.DropoffLongitude, geo.ZoneAnalyticsResolution); err == nil {
		shipment.DropoffH3CellZone = &cell
	}
	return shipment, nil
}

// uploadPackagePhoto es opcional, a diferencia de payment_proof_image: una
// cadena vacía no es un error, simplemente no hay foto que subir.
func (a *shipmentsApp) uploadPackagePhoto(ctx context.Context, packagePhoto string) (*string, error) {
	if strings.TrimSpace(packagePhoto) == "" {
		return nil, nil
	}
	upload, err := a.upload.UploadImage(ctx, packagePhoto)
	if err != nil || upload == nil || upload.SecureURL == "" {
		return nil, errcustom.ErrUploadingImage
	}
	return &upload.SecureURL, nil
}

// resolveContact completa Name/Phone desde la cuenta del cliente cuando
// IsCustomer es true; si es un tercero, usa lo que llegó en el request
// (ya validado non-empty en dtos.CreateShipmentRequest.Validate).
func (a *shipmentsApp) resolveContact(ctx context.Context, customerID uuid.UUID, contact dtos.ShipmentContact) (name, phone string, err error) {
	if !contact.IsCustomer {
		return contact.Name, contact.Phone, nil
	}
	if a.contacts == nil {
		return contact.Name, contact.Phone, nil
	}
	return a.contacts.GetByCustomerID(ctx, customerID)
}

func (a *shipmentsApp) GetShipment(ctx context.Context, shipmentID, requesterID uuid.UUID) (dtos.ShipmentResponse, error) {
	shipment, err := a.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	if shipment.CustomerID != requesterID {
		return dtos.ShipmentResponse{}, errcustom.ErrShipmentForbidden
	}
	return a.toResponse(ctx, shipment, a.bcvRate(ctx))
}

func (a *shipmentsApp) GetShipmentHistory(ctx context.Context, customerID uuid.UUID) (dtos.ShipmentHistoryResponse, error) {
	shipments, err := a.repo.GetShipmentHistoryByCustomer(ctx, customerID)
	if err != nil {
		return dtos.ShipmentHistoryResponse{}, err
	}
	rate := a.bcvRate(ctx)
	out := make([]dtos.ShipmentResponse, len(shipments))
	for i := range shipments {
		resp, err := a.toResponse(ctx, &shipments[i], rate)
		if err != nil {
			return dtos.ShipmentHistoryResponse{}, err
		}
		out[i] = resp
	}
	return dtos.ShipmentHistoryResponse{Shipments: out}, nil
}

func (a *shipmentsApp) CancelShipment(ctx context.Context, shipmentID, requesterID uuid.UUID) (dtos.ShipmentResponse, error) {
	shipment, err := a.repo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	if shipment.CustomerID != requesterID {
		return dtos.ShipmentResponse{}, errcustom.ErrShipmentForbidden
	}
	if shipment.Status == models.ShipmentStatusPaymentReview {
		return dtos.ShipmentResponse{}, errcustom.ErrShipmentPaymentReviewPending
	}
	if !shipment.IsCancellable() {
		return dtos.ShipmentResponse{}, errcustom.ErrShipmentNotCancellable
	}

	ok, err := a.repo.UpdateShipmentStatus(ctx, shipmentID, shipment.Status, models.ShipmentStatusCancelled)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	if !ok {
		return dtos.ShipmentResponse{}, errcustom.ErrShipmentNotCancellable
	}

	verifiedPagoMovil := shipment.HasVerifiedPagoMovil()
	shipment.Status = models.ShipmentStatusCancelled
	a.publishShipmentEvent(ctx, models.EventShipmentStatusChanged, shipment)
	saveCancelReason(ctx, shipment, models.CancelReasonCustomerCancelled, a.repo.SaveShipmentCancelReason)
	if verifiedPagoMovil {
		CreditUnservedPagoMovil(ctx, a.balance, shipment, a.repo.SaveShipmentPayment)
	}
	if shipment.BalancePaidBs != nil && *shipment.BalancePaidBs > 0 && a.balance != nil {
		if refundErr := a.balance.RefundBalancePayment(ctx, shipment.CustomerID, shipment.ID, models.ReferenceTypeShipment, *shipment.BalancePaidBs); refundErr != nil {
			slog.Error("CancelShipment refund balance", "shipmentID", shipment.ID, "error", refundErr)
		}
	}
	resp, err := a.toResponse(ctx, shipment, a.bcvRate(ctx))
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	a.notifyCustomer(ctx, shipment)
	a.notifyRidersCancelled(ctx, shipment)
	return resp, nil
}

func (a *shipmentsApp) toResponse(ctx context.Context, shipment *models.Shipment, rate float64) (dtos.ShipmentResponse, error) {
	shipment.ApplyBsFare(rate)
	resp := shipment.ToCustomerDTO()
	if shipment.RiderID == nil || a.riders == nil {
		return resp, nil
	}
	info, err := a.riders.GetByRiderID(ctx, *shipment.RiderID)
	if err != nil {
		return dtos.ShipmentResponse{}, err
	}
	resp.Rider = &info
	return resp, nil
}

func (a *shipmentsApp) notifyCustomer(ctx context.Context, shipment *models.Shipment) {
	if a.customers == nil || shipment == nil {
		return
	}
	a.customers.NotifyShipment(ctx, *shipment)
}

func (a *shipmentsApp) publishShipmentEvent(ctx context.Context, messageType string, shipment *models.Shipment) {
	if a.events == nil {
		return
	}
	event := models.DataEventShipment{}
	event.BuildFromShipment(messageType, *shipment)
	if err := a.events.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing shipment event", "type", messageType, "shipmentID", shipment.ID, "error", err)
	}
}

func (a *shipmentsApp) publishPaymentProofPending(ctx context.Context, shipmentID uuid.UUID, review *paymentsModels.PaymentReview) {
	if a.events == nil || review == nil {
		return
	}
	event := models.DataEventShipmentPaymentProofPending{}
	event.BuildFrom(shipmentID, review.ID, review.ExpectedAmountBs, review.ProofURL, 0)
	if err := a.events.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing shipment payment_proof_pending", "shipmentID", shipmentID, "error", err)
	}
}

func (a *shipmentsApp) notifyRidersCancelled(ctx context.Context, shipment *models.Shipment) {
	if a.riderNotify == nil || shipment == nil {
		return
	}
	a.riderNotify.NotifyShipmentCancelled(ctx, *shipment)
}

// EstimateFare calcula el fare una sola vez y lo congela en una ShipmentQuote.
func (a *shipmentsApp) EstimateFare(ctx context.Context, req dtos.FareEstimateRequest) (dtos.FareEstimateResponse, error) {
	if !geo.InVenezuela(req.PickupLatitude, req.PickupLongitude) || !geo.InVenezuela(req.DropoffLatitude, req.DropoffLongitude) {
		return dtos.FareEstimateResponse{}, errcustom.ErrInvalidCoordinates
	}

	estimate, err := a.fareEstimate(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	if err != nil {
		return dtos.FareEstimateResponse{}, err
	}
	rate := a.bcvRate(ctx)
	if rate <= 0 {
		return dtos.FareEstimateResponse{}, errcustom.ErrBcvRateUnavailable
	}

	fare := money.RoundUSD(estimate.Fee)
	now := time.Now()
	quote := &models.ShipmentQuote{
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
		FareBs:           money.RoundBs(fare * rate),
		BcvRate:          rate,
		FareBand:         string(estimate.Band),
		CreatedAt:        now,
		ExpiresAt:        now.Add(models.ShipmentQuoteTTL),
	}
	if err := a.quotes.Save(ctx, quote, models.ShipmentQuoteTTL); err != nil {
		return dtos.FareEstimateResponse{}, err
	}
	a.recordFareQuote(ctx, quote)

	return dtos.FareEstimateResponse{
		QuoteID:        quote.QuoteID,
		DistanceMeters: quote.DistanceMeters,
		DurationMin:    quote.DurationMin,
		FareUSD:        quote.FareUSD,
		FareBs:         quote.FareBs,
		BcvRate:        quote.BcvRate,
		ExpiresAt:      quote.ExpiresAt,
	}, nil
}

// recordFareQuote guarda la cotización en OLTP para medir conversión. Es un
// registro de analítica: un fallo acá se loguea pero no rompe la cotización.
func (a *shipmentsApp) recordFareQuote(ctx context.Context, quote *models.ShipmentQuote) {
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
		BcvRate:          quote.BcvRate,
		FareBand:         quote.FareBand,
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
	}
}

// fareEstimate evita gastar cuota de Google Maps fuera de producción.
func (a *shipmentsApp) fareEstimate(pickupLat, pickupLon, dropoffLat, dropoffLon float64) (tripfare.Estimate, error) {
	if !config.Get().IsProduction() {
		distanceMeters := geo.HaversineKm(pickupLat, pickupLon, dropoffLat, dropoffLon) * 1000
		now := bizclock.Now()
		return tripfare.Estimate{
			DistanceMeters: distanceMeters,
			DurationMin:    0,
			Fee:            shipmentfare.DevPlaceholderFee,
			Band:           tripfare.BandAt(now),
		}, nil
	}
	return shipmentfare.CalculateWithDuration(pickupLat, pickupLon, dropoffLat, dropoffLon)
}

func (a *shipmentsApp) bcvRate(ctx context.Context) float64 {
	rate, err := a.bcv.GetRate(ctx)
	if err != nil || rate == nil {
		return 0
	}
	return rate.Rate
}
