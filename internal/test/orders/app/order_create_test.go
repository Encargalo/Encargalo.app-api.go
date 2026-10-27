package app_test

import (
	"context"
	"testing"
	"time"

	bcvmodels "Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	ordermodels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/pkg/geo"
	productmodels "Encargalo.app-api.go/internal/products/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	usersdto "Encargalo.app-api.go/internal/users/domain/dto"
	usersmodels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func sampleOpenShop(shopID uuid.UUID) *ordermodels.ShopOrder {
	return &ordermodels.ShopOrder{ID: shopID, Opened: true, Type: "Restaurant"}
}

func sampleCatalogItem(itemID uuid.UUID, price float64) productmodels.Item {
	return productmodels.Item{ID: itemID, Price: price, AvailabilityStatus: productmodels.AvailabilityActive}
}

func sampleAccount(customerID uuid.UUID) *usersmodels.Account {
	return &usersmodels.Account{ID: customerID, Name: "Juan"}
}

// wireNoQuoteItemCatalog configura las expectativas de repo/shops/bcv comunes
// al camino sin quote (priceOrderDraft), usado por CreateQuote y por
// CreateOrder cuando order.QuoteID es uuid.Nil.
func wireNoQuoteItemCatalog(m *orderAppMocks, shopID, itemID uuid.UUID, price float64) {
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, price)}, nil)
	m.repo.On("SearchAdditionsByID", mock.Anything, mock.Anything).Return([]ordermodels.Addition{}, nil)
	m.bcv.On("GetRate", mock.Anything).Return(&bcvmodels.ExchangeRate{Rate: 40}, nil)
}

func sampleCreateItems(itemID uuid.UUID) []dtos.CreateItemsOrder {
	return []dtos.CreateItemsOrder{{ItemID: itemID, Amount: 2}}
}

func TestCreateQuote_Success(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(uuid.Nil, nil)
	wireNoQuoteItemCatalog(m, shopID, itemID, 10)
	m.repo.On("HasBillableOrder", mock.Anything, customerID).Return(true, nil)
	m.quotes.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.quotes.On("SetActiveID", mock.Anything, customerID, mock.Anything, mock.Anything).Return(nil)

	req := &dtos.CreateQuote{
		ShopID:     shopID,
		CustomerID: customerID,
		Address:    dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9},
		Items:      sampleCreateItems(itemID),
	}

	resp, err := svc.CreateQuote(context.Background(), req)

	require.NoError(t, err)
	// 2 unidades a $10 + delivery fee placeholder ($2) = $22.
	assert.Equal(t, 22.0, resp.TotalUSD)
	assert.Equal(t, 2.0, resp.DeliveryFeeUSD)
	assert.NotEqual(t, uuid.Nil, resp.QuoteID)
}

// El ETL resincroniza CubeDeliveryQuote por evento, no por el batch horario.
func TestCreateQuote_PublishesDeliveryQuoteCreated(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(uuid.Nil, nil)
	wireNoQuoteItemCatalog(m, shopID, itemID, 10)
	m.repo.On("HasBillableOrder", mock.Anything, customerID).Return(true, nil)
	m.quotes.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.quotes.On("SetActiveID", mock.Anything, customerID, mock.Anything, mock.Anything).Return(nil)

	req := &dtos.CreateQuote{
		ShopID:     shopID,
		CustomerID: customerID,
		Address:    dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9},
		Items:      sampleCreateItems(itemID),
	}

	resp, err := svc.CreateQuote(context.Background(), req)

	require.NoError(t, err)
	m.stream.AssertCalled(t, "PublisherEvent", mock.Anything, ordermodels.DataEventDeliveryQuoteCreated{
		MessageType: "delivery_quote_created",
		QuoteID:     resp.QuoteID.String(),
	})
}

func TestCreateQuote_ShopClosed(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(uuid.Nil, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(&ordermodels.ShopOrder{ID: shopID, Opened: false}, nil)

	req := &dtos.CreateQuote{
		ShopID:     shopID,
		CustomerID: customerID,
		Address:    dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9},
		Items:      sampleCreateItems(itemID),
	}

	_, err := svc.CreateQuote(context.Background(), req)

	assert.ErrorIs(t, err, errcustom.ErrShopClosed)
}

