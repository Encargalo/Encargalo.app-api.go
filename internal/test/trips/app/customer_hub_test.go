package app_test

import (
	"context"
	"testing"

	mockstrips "Encargalo.app-api.go/internal/mocks/trips/domain/ports"
	"Encargalo.app-api.go/internal/trips/app"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type captureConn struct {
	msgs [][]byte
}

func (c *captureConn) Send(msg []byte) {
	c.msgs = append(c.msgs, append([]byte(nil), msg...))
}

func TestCustomerHub_PushesActiveTripAndRiderLocation(t *testing.T) {
	repo := mockstrips.NewTripsRepo(t)
	riders := mockstrips.NewRiderPublicInfoLookup(t)
	hub := app.NewCustomerTripHub(repo, riders)

	customerID, tripID := uuid.New(), uuid.New()
	repo.On("ListActiveTripsByCustomer", mock.Anything, customerID).Return([]models.Trip{{
		ID:         tripID,
		CustomerID: customerID,
		Status:     models.TripStatusSearching,
	}}, nil)

	conn := &captureConn{}
	require.NoError(t, hub.Register(context.Background(), customerID, conn))
	require.NotEmpty(t, conn.msgs)
	require.Contains(t, string(conn.msgs[0]), `"type":"trip_update"`)

	hub.NotifyRiderLocation(context.Background(), customerID, tripID, 10.48, -66.90)
	require.Contains(t, string(conn.msgs[len(conn.msgs)-1]), `"type":"rider_location"`)
	require.Contains(t, string(conn.msgs[len(conn.msgs)-1]), tripID.String())
}
