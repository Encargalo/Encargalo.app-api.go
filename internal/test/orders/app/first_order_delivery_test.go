package app_test

import (
	"context"
	"errors"
	"testing"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	productmodels "Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	usersdto "Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func quoteRequest(customerID, shopID, itemID uuid.UUID) *dtos.CreateQuote {
	return &dtos.CreateQuote{
		ShopID:     shopID,
		CustomerID: customerID,
		Address:    dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9},
		Items:      sampleCreateItems(itemID),
	}
}

func TestCreateQuote_FirstOrderGetsFreeDelivery(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(uuid.Nil, nil)
	m.repo.On("HasBillableOrder", mock.Anything, customerID).Return(false, nil)
	wireNoQuoteItemCatalog(m, shopID, itemID, 10)
	var saved *ordermodels.OrderQuote
	m.quotes.On("Save", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		saved = args.Get(1).(*ordermodels.OrderQuote)
	}).Return(nil)
	m.quotes.On("SetActiveID", mock.Anything, customerID, mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.CreateQuote(context.Background(), quoteRequest(customerID, shopID, itemID))

	require.NoError(t, err)
	// 2 unidades a $10; el envío placeholder de $2 lo pone la plataforma.
	assert.Equal(t, 20.0, resp.TotalUSD)
	assert.Equal(t, 0.0, resp.DeliveryFeeUSD)
	assert.Equal(t, 2.0, resp.DeliveryFeeOriginalUSD)
	assert.Equal(t, 2.0, resp.DeliverySubsidyUSD)
	assert.Equal(t, 800.0, resp.TotalBs)
	require.NotNil(t, saved)
	assert.Equal(t, 2.0, saved.DeliverySubsidyUSD, "la quote congela el subsidio")
}

func TestCreateQuote_CustomerWithOrdersPaysDelivery(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(uuid.Nil, nil)
	m.repo.On("HasBillableOrder", mock.Anything, customerID).Return(true, nil)
	wireNoQuoteItemCatalog(m, shopID, itemID, 10)
	m.quotes.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.quotes.On("SetActiveID", mock.Anything, customerID, mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.CreateQuote(context.Background(), quoteRequest(customerID, shopID, itemID))

	require.NoError(t, err)
	assert.Equal(t, 22.0, resp.TotalUSD)
	assert.Equal(t, 2.0, resp.DeliveryFeeUSD)
	assert.Zero(t, resp.DeliverySubsidyUSD)
	assert.Zero(t, resp.DeliveryFeeOriginalUSD)
}

// Si no se puede saber si es el primer pedido, se cobra el envío en vez de fallar.
func TestCreateQuote_LookupFailureChargesDelivery(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(uuid.Nil, nil)
	m.repo.On("HasBillableOrder", mock.Anything, customerID).Return(false, errors.New("db caída"))
	wireNoQuoteItemCatalog(m, shopID, itemID, 10)
	m.quotes.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.quotes.On("SetActiveID", mock.Anything, customerID, mock.Anything, mock.Anything).Return(nil)

	resp, err := svc.CreateQuote(context.Background(), quoteRequest(customerID, shopID, itemID))

	require.NoError(t, err)
	assert.Equal(t, 2.0, resp.DeliveryFeeUSD)
	assert.Zero(t, resp.DeliverySubsidyUSD)
}

// Dos quotes con envío gratis: el índice único deja pasar un solo pedido y la quote rechazada se descarta.
func TestCreateOrder_SecondFreeDeliveryOrderIsRejectedAndQuoteDiscarded(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 20, 0, 40)
	quote.DeliverySubsidyUSD = 2

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.customerBalance.On("GetBalanceBs", mock.Anything, customerID).Return(800.0, nil)
	m.quotes.On("TryConsume", mock.Anything, quoteID, mock.Anything).Return(true, nil)
	m.quotes.On("ClearActiveID", mock.Anything, customerID).Return(nil)
	m.customerBalance.On("DebitForReference", mock.Anything, customerID, orderID, "order", 800.0).Return(nil)
	var inserted *ordermodels.Order
	m.repo.On("CreateOrder", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		inserted = args.Get(1).(*ordermodels.Order)
	}).Return(nil, errcustom.ErrFirstOrderDeliveryUsed)
	m.customerBalance.On("RefundBalancePayment", mock.Anything, customerID, orderID, "order", 800.0).Return(nil)
	m.quotes.On("Delete", mock.Anything, quoteID).Return(nil)

	_, err := svc.CreateOrder(context.Background(), &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil", BalanceAmountBs: 800,
	})

	require.ErrorIs(t, err, errcustom.ErrFirstOrderDeliveryUsed)
	require.NotNil(t, inserted)
	assert.Equal(t, 2.0, inserted.DeliverySubsidyUSD, "el pedido hereda el subsidio de la quote")
	m.quotes.AssertNotCalled(t, "ReleaseConsume", mock.Anything, quoteID)
}