func TestCreateQuote_ReusesActiveQuote(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	customerID, shopID, itemID := uuid.New(), uuid.New(), uuid.New()
	activeQuoteID := uuid.New()
	address := dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9}
	items := sampleCreateItems(itemID)
	fingerprint := ordermodels.CartFingerprint(shopID, address, items)

	existing := &ordermodels.OrderQuote{
		QuoteID:         activeQuoteID,
		CustomerID:      customerID,
		ShopID:          shopID,
		CartFingerprint: fingerprint,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		TotalUSD:        22,
	}

	m.quotes.On("GetActiveID", mock.Anything, customerID).Return(activeQuoteID, nil)
	m.quotes.On("Get", mock.Anything, activeQuoteID).Return(existing, nil)
	m.quotes.On("IsConsumed", mock.Anything, activeQuoteID).Return(false, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)

	req := &dtos.CreateQuote{ShopID: shopID, CustomerID: customerID, Address: address, Items: items}

	resp, err := svc.CreateQuote(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, activeQuoteID, resp.QuoteID)
	assert.Equal(t, 22.0, resp.TotalUSD)
}

func TestCreateOrder_AlreadyExists(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID := uuid.New()

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(&ordermodels.Order{ID: orderID}, nil)

	_, err := svc.CreateOrder(context.Background(), &dtos.CreateOrder{ID: orderID})

	assert.ErrorIs(t, err, errcustom.ErrOrderAlreadyExists)
}

func TestCreateOrder_PriceMismatch(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	wireNoQuoteItemCatalog(m, shopID, itemID, 10)
	m.repo.On("HasBillableOrder", mock.Anything, customerID).Return(true, nil)

	order := &dtos.CreateOrder{
		ID:                 orderID,
		ShopID:             shopID,
		CustomerID:         customerID,
		MethodPayment:      "PagoMovil",
		Address:            dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9},
		CreateItemsOrder:   sampleCreateItems(itemID),
		ExpectedTotalPrice: 999,
	}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrPriceChanged)
}

// sampleQuoteForOrder arma un OrderQuote vigente listo para ApplyQuoteSnapshot,
// evitando el camino priceOrderDraft (que ejercita SearchItemsByID/SearchAdditionsByID
// con un ItemsOrders construido desde dtos.CreateItemsOrder en vez de desde el
// quote). ApplyQuoteSnapshot no setea Additions cuando el line no trae
// adiciones, así que este camino no necesita mockear SearchAdditionsByID.
func sampleQuoteForOrder(quoteID, customerID, shopID, itemID uuid.UUID, totalUSD, deliveryFeeUSD, bcvRate float64) *ordermodels.OrderQuote {
	now := time.Now()
	return &ordermodels.OrderQuote{
		QuoteID:         quoteID,
		CustomerID:      customerID,
		ShopID:          shopID,
		Address:         dtos.Address{Address: "Calle 1", Latitude: 10.48, Longitude: -66.9},
		TotalUSD:        totalUSD,
		DeliveryFeeUSD:  deliveryFeeUSD,
		BcvRate:         bcvRate,
		CreatedAt:       now,
		ExpiresAt:       now.Add(time.Hour),
		ProofGraceUntil: now.Add(2 * time.Hour),
		Lines: []ordermodels.QuoteLineSnapshot{
			{ItemID: itemID, Name: "Item", Amount: 2, UnitPrice: 10, TotalPrice: 20},
		},
	}
}

func TestCreateOrder_WithQuote_BalanceCoversFull(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.customerBalance.On("GetBalanceBs", mock.Anything, customerID).Return(880.0, nil)
	m.quotes.On("TryConsume", mock.Anything, quoteID, mock.Anything).Return(true, nil)
	m.quotes.On("ClearActiveID", mock.Anything, customerID).Return(nil)
	m.customerBalance.On("DebitForReference", mock.Anything, customerID, orderID, "order", 880.0).Return(nil)
	m.repo.On("CreateOrder", mock.Anything, mock.Anything).Return(&ordermodels.Order{
		ID: orderID, ShopID: shopID, CustomerID: customerID, TotalPrice: 22, DeliveryFee: 2,
		MethodPayment: "PagoMovil", Status: string(ordermodels.StatusPending),
	}, nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	order := &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil",
		BalanceAmountBs: 880,
	}

	orderDB, err := svc.CreateOrder(context.Background(), order)

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusPending), orderDB.Status)
}

func TestCreateOrder_WithQuote_RequiresPaymentProof(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.upload.On("UploadImage", mock.Anything, "data:image/jpeg;base64,abc").Return(&uploader.UploadResult{SecureURL: "https://cloudinary/proof.jpg"}, nil)
	m.quotes.On("TryConsume", mock.Anything, quoteID, mock.Anything).Return(true, nil)
	m.quotes.On("ClearActiveID", mock.Anything, customerID).Return(nil)
	m.repo.On("CreateOrder", mock.Anything, mock.Anything).Return(&ordermodels.Order{
		ID: orderID, ShopID: shopID, CustomerID: customerID, TotalPrice: 22, DeliveryFee: 2,
		MethodPayment: "PagoMovil", Status: string(ordermodels.StatusPaymentReview),
	}, nil)
	m.reviewsRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	m.staffHub.On("BroadcastReview", mock.Anything).Return(nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	order := &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil",
		PaymentProofImage: "data:image/jpeg;base64,abc",
	}

	orderDB, err := svc.CreateOrder(context.Background(), order)

	require.NoError(t, err)
	assert.Equal(t, string(ordermodels.StatusPaymentReview), orderDB.Status)
}

