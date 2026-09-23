package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestToCustomerDTO_ShowsCodesOnlyToTheCustomer(t *testing.T) {
	pickup := "1111"
	dropoff := "2222"
	trip := Trip{
		Status:      TripStatusAccepted,
		PickupCode:  &pickup,
		DropoffCode: &dropoff,
	}

	require.Nil(t, trip.ToDTO().PickupCode)
	require.Nil(t, trip.ToDTO().DropoffCode)

	accepted := trip.ToCustomerDTO()
	require.Equal(t, pickup, *accepted.PickupCode)
	require.Nil(t, accepted.DropoffCode)

	used := time.Now()
	trip.Status = TripStatusInProgress
	trip.PickupCodeUsedAt = &used
	boarding := trip.ToCustomerDTO()
	require.Nil(t, boarding.PickupCode)
	require.Equal(t, dropoff, *boarding.DropoffCode)
}