func TestConfirmDelivery_FreeDeliveryBankPaysRiderOnFullFee(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, riderID, shopID := uuid.New(), uuid.New(), uuid.New()
	code := "5678"
	order := &ordermodels.Order{
		ID: orderID, ShopID: shopID, RiderID: &riderID, Status: string(ordermodels.StatusOnTheWay),
		DeliveryCode: &code, MethodPayment: "PagoMovil", TotalPrice: 20, DeliveryFee: 0, DeliverySubsidyUSD: 2,
	}

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("MarkDeliveryCodeUsed", mock.Anything, orderID).Return(nil)
	m.repo.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusCompleted)).Return(nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	m.finance.On("RecordRideEarning", mock.Anything, riderID, orderID, 2.0, 20.0, 40.0, "PagoMovil").Return(nil)
	// El negocio liquida igual: su base es total - envío que pagó el cliente.
	m.shopCommission.On("RecordSaleCommission", mock.Anything, shopID, orderID, 20.0, 0.0, "PagoMovil", false, false, 0.0).Return(nil)
	m.riderMgr.On("BroadcastOrderUpdate", mock.Anything, mock.Anything).Return()
	m.shops.On("TryCloseOpenBatch", mock.Anything, shopID).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.ConfirmDelivery(context.Background(), orderID, code, riderID)

	require.NoError(t, err)
}

// En efectivo el rider cobra $5 y su pago es $1.80: se registra pago - efectivo, sin el tope en 0 de CashOwedUSD.
func TestConfirmDelivery_FreeDeliveryCashUsesCashCollected(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, riderID, shopID := uuid.New(), uuid.New(), uuid.New()
	code := "5678"
	order := &ordermodels.Order{
		ID: orderID, ShopID: shopID, RiderID: &riderID, Status: string(ordermodels.StatusOnTheWay),
		DeliveryCode: &code, MethodPayment: string(ordermodels.MethodEfectivo), TotalPrice: 5, DeliveryFee: 0, DeliverySubsidyUSD: 2,
	}

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(order, nil)
	m.repo.On("MarkDeliveryCodeUsed", mock.Anything, orderID).Return(nil)
	m.repo.On("UpdateOrderStatus", mock.Anything, orderID, string(ordermodels.StatusCompleted)).Return(nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
	m.finance.On("RecordRideCashCollected", mock.Anything, riderID, orderID, 2.0, 5.0, 40.0).Return(nil)
	m.shopCommission.On("RecordSaleCommission", mock.Anything, shopID, orderID, 5.0, 0.0, string(ordermodels.MethodEfectivo), false, false, 0.0).Return(nil)
	m.riderMgr.On("BroadcastOrderUpdate", mock.Anything, mock.Anything).Return()
	m.shops.On("TryCloseOpenBatch", mock.Anything, shopID).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.ConfirmDelivery(context.Background(), orderID, code, riderID)

	require.NoError(t, err)
}
