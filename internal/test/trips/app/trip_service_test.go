package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// La app actual no manda vehicle_type: sin default dejaría de cotizar.
func TestEstimateFare_DefaultsToMotoWhenVehicleTypeOmitted(t *testing.T) {
	svc, repo, _, bcv, quotes := newTripsApp(t)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	require.Empty(t, req.VehicleType, "el caso que se prueba es el request sin el campo")
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, riderModels.VehicleTypeMoto, resp.VehicleType)
	require.NotNil(t, saved)
	require.Equal(t, riderModels.VehicleTypeMoto, saved.VehicleType)
}

func TestEstimateFare_FreezesRequestedVehicleType(t *testing.T) {
	svc, repo, _, bcv, quotes := newTripsApp(t)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	req.VehicleType = riderModels.VehicleTypeMoto
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)
	require.NoError(t, err)

	require.Equal(t, riderModels.VehicleTypeMoto, resp.VehicleType)
	require.Equal(t, riderModels.VehicleTypeMoto, saved.VehicleType)
}

// El carro está en el catálogo pero no disponible: cotizarlo se corta antes de
// gastar una llamada a Google Maps y antes de guardar la cotización.
func TestEstimateFare_RejectsUnavailableService(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := sampleFareEstimateRequest(uuid.New())
	req.VehicleType = riderModels.VehicleTypeCar

	// Sin expectativas de Save ni GetRate: si el guard no corta, mockery falla.
	_, err := svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrTripServiceUnavailable)
}

func TestEstimateFare_RejectsUnknownService(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	svc := app.NewTripsApp(repo, nil, mockbcvports.NewBcvApp(t), mockstrips.NewTripQuotesRepo(t), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	req := sampleFareEstimateRequest(uuid.New())
	req.VehicleType = "helicoptero"

	_, err := svc.EstimateFare(context.Background(), req)

	require.ErrorIs(t, err, errcustom.ErrTripServiceUnavailable)
}

// El tipo llega a OLTP: sin él no se puede medir conversión por servicio.
func TestEstimateFare_RecordsVehicleTypeForConversion(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	fareQuotes := mockstrips.NewFareQuotesRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, fareQuotes, nil)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	req.VehicleType = riderModels.VehicleTypeMoto
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	var recorded *models.FareQuote
	fareQuotes.On("Insert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		recorded = args.Get(1).(*models.FareQuote)
	}).Return(nil)

	_, err := svc.EstimateFare(context.Background(), req)
	require.NoError(t, err)

	require.NotNil(t, recorded)
	require.Equal(t, riderModels.VehicleTypeMoto, recorded.VehicleType)
}

// El viaje hereda el tipo de la cotización. Si saliera del request, un cliente
// podría cotizar en moto y pedir un carro con la tarifa ya congelada.
func TestRequestTrip_InheritsVehicleTypeFromQuote(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)
	quote.VehicleType = riderModels.VehicleTypeMoto

	var created *models.Trip
	d.quotes.On("GetAndConsume", mock.Anything, customerID, quote.QuoteID).Return(quote, nil)
	d.repo.On("CreateTrip", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		created = args.Get(1).(*models.Trip)
		created.ID = uuid.New()
	}).Return(nil)
	d.matcher.On("OfferTrip", mock.Anything, mock.Anything).Return()

	_, err := d.svc.RequestTrip(context.Background(), customerID, dtos.CreateTripRequest{
		QuoteID:       quote.QuoteID,
		MethodPayment: models.MethodEfectivo,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, riderModels.VehicleTypeMoto, created.VehicleType)
}

// El catálogo es lo que la app usa para pintar la elección: la moto se puede
// pedir y el carro aparece deshabilitado, sin cambiar el contrato al habilitarlo.
func TestServices_ExposesMotoAvailableAndCarNotYet(t *testing.T) {
	d := newTripsDeps(t)

	resp := d.svc.Services()

	require.Len(t, resp.Services, 2)
	byType := map[string]dtos.TripServiceResponse{}
	for _, s := range resp.Services {
		byType[s.Type] = s
	}

	moto, ok := byType[riderModels.VehicleTypeMoto]
	require.True(t, ok, "la moto tiene que estar en el catálogo")
	require.True(t, moto.Available)
	require.NotEmpty(t, moto.Name)

	car, ok := byType[riderModels.VehicleTypeCar]
	require.True(t, ok, "el carro nace en el contrato aunque no se pueda pedir")
	require.False(t, car.Available, "sin riders en carro, un viaje así expiraría sin conductor")
}

// El catálogo y el guard no se pueden contradecir: lo que dice available es
// exactamente lo que EstimateFare acepta.
func TestTripServiceAvailable_MatchesTheCatalog(t *testing.T) {
	for _, s := range models.TripServices {
		require.Equal(t, s.Available, models.TripServiceAvailable(s.Type), "servicio %q", s.Type)
	}
	require.False(t, models.TripServiceAvailable("helicoptero"))
	require.False(t, models.TripServiceAvailable(""))
}
