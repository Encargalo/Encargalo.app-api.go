package app_test

import (
	"context"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTripsAppWithZoneFareRules(t *testing.T) (*mockstrips.TripsRepo, *mockbcvports.BcvApp, *mockstrips.TripQuotesRepo, *mockstrips.ZoneFareRulesRepo) {
	t.Helper()
	repo := mockstrips.NewTripsRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	zoneRules := mockstrips.NewZoneFareRulesRepo(t)
	return repo, bcv, quotes, zoneRules
}

var charallaveRule = &models.ZoneFareRule{
	ZoneID:                        uuid.New(),
	Name:                          "Charallave",
	IncludedKm:                    3,
	DayMinFareUSD:                 1.2375,
	DayExtraRatePerKmUSD:          0.30,
	NightMinFareUSD:               1.575,
	NightExtraRatePerKmUSD:        0.50,
	EarlyMorningMinFareUSD:        1.80,
	EarlyMorningExtraRatePerKmUSD: 0.70,
	Active:                        true,
}

// Pickup dentro de una zona con regla activa: el fallback global (placeholder
// fuera de producción) se reemplaza por la fórmula de esa zona, y la quote
// congela el ZoneID usado.
func TestEstimateFare_ZoneFareRule_AppliesWhenCellMatches(t *testing.T) {
	repo, bcv, quotes, zoneRules := newTripsAppWithZoneFareRules(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, zoneRules)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	zoneRules.On("FindActiveByCell", mock.Anything, mock.AnythingOfType("string")).Return(charallaveRule, nil)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)
	require.NoError(t, err)

	distanceMeters := geo.HaversineKm(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude) * 1000
	zoneParams := &tripfare.ZoneFareParams{
		IncludedKm:                 charallaveRule.IncludedKm,
		DayMinFare:                 charallaveRule.DayMinFareUSD,
		DayExtraRatePerKm:          charallaveRule.DayExtraRatePerKmUSD,
		NightMinFare:               charallaveRule.NightMinFareUSD,
		NightExtraRatePerKm:        charallaveRule.NightExtraRatePerKmUSD,
		EarlyMorningMinFare:        charallaveRule.EarlyMorningMinFareUSD,
		EarlyMorningExtraRatePerKm: charallaveRule.EarlyMorningExtraRatePerKmUSD,
	}
	expectedFee := tripfare.FromDistanceAtZone(distanceMeters, bizclock.Now(), zoneParams)

	require.Equal(t, expectedFee, resp.FareUSD)
	require.NotNil(t, saved.ZoneFareRuleID)
	require.Equal(t, charallaveRule.ZoneID, *saved.ZoneFareRuleID)
}

// Pickup fuera de cualquier zona (ErrNotFound): cae en el fallback global de
// siempre, sin tratar la ausencia de regla como error de la cotización.
func TestEstimateFare_ZoneFareRule_FallsBackWhenNoRuleMatches(t *testing.T) {
	repo, bcv, quotes, zoneRules := newTripsAppWithZoneFareRules(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, zoneRules)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)
	zoneRules.On("FindActiveByCell", mock.Anything, mock.AnythingOfType("string")).Return(nil, errcustom.ErrNotFound)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
	require.Nil(t, saved.ZoneFareRuleID)
}

// Sin el repo de zonas conectado (nil), igual que antes de esta feature:
// cotiza por el fallback global sin intentar resolver ninguna celda.
func TestEstimateFare_ZoneFareRule_NilRepoSkipsLookup(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	bcv := mockbcvports.NewBcvApp(t)
	quotes := mockstrips.NewTripQuotesRepo(t)
	svc := app.NewTripsApp(repo, nil, bcv, quotes, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	customerID := uuid.New()
	req := sampleFareEstimateRequest(customerID)
	mockBcvRate(bcv, 40)
	repo.On("HasBillableTrip", mock.Anything, customerID).Return(true, nil)

	var saved *models.TripQuote
	quotes.On("Save", mock.Anything, mock.Anything, models.TripQuoteTTL).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*models.TripQuote)
	}).Return(nil)

	resp, err := svc.EstimateFare(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, tripfare.DevPlaceholderFee, resp.FareUSD)
	require.Nil(t, saved.ZoneFareRuleID)
}