func TestCreateOrder_ProofRequiredButMissing(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)

	order := &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil",
	}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrInvalidBalanceAmount)
}

func TestCreateOrder_InsufficientBalance(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.customerBalance.On("GetBalanceBs", mock.Anything, customerID).Return(100.0, nil)

	order := &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil",
		BalanceAmountBs: 500,
	}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrInsufficientBalance)
}

func TestCreateOrder_QuoteExpired(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)
	quote.ExpiresAt = time.Now().Add(-time.Hour)
	quote.ProofGraceUntil = time.Now().Add(-time.Minute)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)

	order := &dtos.CreateOrder{ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil"}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrQuoteExpired)
}

func TestCreateOrder_QuoteForbidden(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, uuid.New() /* dueño distinto */, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)

	order := &dtos.CreateOrder{ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil"}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrQuoteForbidden)
}

// TestCreateOrder_WithQuote_ProductBecameUnavailable cubre validateOrderCatalogAvailability:
// aunque el precio ya fue congelado en el quote, CreateOrder re-valida el
// catálogo en vivo (el negocio pudo ocultar/agotar el item entre el quote y el pago).
func TestCreateOrder_WithQuote_ProductBecameUnavailable(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	unavailable := sampleCatalogItem(itemID, 10)
	unavailable.AvailabilityStatus = productmodels.AvailabilityOutOfStock
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{unavailable}, nil)

	order := &dtos.CreateOrder{ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil"}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrProductUnavailable)
}

func TestCreateOrder_ComputesH3CellFromAddress(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 40)
	wantCell, err := geo.LatLngToCell(quote.Address.Latitude, quote.Address.Longitude, geo.ZoneAnalyticsResolution)
	require.NoError(t, err)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)
	m.customerBalance.On("GetBalanceBs", mock.Anything, customerID).Return(880.0, nil)
	m.quotes.On("TryConsume", mock.Anything, quoteID, mock.Anything).Return(true, nil)
	m.quotes.On("ClearActiveID", mock.Anything, customerID).Return(nil)
	m.customerBalance.On("DebitForReference", mock.Anything, customerID, orderID, "order", 880.0).Return(nil)
	m.repo.On("CreateOrder", mock.Anything, mock.MatchedBy(func(o *ordermodels.Order) bool {
		return o.H3Cell == wantCell
	})).Return(&ordermodels.Order{
		ID: orderID, ShopID: shopID, CustomerID: customerID, TotalPrice: 22, DeliveryFee: 2,
		MethodPayment: "PagoMovil", Status: string(ordermodels.StatusPending),
	}, nil)
	m.stream.On("PublisherEvent", mock.Anything, mock.Anything).Return(nil)

	order := &dtos.CreateOrder{
		ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil",
		BalanceAmountBs: 880,
	}

	_, err = svc.CreateOrder(context.Background(), order)

	require.NoError(t, err)
	m.repo.AssertExpectations(t)
}

func TestCreateOrder_BcvRateUnavailable(t *testing.T) {
	svc, m := newOrderAppFixture(t)
	orderID, quoteID, shopID, customerID, itemID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	quote := sampleQuoteForOrder(quoteID, customerID, shopID, itemID, 22, 2, 0)

	m.repo.On("SearchOrdersByID", mock.Anything, orderID).Return(nil, errcustom.ErrNotFound)
	m.quotes.On("Get", mock.Anything, quoteID).Return(quote, nil)
	m.repo.On("SearchShopByID", mock.Anything, shopID).Return(sampleOpenShop(shopID), nil)
	m.shops.On("SearchBatchByShopID", mock.Anything, shopID).Return(uuid.New(), nil)
	m.repo.On("SearchItemsByID", mock.Anything, mock.Anything).Return(productmodels.Items{sampleCatalogItem(itemID, 10)}, nil)
	m.users.On("SearchCustomerBy", mock.Anything, usersdto.SearchCustomerBy{ID: customerID}).Return(sampleAccount(customerID), nil)

	order := &dtos.CreateOrder{ID: orderID, QuoteID: quoteID, CustomerID: customerID, MethodPayment: "PagoMovil"}

	_, err := svc.CreateOrder(context.Background(), order)

	assert.ErrorIs(t, err, errcustom.ErrBcvRateUnavailable)
}
