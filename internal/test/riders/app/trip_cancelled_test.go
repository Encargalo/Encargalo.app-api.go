package app_test

import (
	"context"
	"encoding/json"
	"testing"

	mockbcvports "Encargalo.app-api.go/internal/mocks/bcv/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/riders/app"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	tripsPorts "Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func asTripNotifier(t *testing.T, svc ridersPorts.RidersApp) tripsPorts.TripRiderNotifier {
	t.Helper()
	n, ok := svc.(tripsPorts.TripRiderNotifier)
	require.True(t, ok, "appRidersManager debe implementar tripsPorts.TripRiderNotifier")
	return n
}

func TestNotifyTripCancelled_AssignedRiderStopsLocationRelay(t *testing.T) {
	for _, status := range []tripsModels.TripStatus{tripsModels.TripStatusAccepted, tripsModels.TripStatusArriving} {
		t.Run(string(status), func(t *testing.T) {
			repo := mocksriders.NewRidersRepo(t)
			stream := newRidersStream(t)
			bcv := mockbcvports.NewBcvApp(t)
			tripsRepo := mockstrips.NewTripsRepo(t)
			tripEarnings := mocksriders.NewTripEarningRecorder(t)
			customer := &recordingCustomer{}
			svc := app.New(repo, stream, bcv, tripsRepo, tripEarnings, customer, nil, nil, nil, newPresenceStore(t), newOfferTimeoutStore(t), nil, nil)

			repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
			repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

			riderID := uuid.New()
			customerID := uuid.New()
			var sent [][]byte
			conn := mocksriders.NewRiderConn(t)
			conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
				sent = append(sent, append([]byte(nil), args.Get(0).([]byte)...))
			}).Return()
			require.NoError(t, svc.Register(context.Background(), riderID, conn, riderModels.RiderModeTrips))
			require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.4806, -66.9036))

			trip := samplePickupTrip(t)
			t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
			trip.CustomerID = customerID
			tripsRepo.On("GetTripByID", mock.Anything, trip.ID).Return(&tripsModels.Trip{
				ID: trip.ID, CustomerID: customerID, Status: tripsModels.TripStatusSearching,
			}, nil)
			tripsRepo.On("AssignTripRider", mock.Anything, trip.ID, riderID, mock.Anything, mock.Anything).Return(true, nil)
			asMatcher(t, svc).OfferTrip(context.Background(), trip)
			require.NoError(t, svc.AcceptTrip(context.Background(), riderID, trip.ID, conn))

			require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.481, -66.904))
			require.Equal(t, 1, customer.locations)

			cancelled := trip
			cancelled.Status = status
			cancelled.RiderID = &riderID
			cancelled.Status = tripsModels.TripStatusCancelled
			asTripNotifier(t, svc).NotifyTripCancelled(context.Background(), cancelled)

			require.NoError(t, svc.UpdateLocation(context.Background(), riderID, 10.482, -66.905))
			require.Equal(t, 1, customer.locations, "el rider asignado no debe seguir reenviando ubicación")

			var notice struct {
				Type   string `json:"type"`
				TripID string `json:"trip_id"`
			}
			var found bool
			for _, raw := range sent {
				if messageType(t, raw) != "trip_cancelled" {
					continue
				}
				require.NoError(t, json.Unmarshal(raw, &notice))
				found = true
			}
			require.True(t, found)
			require.Equal(t, trip.ID.String(), notice.TripID)
			require.NotContains(t, string(mustJSON(t, notice)), "reason")
		})
	}
}

func TestNotifyTripCancelled_WithdrawsOffersAndSkipsUnrelatedRider(t *testing.T) {
	svc, _, repo := newTripOfferSvc(t)
	trip := samplePickupTrip(t)
	t.Cleanup(func() { app.StopTripOfferTimerForTest(svc, trip.ID) })
	repo.On("SearchActiveOrdersByRiderID", mock.Anything, mock.Anything).Return(nil, errcustom.ErrNotFound)
	repo.On("SearchAvailableOrders", mock.Anything).Return(nil, errcustom.ErrNotFound)

	type watched struct {
		id  uuid.UUID
		got []string
	}
	near := make([]*watched, 3)
	for i := range near {
		w := &watched{id: uuid.New()}
		conn := mocksriders.NewRiderConn(t)
		conn.On("Send", mock.Anything).Run(func(args mock.Arguments) {
			w.got = append(w.got, messageType(t, args.Get(0).([]byte)))
		}).Return()
		near[i] = w
		require.NoError(t, svc.Register(context.Background(), w.id, conn, riderModels.RiderModeTrips))
		require.NoError(t, svc.UpdateLocation(context.Background(), w.id, 10.4806+float64(i)*0.0001, -66.9036))
	}

	farID := uuid.New()
	farConn := mocksriders.NewRiderConn(t)
	farLat, farLng := latLngAtRing(t, *trip.PickupH3Cell, 4)
	require.NoError(t, svc.Register(context.Background(), farID, farConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), farID, farLat, farLng))

	asMatcher(t, svc).OfferTrip(context.Background(), trip)

	cancelled := trip
	cancelled.Status = tripsModels.TripStatusCancelled
	asTripNotifier(t, svc).NotifyTripCancelled(context.Background(), cancelled)

	for _, w := range near {
		require.Equal(t, 1, countKind(w.got, "trip_offer"))
		require.Equal(t, 1, countKind(w.got, "trip_cancelled"))
	}
	farConn.AssertNotCalled(t, "Send", mock.Anything)

	lateID := uuid.New()
	lateConn := mocksriders.NewRiderConn(t)
	require.NoError(t, svc.Register(context.Background(), lateID, lateConn, riderModels.RiderModeTrips))
	require.NoError(t, svc.UpdateLocation(context.Background(), lateID, 10.4806, -66.9036))
	lateConn.AssertNotCalled(t, "Send", mock.Anything)
}

func countKind(kinds []string, want string) int {
	n := 0
	for _, k := range kinds {
		if k == want {
			n++
		}
	}
	return n
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// latLngAtRing is defined in riders_manager_test.go. Keep a local reference
// so this file documents the ring used to place a rider outside the first offer.
var _ = geo.DriverMatchResolution
