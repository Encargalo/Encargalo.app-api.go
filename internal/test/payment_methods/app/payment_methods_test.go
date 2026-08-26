package app_test

import (
	"context"
	"testing"

	mockspm "Encargalo.app-api.go/internal/mocks/payment_methods/domain/ports"
	"Encargalo.app-api.go/internal/payment_methods/app"
	"Encargalo.app-api.go/internal/payment_methods/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetActivePaymentMethods_DelegatesToRepo(t *testing.T) {
	repo := mockspm.NewPaymentMethodsRepo(t)
	svc := app.NewPaymentMethodsApp(repo)

	repo.On("GetActivePaymentMethods", context.Background()).
		Return([]models.PaymentMethod{{ID: uuid.New(), Type: "PagoMovil", Name: "Pago Móvil"}}, nil)

	resp, err := svc.GetActivePaymentMethods(context.Background())

	require.NoError(t, err)
	require.Len(t, resp, 1)
	require.Equal(t, "PagoMovil", resp[0].Type)
}

func TestGetActivePaymentMethods_RepoError(t *testing.T) {
	repo := mockspm.NewPaymentMethodsRepo(t)
	svc := app.NewPaymentMethodsApp(repo)

	repo.On("GetActivePaymentMethods", context.Background()).
		Return(nil, errcustom.ErrUnexpectedError)

	_, err := svc.GetActivePaymentMethods(context.Background())

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}
