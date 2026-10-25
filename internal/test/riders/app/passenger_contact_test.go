package app_test

import (
	"context"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOfferTrip_IncludesPassengerNameWithoutPhone(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	passengers := mocksriders.NewPassengerLookup(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, passengers, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	var offers [][]byte
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		raw := args.Get(0).([]byte)
		if messageType(t, raw) == "trip_offer" {
			offers = append(offers, append([]byte(nil), raw...))
		}
	}).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	trip.CustomerID = uuid.New()
	passengers.On("GetByCustomerID", mock.Anything, trip.CustomerID).
		Return(tripsDtos.PassengerContact{Name: "Ana", LastName: "Pérez", Phone: "04141234567"}, nil)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	require.NotEmpty(t, offers)
	for _, msg := range offers {
		require.Contains(t, string(msg), `"customer":{"name":"Ana","last_name":"Pérez"}`)
		require.NotContains(t, string(msg), "04141234567")
		require.NotContains(t, string(msg), `"phone"`)
	}
}

func TestOfferTrip_OmitsCustomerWhenPassengerLookupFails(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	passengers := mocksriders.NewPassengerLookup(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, passengers, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	riderID := uuid.New()
	var offers [][]byte
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		raw := args.Get(0).([]byte)
		if messageType(t, raw) == "trip_offer" {
			offers = append(offers, append([]byte(nil), raw...))
		}
	}).Return()
	require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

	trip := samplePickupTrip(t)
	trip.CustomerID = uuid.New()
	passengers.On("GetByCustomerID", mock.Anything, trip.CustomerID).
		Return(tripsDtos.PassengerContact{}, errcustom.ErrNotFound)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	require.NotEmpty(t, offers)
	for _, msg := range offers {
		require.NotContains(t, string(msg), `"customer"`)
	}
}

func TestAcceptTrip_SendsPassengerPhoneOnlyToAccepter(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	passengers := mocksriders.NewPassengerLookup(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, passengers, nil, nil)

	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	accepter := uuid.New()
	other := uuid.New()
	customerID := uuid.New()
	passengers.On("GetByCustomerID", mock.Anything, customerID).
		Return(tripsDtos.PassengerContact{Name: "Ana", LastName: "Pérez", Phone: "04141234567"}, nil)

	var accepted [][]byte
	connA := mocksriders.NewRiderConn(t)
	connA.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		raw := args.Get(0).([]byte)
		if messageType(t, raw) == "trip_accepted" {
			accepted = append(accepted, append([]byte(nil), raw...))
		}
	}).Return()
	connB := mocksriders.NewRiderConn(t)
	connB.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		require.NotEqual(t, "trip_accepted", messageType(t, args.Get(0).([]byte)))
		require.NotContains(t, string(args.Get(0).([]byte)), "04141234567")
	}).Return()

	require.NoError(t, svc.Register(context.Background(), accepter, connA, riderModels.RiderModeTrips))
	require.NoError(t, svc.Register(context.Background(), other, connB, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), accepter, 10.4806, -66.9036))
	require.NoError(t, svc.UpdateLocation(context.Background(), other, 10.4810, -66.9040))

	trip := samplePickupTrip(t)
	trip.CustomerID = customerID
	tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{
		ID: trip.ID, CustomerID: customerID, Status: tripsModels.TripStatusSearching,
		PickupLatitude: trip.PickupLatitude, PickupLongitude: trip.PickupLongitude, PickupH3Cell: trip.PickupH3Cell,
	}, nil)
	tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, accepter, mock.Anything).Return(true, nil)
	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	require.NoError(t, svc.AcceptTrip(context.Background(), accepter, trip.ID, connA))

	require.Len(t, accepted, 1)
	body := string(accepted[0])
	require.Contains(t, body, `"customer":{"name":"Ana","last_name":"Pérez","phone":"04141234567"}`)
}

func TestActiveTrips_IncludesPassengerWhileInProgress(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	passengers := mocksriders.NewPassengerLookup(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, passengers, nil, nil)

	riderID := uuid.New()
	customerID := uuid.New()
	fare := 2.5
	trip := tripsModels.Trip{
		ID: uuid.New(), CustomerID: customerID, RiderID: &riderID,
		Status: tripsModels.TripStatusInProgress, Fare: &fare,
	}
	tripsRepo.On("ListActiveTripsByRider", mock.Anything, riderID).Return([]tripsModels.Trip{trip}, nil)
	passengers.On("GetByCustomerID", mock.Anything, customerID).
		Return(tripsDtos.PassengerContact{Name: "Ana", LastName: "Pérez", Phone: "04141234567"}, nil)

	got, err := svc.ActiveTrips(context.Background(), riderID)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].Customer)
	require.Equal(t, "Ana", got[0].Customer.Name)
	require.Equal(t, "Pérez", got[0].Customer.LastName)
	require.Equal(t, "04141234567", got[0].Customer.Phone)
}

func TestCompleteTrip_DropsPassengerPhoneKeepsName(t *testing.T) {
	repo := mocksriders.NewRidersRepo(t)
	stream := mocksriders.NewRidersStream(t)
	bcv := mockbcvports.NewBcvApp(t)
	tripsRepo := mockstrips.NewTripsRepo(t)
	tripEarnings := mocksriders.NewTripEarningRecorder(t)
	passengers := mocksriders.NewPassengerLookup(t)
	svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, nil, nil, passengers, nil, nil)

	riderID, customerID, tripID := uuid.New(), uuid.New(), uuid.New()
	fare := deliveryfee.DevPlaceholderFee
	tripsRepo.On("GetTripByID", mock.Anything, tripID).Return(&tripsModels.Trip{
		ID: tripID, CustomerID: customerID, RiderID: &riderID,
		Status: tripsModels.TripStatusInProgress, Fare: &fare,
	}, nil)
	tripsRepo.On("CompleteTrip", mock.Anything, tripID, fare, "5678").Return(true, nil)
	bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	tripEarnings.On("RecordTripEarning", mock.Anything, riderID, tripID, fare, 40.0).Return(nil)
	passengers.On("GetByCustomerID", mock.Anything, customerID).
		Return(tripsDtos.PassengerContact{Name: "Ana", LastName: "Pérez", Phone: "04141234567"}, nil)

	var sent [][]byte
	conn := mocksriders.NewRiderConn(t)
	conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		sent = append(sent, append([]byte(nil), args.Get(0).([]byte)...))
	}).Return()

	require.NoError(t, svc.AdvanceTrip(context.Background(), riderID, tripID, "5678", conn))

	require.Len(t, sent, 1)
	require.Equal(t, "trip_status", messageType(t, sent[0]))
	require.Contains(t, string(sent[0]), `"customer":{"name":"Ana","last_name":"Pérez"}`)
	require.NotContains(t, string(sent[0]), "04141234567")
}
