package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	productmodels "Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	usersdto "Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateOrder_CashSkipsProofAndStartsPending(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.quotes.On("TryConsume", mock.Anything, quoteID, mock.Anything).Return(true, nil)
	m.quotes.On("ClearActiveID", mock.Anything, customerID).Return(nil)

	var created *ordermodels.Order
	m.repo.On("CreateOrder", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		created = args.Get(1).(*ordermodels.Order)
	}).Return(&ordermodels.Order{ID: orderID, Status: string(ordermodels.StatusPending)}, nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.CreateOrder(context.Background(), &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "Efectivo",
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, string(ordermodels.StatusPending), created.Status)
	require.Equal(t, string(ordermodels.MethodEfectivo), created.MethodPayment)
	m.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
	m.reviewsRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
