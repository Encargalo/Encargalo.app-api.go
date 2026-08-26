package app_test

import (
	"context"
	"testing"

	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStaffRefundHub_RegisterWithoutFinanceApp(t *testing.T) {
	hub := app.NewStaffRefundHub()
	conn := mockports.NewRefundWSConn(t)

	err := hub.Register(context.Background(), uuid.New(), conn)

	require.NoError(t, err)
}

func TestStaffRefundHub_RegisterSendsActiveSnapshot(t *testing.T) {
	hub := app.NewStaffRefundHub()
	finance := mockports.NewCustomerFinanceApp(t)
	hub.SetFinanceApp(finance)
	conn := mockports.NewRefundWSConn(t)

	finance.On("ListActiveForStaff", mock.Anything).Return([]dto.RefundRequestBroadcast{
		{ID: uuid.New().String(), Status: "pending"},
	}, nil)
	conn.On("Send", mock.Anything).Return()

	err := hub.Register(context.Background(), uuid.New(), conn)

	require.NoError(t, err)
	conn.AssertCalled(t, "Send", mock.Anything)
}

func TestStaffRefundHub_RegisterPropagatesListError(t *testing.T) {
	hub := app.NewStaffRefundHub()
	finance := mockports.NewCustomerFinanceApp(t)
	hub.SetFinanceApp(finance)
	conn := mockports.NewRefundWSConn(t)

	finance.On("ListActiveForStaff", mock.Anything).Return(nil, errcustom.ErrUnexpectedError)

	err := hub.Register(context.Background(), uuid.New(), conn)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestStaffRefundHub_UnregisterRemovesConnection(t *testing.T) {
	hub := app.NewStaffRefundHub()
	conn := mockports.NewRefundWSConn(t)
	staffID := uuid.New()

	conn.On("Close").Return()
	require.NoError(t, hub.Register(context.Background(), staffID, conn))

	hub.Unregister(staffID, conn)

	conn.AssertCalled(t, "Close")
}

func TestStaffRefundHub_UnregisterUnknownStaffClosesConn(t *testing.T) {
	hub := app.NewStaffRefundHub()
	conn := mockports.NewRefundWSConn(t)
	conn.On("Close").Return()

	hub.Unregister(uuid.New(), conn)

	conn.AssertCalled(t, "Close")
}

func TestStaffRefundHub_BroadcastSendsToAllConnections(t *testing.T) {
	hub := app.NewStaffRefundHub()
	conn1 := mockports.NewRefundWSConn(t)
	conn2 := mockports.NewRefundWSConn(t)
	conn1.On("Send", mock.Anything).Return()
	conn2.On("Send", mock.Anything).Return()

	require.NoError(t, hub.Register(context.Background(), uuid.New(), conn1))
	require.NoError(t, hub.Register(context.Background(), uuid.New(), conn2))

	err := hub.Broadcast(dto.RefundRequestBroadcast{ID: uuid.New().String(), Status: "pending"})

	require.NoError(t, err)
	conn1.AssertCalled(t, "Send", mock.Anything)
	conn2.AssertCalled(t, "Send", mock.Anything)
}
