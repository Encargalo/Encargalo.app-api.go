package dtos

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateItemsOrder_AcceptsV4AndV7(t *testing.T) {
	v7, err := uuid.NewV7()
	require.NoError(t, err)

	item := CreateItemsOrder{ItemID: v7, Amount: 1}
	require.NoError(t, item.Validate())

	item.ItemID = uuid.New()
	require.NoError(t, item.Validate())
}
