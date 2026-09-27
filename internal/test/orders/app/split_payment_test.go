package app_test

import (
	"context"
	"encoding/json"
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

func TestCreateOrder_CashAndBalanceStartsPending(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.customerBalance.On("GetBalanceBs", mock.Anything, customerID).Return(1000.0, nil)
	m.quotes.On("TryConsume", mock.Anything, quoteID, mock.Anything).Return(true, nil)
	m.quotes.On("ClearActiveID", mock.Anything, customerID).Return(nil)
	m.customerBalance.On("DebitForReference", mock.Anything, customerID, orderID, "order", 400.0).Return(nil)

	var created *ordermodels.Order
	m.repo.On("CreateOrder", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		created = args.Get(1).(*ordermodels.Order)
	}).Return(&ordermodels.Order{ID: orderID, Status: string(ordermodels.StatusPending)}, nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.CreateOrder(context.Background(), &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID,
		MethodPayment: "Efectivo", BalanceAmountBs: 400,
	})

	require.NoError(t, err)
	require.Equal(t, string(ordermodels.StatusPending), created.Status)
	var details ordermodels.Details
	require.NoError(t, json.Unmarshal(created.Details, &details))
	require.Equal(t, 400.0, details.PaymentDetails.BalanceAmountBs)
	require.Equal(t, 480.0, details.PaymentDetails.CashAmountBs)
	require.Equal(t, 0.0, details.PaymentDetails.PagoMovilAmountBs)
	m.upload.AssertNotCalled(t, "UploadImage", mock.Anything, mock.Anything)
}

func TestCreateOrder_ThreeTendersRejected(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.customerBalance.On("GetBalanceBs", mock.Anything, customerID).Return(1000.0, nil)

	_, err := svc.CreateOrder(context.Background(), &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID,
		MethodPayment: "PagoMovil", BalanceAmountBs: 200, CashAmountBs: 200,
		PaymentProofImage: "data:image/jpeg;base64,abc",
	})

	require.ErrorIs(t, err, errcustom.ErrInvalidBalanceAmount)
}
