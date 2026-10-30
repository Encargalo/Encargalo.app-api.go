package app_test

import (
	"context"
	"errors"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEstimateFare_RecordsFareQuoteForConversion(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	fareQuotes := mockstrips.NewFareQuotesRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, fareQuotes, nil, nil)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)

	var recorded *models.FareQuote
	fareQuotes.On("Insert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		recorded = args.Get(1).(*models.FareQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)
	require.NoError(t, err)

	require.NotNil(t, recorded, "la cotización debe quedar registrada para medir conversión")
	require.Equal(t, resp.QuoteID, recorded.QuoteID)
	require.Equal(t, customerID, recorded.CustomerID)
	require.Equal(t, resp.FareUSD, recorded.FareUSD)
	require.Equal(t, resp.FareBs, recorded.FareBs)
	require.Equal(t, resp.DistanceMeters, recorded.DistanceMeters)
	require.Equal(t, 40.0, recorded.BcvRate)
	// El tramo viene congelado de la cotización, no recalculado al registrar.
	require.Equal(t, string(tripfare.BandAt(bizclock.Now())), recorded.FareBand)
	require.NotEmpty(t, recorded.FareBand)
	require.NotNil(t, recorded.PickupH3CellZone)
	require.NotNil(t, recorded.DropoffH3CellZone)
}

// El ETL resincroniza CubeFareQuote por evento, no por el batch horario.
func TestEstimateFare_PublishesFareQuoteCreated(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	fareQuotes := mockstrips.NewFareQuotesRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	events := &fakeTripEvents{}
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, events, fareQuotes, nil, nil)

	req := sampleFareEstimateRequest(uuid.New())
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, req.CustomerID).Return(true, nil)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)
	fareQuotes.On("Insert", mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, []any{models.DataEventFareQuoteCreated{
		MessageType: "fare_quote_created",
		QuoteID:     resp.QuoteID.String(),
	}}, events.events)
}

// El registro es analítica: si la escritura falla, la cotización sigue sirviéndose.
func TestEstimateFare_SucceedsWhenFareQuoteRecordFails(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	fareQuotes := mockstrips.NewFareQuotesRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, fareQuotes, nil, nil)

	req := sampleFareEstimateRequest(uuid.New())
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, req.CustomerID).Return(true, nil)
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Return(nil)
	fareQuotes.On("Insert", mock.Anything, mock.Anything).Return(errors.New("db caída"))

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, resp.QuoteID)
}

// El enlace de conversión: el viaje guarda el quote_id de la cotización que lo
// originó. Una fila de fare_quotes sin trip que la referencie no convirtió.
func TestRequestTrip_LinksQuoteIDOnTrip(t *testing.T) {
	d := newTripsDeps(t)
	customerID := uuid.New()
	quote := sampleTripQuote(customerID)

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
	require.NotNil(t, created.QuoteID, "sin quote_id el viaje no se puede atribuir a su cotización")
	require.Equal(t, quote.QuoteID, *created.QuoteID)
}
