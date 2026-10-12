package dtos

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateTripRequest_AcceptsV4AndV7(t *testing.T) {
	v7, err := uuid.NewV7()
	require.NoError(t, err)

	req := CreateTripRequest{ID: v7, QuoteID: uuid.New(), MethodPayment: "Efectivo"}
	require.NoError(t, req.Validate())

	req.ID = uuid.New()
	req.QuoteID = v7
	require.NoError(t, req.Validate())
}
