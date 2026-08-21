package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strings"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/pkg/geo"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	productsModels "Encargalo.app-api.go/internal/products/domain/models"
	productsPorts "Encargalo.app-api.go/internal/products/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	shops "Encargalo.app-api.go/internal/shops/domain/ports"
	"Encargalo.app-api.go/internal/users/domain/dto"
	portsCustomers "Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

// priceTolerance absorbe errores de redondeo de flotantes al comparar el
// total recalculado en el servidor contra lo que el front mostró al cliente.
const priceTolerance = 0.01

// PriceMismatch indica si el total recalculado en el servidor difiere del
// que el front mostró al cliente más allá de lo que puede explicar el
// redondeo de flotantes (ver priceTolerance). Exportada como función pura
// para poder testearla sin depender de red (deliveryfee.Calculate, más
// arriba en CreateOrder, hace una llamada real a Google Maps que no se
// puede mockear sin refactorizarla a un port inyectable).
func (o *orderApp) resolveOrderItemFlavors(ctx context.Context, order *models.Order) error {
	for i := range order.ItemsOrders {
		itemOrder := &order.ItemsOrders[i]
		chosenIDs := itemOrder.GetFlavorIDs()
		if len(chosenIDs) == 0 {
			continue
		}

		catalog, err := o.repo.SearchFlavorsForItem(ctx, itemOrder.ItemID)
		if err != nil {
			return err
		}

		catalogByID := make(map[uuid.UUID]models.Flavor, len(catalog))
		for _, flavor := range catalog {
			catalogByID[flavor.ID] = flavor
		}

		resolved := make([]models.Flavor, 0, len(chosenIDs))
		for _, id := range chosenIDs {
			flavor, ok := catalogByID[id]
			if !ok {
				return errcustom.ErrInvalidFlavors
			}
			if !flavor.IsAvailable() {
				return errcustom.ErrProductUnavailable
			}
			resolved = append(resolved, flavor)
		}

		itemOrder.SetFlavorsDetails(resolved)
	}

	return nil
}

func PriceMismatch(actual, expected float64) bool {
	return math.Abs(actual-expected) > priceTolerance
}

// ValidateTurboCoverage rechaza pedidos/cotizaciones a shops Turbo cuando la
// dirección del cliente cae fuera del radio de cobertura (2 km). No-op para
// shops que no son Turbo. Exportada como función pura para poder testearla
// sin depender de red (ver comentario de PriceMismatch, arriba).
func ValidateTurboCoverage(shop *models.ShopOrder, address dtos.Address) error {
	if shop.Type != shopsmodels.ShopTypeTurbo {
		return nil
	}
	distanceKm := geo.HaversineKm(shop.Latitude, shop.Longitude, address.Latitude, address.Longitude)
	if distanceKm > shopsmodels.TurboCoverageRadiusKm {
		return errcustom.ErrTurboOutOfCoverage
	}
	return nil
}

var allowedTransitionStatuses = map[models.OrderStatus]bool{
	models.StatusAccepted:      true,
	models.StatusInPreparation: true,
	models.StatusReady:         true,
	models.StatusOnTheWay:      true,
	models.StatusCompleted:     true,
	models.StatusRejected:      true,
}

var paymentAutoTransitions = map[models.PaymentMethod]map[models.OrderStatus]models.OrderStatus{
	models.MethodPagoMovil: {models.StatusAccepted: models.StatusInPreparation},
}

type statusHandler func(ctx context.Context, order models.Order) error

type orderApp struct {
	repo            ports.OrdersRepo
	shops           shops.ShopsRepo
	custo           portsCustomers.UsersRepo
	stream          ports.RedisStream
	uploadImage     uploadimage.UploadImage
	riderMgr        ports.OrderRiderBroadcaster
	finance         ports.RideEarningRecorder
	shopCommission  ports.ShopCommissionRecorder
	bcv             bcvPorts.BcvApp
	reviewsRepo     ports.PaymentReviewsRepo
	staffHub        ports.StaffReviewHub
	customerBalance ports.CustomerBalanceCreditor
	quotes          ports.OrderQuotesRepo
	products        productsPorts.ProductsRepo
	statusHandlers  map[models.OrderStatus]statusHandler
}

func NewOrderApp(repo ports.OrdersRepo, shops shops.ShopsRepo, custo portsCustomers.UsersRepo, stream ports.RedisStream, uploadImage uploadimage.UploadImage, riderMgr ports.OrderRiderBroadcaster, finance ports.RideEarningRecorder, shopCommission ports.ShopCommissionRecorder, bcv bcvPorts.BcvApp, reviewsRepo ports.PaymentReviewsRepo, staffHub ports.StaffReviewHub, customerBalance ports.CustomerBalanceCreditor, quotes ports.OrderQuotesRepo, products productsPorts.ProductsRepo) ports.OrdersApp {
	o := &orderApp{
		repo:            repo,
		shops:           shops,
		custo:           custo,
		stream:          stream,
		uploadImage:     uploadImage,
		riderMgr:        riderMgr,
		finance:         finance,
		shopCommission:  shopCommission,
		bcv:             bcv,
		reviewsRepo:     reviewsRepo,
		staffHub:        staffHub,
		customerBalance: customerBalance,
		quotes:          quotes,
		products:        products,
	}
	o.statusHandlers = map[models.OrderStatus]statusHandler{
		models.StatusInPreparation: func(ctx context.Context, order models.Order) error {
			o.generateAndSavePickupCode(ctx, order.ID)
			o.riderMgr.BroadcastNewOrder(ctx, order)
			return nil
		},
	}
	return o
}

// bcvRate resuelve la tasa BCV vigente; si el proveedor falla, se ignora y
// los montos en bolívares del recibo quedan en 0 en vez de romper el checkout.
func (o *orderApp) bcvRate(ctx context.Context) float64 {
	rate, err := o.bcv.GetRate(ctx)
	if err != nil {
		slog.Warn("bcv rate unavailable, order details bs prices will be 0", "error", err)
		return 0
	}
	return rate.Rate
}

func (o *orderApp) CreateQuote(ctx context.Context, req *dtos.CreateQuote) (*dtos.QuoteResponse, error) {
	fingerprint := models.CartFingerprint(req.ShopID, req.Address, req.Items)

	if existing, ok := o.reusableActiveQuote(ctx, req.CustomerID, fingerprint); ok {
		shop, err := o.repo.SearchShopByID(ctx, existing.ShopID)
		if err != nil {
			return nil, err
		}
		if !shop.Opened {
			return nil, errcustom.ErrShopClosed
		}
		return quoteResponseFromModel(existing), nil
	}

	priced, err := o.priceOrderDraft(ctx, &dtos.CreateOrder{
		ID:               uuid.New(),
		ShopID:           req.ShopID,
		CustomerID:       req.CustomerID,
		MethodPayment:    "PagoMovil",
		Address:          req.Address,
		CreateItemsOrder: req.Items,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	quoteID := uuid.New()
	quote := &models.OrderQuote{
		QuoteID:            quoteID,
		CustomerID:         req.CustomerID,
		ShopID:             req.ShopID,
		Address:            req.Address,
		Items:              append([]dtos.CreateItemsOrder(nil), req.Items...),
		TotalUSD:           money.RoundUSD(priced.order.TotalPrice),
		TotalBs:            money.RoundBs(priced.order.TotalPrice * priced.rate),
		DeliveryFeeUSD:     money.RoundUSD(priced.order.DeliveryFee),
		DeliveryFeeBs:      money.RoundBs(priced.order.DeliveryFee * priced.rate),
		BcvRate:            priced.rate,
		IsLaunchPromo:      priced.order.IsLaunchPromo,
		PromoSubsidyAmount: priced.order.PromoSubsidyAmount,
		IsZeroCommission:   priced.order.IsZeroCommission,
		Lines:              models.SnapshotLinesFromOrder(priced.order),
		CartFingerprint:    fingerprint,
		CreatedAt:          now,
		ExpiresAt:          now.Add(models.QuotePaymentTTL),
		ProofGraceUntil:    now.Add(models.QuotePaymentTTL + models.QuoteProofGrace),
	}

	ttl := models.QuotePaymentTTL + models.QuoteProofGrace
	if err := o.replaceActiveQuote(ctx, req.CustomerID, quote, ttl); err != nil {
		return nil, err
	}

	return quoteResponseFromModel(quote), nil
}

// reusableActiveQuote returns the customer's current quote when it is still
// within expires_at, not consumed, and matches the same cart fingerprint.
func (o *orderApp) reusableActiveQuote(ctx context.Context, customerID uuid.UUID, fingerprint string) (*models.OrderQuote, bool) {
	activeID, err := o.quotes.GetActiveID(ctx, customerID)
	if err != nil || activeID == uuid.Nil {
		return nil, false
	}
	quote, err := o.quotes.Get(ctx, activeID)
	if err != nil {
		_ = o.quotes.ClearActiveID(ctx, customerID)
		return nil, false
	}
	consumed, err := o.quotes.IsConsumed(ctx, activeID)
	if err != nil || consumed {
		_ = o.quotes.ClearActiveID(ctx, customerID)
		return nil, false
	}
	now := time.Now()
	if now.After(quote.ExpiresAt) {
		return nil, false
	}
	if quote.CustomerID != customerID || quote.CartFingerprint != fingerprint {
		return nil, false
	}
	return quote, true
}

func (o *orderApp) replaceActiveQuote(ctx context.Context, customerID uuid.UUID, quote *models.OrderQuote, ttl time.Duration) error {
	if prevID, err := o.quotes.GetActiveID(ctx, customerID); err == nil && prevID != uuid.Nil && prevID != quote.QuoteID {
		_ = o.quotes.Delete(ctx, prevID)
	}
	if err := o.quotes.Save(ctx, quote, ttl); err != nil {
		return err
	}
	return o.quotes.SetActiveID(ctx, customerID, quote.QuoteID, ttl)
}

func quoteResponseFromModel(quote *models.OrderQuote) *dtos.QuoteResponse {
	itemsView := make([]dtos.QuoteItemView, 0, len(quote.Lines))
	for _, line := range quote.Lines {
		itemsView = append(itemsView, dtos.QuoteItemView{
			ItemID:     line.ItemID,
			Name:       line.Name,
			Amount:     line.Amount,
			UnitPrice:  line.UnitPrice,
			TotalPrice: line.TotalPrice,
		})
	}
	return &dtos.QuoteResponse{
		QuoteID:            quote.QuoteID,
		TotalUSD:           quote.TotalUSD,
		TotalBs:            quote.TotalBs,
		DeliveryFeeUSD:     quote.DeliveryFeeUSD,
		DeliveryFeeBs:      quote.DeliveryFeeBs,
		BcvRate:            quote.BcvRate,
		IsLaunchPromo:      quote.IsLaunchPromo,
		PromoSubsidyAmount: quote.PromoSubsidyAmount,
		ExpiresAt:          quote.ExpiresAt,
		ProofGraceUntil:    quote.ProofGraceUntil,
		Items:              itemsView,
	}
}

type pricedOrderDraft struct {
	order *models.Order
	shop  *models.ShopOrder
	rate  float64
}

// priceOrderDraft calcula totales (items, delivery, promo) sin persistir ni cobrar.
func (o *orderApp) priceOrderDraft(ctx context.Context, order *dtos.CreateOrder) (*pricedOrderDraft, error) {
	shop, batchID, err := o.requireOpenShopAndBatch(ctx, order.ShopID)
	if err != nil {
		return nil, err
	}

	if err := ValidateTurboCoverage(shop, order.Address); err != nil {
		return nil, err
	}

	orderModel := new(models.Order)
	orderModel.BuildDtoToModel(*order, batchID)

	var fee float64
	if config.Get().Environment.Current == "production" {
		fee, err = deliveryfee.Calculate(shop.Latitude, shop.Longitude, order.Address.Latitude, order.Address.Longitude)
		if err != nil {
			slog.Error("calcDeliveryFee: google maps failed", "shopID", shop.ID, "error", err)
			return nil, errcustom.ErrUnexpectedError
		}
	} else {
		fee = deliveryfee.DevPlaceholderFee
	}
	orderModel.DeliveryFee = fee

	items, err := o.repo.SearchItemsByID(ctx, orderModel.GetItemsID())
	if err != nil {
		return nil, err
	}

	if err := validateOrderCatalogAvailability(items, orderModel); err != nil {
		return nil, err
	}

	orderModel.SetPrices(items)
	orderModel.SetItemName(items)
	orderModel.SetItemImage(items)

	for _, v := range orderModel.ItemsOrders {
		if v.Additions != nil {
			additions, err := o.repo.SearchAdditionsByID(ctx, v.GetAdditionsID())
			if err != nil {
				return nil, err
			}
			orderModel.SetAdditionalsPrices(additions)
			orderModel.SetAdditionalsName(additions)
		}
	}

	if flavorIDs := orderModel.GetFlavorsID(); len(flavorIDs) > 0 {
		if err := o.resolveOrderItemFlavors(ctx, orderModel); err != nil {
			return nil, err
		}
	}

	orderModel.ApplyLaunchPromo(shop.HasActiveLaunchPromo())
	orderModel.ApplyZeroCommission(shop.HasActiveZeroCommission())

	return &pricedOrderDraft{
		order: orderModel,
		shop:  shop,
		rate:  o.bcvRate(ctx),
	}, nil
}

// requireOpenShopAndBatch valida que el local esté abierto y resuelve el lote
// de la jornada. Si el local está abierto sin lote, es inconsistencia (500).
func (o *orderApp) requireOpenShopAndBatch(ctx context.Context, shopID uuid.UUID) (*models.ShopOrder, uuid.UUID, error) {
	shop, err := o.repo.SearchShopByID(ctx, shopID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if !shop.Opened {
		return nil, uuid.Nil, errcustom.ErrShopClosed
	}

	batchID, err := o.shops.SearchBatchByShopID(ctx, shopID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			slog.Error("shop open without open batch", "shopID", shopID)
			return nil, uuid.Nil, errcustom.ErrUnexpectedError
		}
		return nil, uuid.Nil, err
	}
	return shop, batchID, nil
}

func (o *orderApp) tryCloseShopBatch(ctx context.Context, shopID uuid.UUID) {
	if err := o.shops.TryCloseOpenBatch(ctx, shopID); err != nil {
		slog.Error("tryCloseShopBatch failed", "shopID", shopID, "error", err)
	}
}

func (o *orderApp) loadQuoteForCreate(ctx context.Context, order *dtos.CreateOrder) (*models.OrderQuote, error) {
	quote, err := o.quotes.Get(ctx, order.QuoteID)
	if err != nil {
		return nil, err
	}
	if quote.CustomerID != order.CustomerID {
		return nil, errcustom.ErrQuoteForbidden
	}
	hasProof := strings.TrimSpace(order.PaymentProofImage) != ""
	if !quote.AcceptableForCreate(time.Now(), hasProof) {
		return nil, errcustom.ErrQuoteExpired
	}
	return quote, nil
}

func (o *orderApp) CreateOrder(ctx context.Context, order *dtos.CreateOrder) (*models.Order, error) {

	ord, err := o.SearchOrdersByID(ctx, order.ID)
	if ord != nil {
		return &models.Order{}, errcustom.ErrOrderAlreadyExists
	}

	if err != nil {
		if !errors.Is(err, errcustom.ErrNotFound) {
			return &models.Order{}, err
		}
	}

	var (
		quote      *models.OrderQuote
		orderModel *models.Order
		shop       *models.ShopOrder
		rate       float64
	)

	if order.QuoteID != uuid.Nil {
		quote, err = o.loadQuoteForCreate(ctx, order)
		if err != nil {
			return &models.Order{}, err
		}
		order.ShopID = quote.ShopID
		order.Address = quote.Address
		order.CreateItemsOrder = quote.Items

		var batchID uuid.UUID
		shop, batchID, err = o.requireOpenShopAndBatch(ctx, order.ShopID)
		if err != nil {
			return &models.Order{}, err
		}

		orderModel = new(models.Order)
		orderModel.BuildDtoToModel(*order, batchID)
		models.ApplyQuoteSnapshot(orderModel, quote)
		rate = quote.BcvRate

		items, err := o.repo.SearchItemsByID(ctx, orderModel.GetItemsID())
		if err != nil {
			return &models.Order{}, err
		}
		if err := validateOrderCatalogAvailability(items, orderModel); err != nil {
			return &models.Order{}, err
		}
	} else {
		priced, err := o.priceOrderDraft(ctx, order)
		if err != nil {
			return &models.Order{}, err
		}
		orderModel = priced.order
		shop = priced.shop
		rate = priced.rate

		if PriceMismatch(orderModel.TotalPrice, order.ExpectedTotalPrice) {
			slog.Warn("price mismatch detected at checkout", "shopID", shop.ID, "customerID", order.CustomerID,
				"expected", order.ExpectedTotalPrice, "actual", orderModel.TotalPrice)
			return &models.Order{}, errcustom.ErrPriceChanged
		}
	}

	customer, err := o.custo.SearchCustomerBy(ctx, dto.SearchCustomerBy{ID: order.CustomerID})
	if err != nil {
		return &models.Order{}, err
	}

	// rate es la tasa BCV congelada del quote (o recién resuelta si no hubo
	// quote): es la única tasa válida para convertir el balance en Bs que el
	// customer pide usar al USD del total de la orden — la misma con la que
	// se calcularon TotalBs/DeliveryFeeBs que el customer vio antes de pagar.
	if rate <= 0 {
		return &models.Order{}, errcustom.ErrBcvRateUnavailable
	}

	balanceUsedBs := money.RoundBs(order.BalanceAmountBs)
	balanceUsedUSD := 0.0
	if balanceUsedBs > 0 {
		balanceUsedUSD = money.RoundUSD(balanceUsedBs / rate)
	}
	totalBs := money.RoundBs(orderModel.TotalPrice * rate)
	pagoMovilBs := money.RoundBs(totalBs - balanceUsedBs)
	if pagoMovilBs < 0 {
		pagoMovilBs = 0
	}

	if balanceUsedBs > 0 {
		if balanceUsedUSD+priceTolerance < 1 {
			return &models.Order{}, fmt.Errorf("%w: balance_amount_bs equivale a menos de 1 USD", errcustom.ErrInvalidBalanceAmount)
		}
		if balanceUsedUSD > orderModel.TotalPrice+priceTolerance {
			return &models.Order{}, fmt.Errorf("%w: balance_amount_bs no puede superar el total de la orden", errcustom.ErrInvalidBalanceAmount)
		}
		available, err := o.customerBalance.GetBalanceBs(ctx, order.CustomerID)
		if err != nil {
			return &models.Order{}, err
		}
		if balanceUsedBs > available+priceTolerance {
			return &models.Order{}, errcustom.ErrInsufficientBalance
		}
		// Cap floating noise: treat near-total as full cover.
		if math.Abs(orderModel.TotalPrice-balanceUsedUSD) <= priceTolerance {
			balanceUsedBs = totalBs
			pagoMovilBs = 0
		}
	}

	if pagoMovilBs > priceTolerance && order.PaymentProofImage == "" {
		return &models.Order{}, fmt.Errorf("%w: payment_proof_image es requerido cuando el balance no cubre el total", errcustom.ErrInvalidBalanceAmount)
	}

	DetailsModel := new(models.Details)
	DetailsModel.BuildDetails(orderModel, shop, customer, order.Address.Instructions, rate)

	DetailsModel.PaymentDetails = &models.PaymentDetails{
		BalanceAmountBs:   balanceUsedBs,
		PagoMovilAmountBs: pagoMovilBs,
		BcvRate:           rate,
	}

	var proofURL string
	if pagoMovilBs > priceTolerance {
		proofUpload, err := o.uploadImage.UploadImage(ctx, order.PaymentProofImage)
		if err != nil {
			return &models.Order{}, errcustom.ErrUploadingImage
		}
		proofURL = proofUpload.SecureURL
		if proofURL == "" {
			return &models.Order{}, errcustom.ErrUploadingImage
		}
		DetailsModel.PaymentDetails.PaymentProofURL = proofURL
		DetailsModel.PaymentDetails.ExpectedAmountBs = pagoMovilBs
		orderModel.Status = string(models.StatusPaymentReview)
	} else {
		// 100% balance (or no PM remainder): va directo al local.
		orderModel.Status = string(models.StatusPending)
		now := time.Now()
		orderModel.PendingAt = &now
	}

	// Consume the quote only after side-effect-free validation (including proof upload).
	if quote != nil {
		ok, err := o.quotes.TryConsume(ctx, quote.QuoteID, quote.RedisTTL(time.Now()))
		if err != nil {
			return &models.Order{}, err
		}
		if !ok {
			return &models.Order{}, errcustom.ErrQuoteConsumed
		}
		_ = o.quotes.ClearActiveID(ctx, order.CustomerID)
	}

	if balanceUsedBs > 0 {
		if err := o.customerBalance.DebitForOrder(ctx, order.CustomerID, orderModel.ID, balanceUsedBs); err != nil {
			o.releaseQuoteConsume(ctx, quote)
			return &models.Order{}, err
		}
	}

	detailsData, err := json.Marshal(DetailsModel)
	if err != nil {
		o.refundBalanceDebit(ctx, order.CustomerID, orderModel.ID, balanceUsedBs)
		o.releaseQuoteConsume(ctx, quote)
		return &models.Order{}, err
	}

	orderModel.Details = detailsData

	orderDB, err := o.repo.CreateOrder(ctx, orderModel)
	if err != nil {
		o.refundBalanceDebit(ctx, order.CustomerID, orderModel.ID, balanceUsedBs)
		o.releaseQuoteConsume(ctx, quote)
		return &models.Order{}, err
	}

	orderDB.ApplyBsPrice(rate)

	if proofURL != "" {
		expectedBs := pagoMovilBs
		nowReview := time.Now()
		review := &models.PaymentReview{
			ID:               uuid.New(),
			OrderID:          orderDB.ID,
			ProofURL:         proofURL,
			ExpectedAmountBs: expectedBs,
			BcvRate:          rate,
			Status:           models.ReviewPendingReview,
			CreatedAt:        nowReview,
			LastAlertedAt:    &nowReview,
		}
		if err := o.reviewsRepo.Create(ctx, review); err != nil {
			slog.Error("CreateOrder: payment review failed", "orderID", orderDB.ID, "error", err)
			o.refundBalanceDebit(ctx, order.CustomerID, orderDB.ID, balanceUsedBs)
			return &models.Order{}, err
		}
		event := models.DataEventPaymentProofPending{}
		event.BuildFrom(*orderDB, *review, 0)
		if err := o.stream.PublisherEvent(ctx, event); err != nil {
			slog.Error("error producing payment_proof_pending", "error", err)
		}
		if o.staffHub != nil {
			b := models.PaymentReviewBroadcast{
				Type:              "payment_review_update",
				ReviewID:          review.ID,
				OrderID:           orderDB.ID,
				ShopID:            orderDB.ShopID,
				CustomerID:        orderDB.CustomerID,
				ProofURL:          review.ProofURL,
				ExpectedAmountBs:  review.ExpectedAmountBs,
				Status:            review.Status,
				BalanceAmountBs:   balanceUsedBs,
				PagoMovilAmountBs: pagoMovilBs,
				TotalUSD:          orderDB.TotalPrice,
				OrderStatus:       orderDB.Status,
				CreatedAt:         review.CreatedAt,
			}
			if DetailsModel.CustomerDetails.Name != "" {
				b.CustomerName = DetailsModel.CustomerDetails.Name
				if DetailsModel.CustomerDetails.LastName != "" {
					b.CustomerName += " " + DetailsModel.CustomerDetails.LastName
				}
				b.CustomerPhone = DetailsModel.CustomerDetails.Phone
			}
			_ = o.staffHub.BroadcastReview(b)
		}
	}

	o.addSoldItemsAsync(*orderDB)

	eventOrderCreateModel := models.DataEventOrderCreated{}
	eventOrderCreateModel.BuidlToModelEvent(*orderDB)

	if err := o.stream.PublisherEvent(ctx, eventOrderCreateModel); err != nil {
		slog.Error("error producing event order created to stream", "error", err)
	}

	return orderDB, nil
}

func (o *orderApp) refundBalanceDebit(ctx context.Context, customerID, orderID uuid.UUID, amount float64) {
	if amount <= 0 || o.customerBalance == nil {
		return
	}
	if err := o.customerBalance.RefundBalancePayment(ctx, customerID, orderID, amount); err != nil {
		slog.Error("refundBalanceDebit failed", "orderID", orderID, "amount", amount, "error", err)
	}
}

func (o *orderApp) releaseQuoteConsume(ctx context.Context, quote *models.OrderQuote) {
	if quote == nil || o.quotes == nil {
		return
	}
	if err := o.quotes.ReleaseConsume(ctx, quote.QuoteID); err != nil {
		slog.Error("releaseQuoteConsume failed", "quoteID", quote.QuoteID, "error", err)
	}
}

func (o *orderApp) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	return o.repo.SearchOrdersByID(ctx, orderID)
}

func (o *orderApp) SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error) {
	return o.repo.SearchActiveOrdersByCriteria(ctx, criteria, isShop)
}

func (o *orderApp) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string, requesterID uuid.UUID, isShop bool, callerShopID uuid.UUID) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	if !isShop && order.CustomerID != requesterID {
		return models.Order{}, errcustom.ErrUnautorized
	}

	if isShop && order.ShopID != callerShopID {
		return models.Order{}, errcustom.ErrUnautorized
	}

	newStatus := models.OrderStatus(strings.TrimSpace(status))

	if !allowedTransitionStatuses[newStatus] {
		return models.Order{}, errcustom.ErrInvalidStatus
	}

	if !isShop {
		return models.Order{}, errcustom.ErrUnautorized
	}

	if isShop && newStatus == models.StatusCompleted {
		return models.Order{}, errcustom.ErrUnautorized
	}

	prevStatus := order.Status
	newStatus = resolveAutoTransition(models.PaymentMethod(strings.TrimSpace(order.MethodPayment)), newStatus)

	if err := o.repo.UpdateOrderStatus(ctx, orderID, string(newStatus)); err != nil {
		return models.Order{}, err
	}

	order.Status = string(newStatus)

	if newStatus == models.StatusRejected && prevStatus != string(models.StatusRejected) {
		o.refundCustomerOnShopReject(ctx, order)
	}

	o.handleStatusTransition(ctx, newStatus, order)

	if newStatus == models.StatusRejected || newStatus == models.StatusCompleted || newStatus == models.StatusCancelled {
		o.tryCloseShopBatch(ctx, order.ShopID)
	}

	return *order, nil
}

// refundCustomerOnShopReject acredita el total de la orden (USD) al balance
// del cliente cuando el local rechaza un pedido ya pagado (Pago Móvil).
// El exceso por overpay, si existía, ya se acreditó al aprobar el comprobante;
// aquí va el monto de la orden propiamente dicho.
func (o *orderApp) refundCustomerOnShopReject(ctx context.Context, order *models.Order) {
	o.refundCustomerOnShopRejectWithReason(ctx, order, "Orden rechazada por el local")
}

func (o *orderApp) refundCustomerOnShopRejectWithReason(ctx context.Context, order *models.Order, reason string) {
	if o.customerBalance == nil {
		return
	}
	if models.PaymentMethod(strings.TrimSpace(order.MethodPayment)) != models.MethodPagoMovil {
		return
	}
	if order.TotalPrice <= 0 {
		return
	}

	rate := bcvRateFromPaymentDetails(order)
	if rate <= 0 {
		// No hay tasa congelada en details (pedido legado); se usa la vigente.
		rate = o.bcvRate(ctx)
	}
	if rate <= 0 {
		slog.Error("refundCustomerOnShopReject: bcv rate unavailable", "orderID", order.ID)
		return
	}

	balanceUsedBs, pmFromDetails, hasPD := paymentSplitFromDetails(order, rate)
	// Fuente de verdad del ledger si details no trae el split (o lo trae mal).
	if debited, err := o.customerBalance.GetBalancePaymentDebitForOrder(ctx, order.CustomerID, order.ID); err == nil && debited > balanceUsedBs {
		balanceUsedBs = debited
	}
	totalBs := money.RoundBs(order.TotalPrice * rate)
	refundablePMBs := cashRefundablePMBs(totalBs, balanceUsedBs, pmFromDetails, hasPD)
	txID, createdAt, err := o.customerBalance.CreditOrderRejected(ctx, order.CustomerID, order.ID, totalBs, refundablePMBs, rate)
	if err != nil {
		slog.Error("refundCustomerOnShopReject credit failed", "orderID", order.ID, "error", err)
		return
	}

	var details models.Details
	if len(order.Details) > 0 {
		_ = json.Unmarshal(order.Details, &details)
	}
	if details.PaymentDetails == nil {
		details.PaymentDetails = &models.PaymentDetails{}
	}
	details.PaymentDetails.BalanceAmountBs = balanceUsedBs
	details.PaymentDetails.PagoMovilAmountBs = refundablePMBs
	details.PaymentDetails.BcvRate = rate
	details.PaymentDetails.Status = models.ReviewRefundedShopReject
	details.PaymentDetails.RejectionReason = &reason
	applyBalanceCreditToPaymentDetails(details.PaymentDetails, txID, totalBs, refundablePMBs, rate, createdAt)

	data, err := json.Marshal(details)
	if err != nil {
		slog.Error("refundCustomerOnShopReject marshal details failed", "orderID", order.ID, "error", err)
		return
	}
	order.Details = data
	if err := o.repo.UpdateOrderDetails(ctx, order.ID, data); err != nil {
		slog.Error("refundCustomerOnShopReject persist details failed", "orderID", order.ID, "error", err)
	}
}

// AutoRejectStalePending rechaza órdenes que llevan demasiado tiempo en Pending
// sin aceptación del local (mismo efecto económico que reject manual del shop).
func (o *orderApp) AutoRejectStalePending(ctx context.Context) ([]models.Order, error) {
	cutoff := time.Now().Add(-models.PendingAcceptTimeout)
	stale, err := o.repo.ListStalePending(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	if len(stale) == 0 {
		return nil, nil
	}

	out := make([]models.Order, 0, len(stale))
	for i := range stale {
		order := stale[i]
		ok, err := o.repo.RejectIfPending(ctx, order.ID)
		if err != nil {
			slog.Error("AutoRejectStalePending RejectIfPending failed", "orderID", order.ID, "error", err)
			continue
		}
		if !ok {
			continue
		}
		order.Status = string(models.StatusRejected)
		o.refundCustomerOnShopRejectWithReason(ctx, &order, "El local no aceptó el pedido a tiempo")
		o.tryCloseShopBatch(ctx, order.ShopID)
		o.publishOrderAutoRejected(ctx, order)
		out = append(out, order)
		slog.Info("AutoRejectStalePending rejected order", "orderID", order.ID, "shopID", order.ShopID)
	}
	return out, nil
}

func (o *orderApp) publishOrderAutoRejected(ctx context.Context, order models.Order) {
	if o.stream == nil {
		return
	}
	minutes := int(models.PendingAcceptTimeout / time.Minute)
	if order.PendingAt != nil {
		minutes = max(int(time.Since(*order.PendingAt)/time.Minute), minutes)
	}
	event := models.DataEventOrderAutoRejected{}
	event.BuildFromOrder(order, minutes)
	if err := o.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("AutoRejectStalePending publish failed", "orderID", order.ID, "error", err)
		return
	}
	slog.Info("order_auto_rejected published", "orderID", order.ID, "shopID", order.ShopID, "minutesPending", minutes)
}

// NotifyPendingUnassigned publica order_pending_unassigned en los minutos 4 y 8
// mientras el local no acepta. En el minuto 10 AutoRejectStalePending cancela.
func (o *orderApp) NotifyPendingUnassigned(ctx context.Context) error {
	if o.stream == nil {
		return nil
	}
	cutoff := time.Now().Add(-time.Duration(models.PendingUnassignedFirstAlertMinute) * time.Minute)
	pending, err := o.repo.ListPendingUnassignedForAlert(ctx, cutoff)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range pending {
		order := pending[i]
		if order.PendingAt == nil {
			continue
		}
		elapsedMin := int(now.Sub(*order.PendingAt) / time.Minute)
		for _, minute := range models.PendingUnassignedAlertMinutes {
			if elapsedMin < minute || order.PendingAlertMinute >= minute {
				continue
			}
			ok, err := o.repo.MarkPendingAlertMinute(ctx, order.ID, minute)
			if err != nil {
				slog.Error("NotifyPendingUnassigned MarkPendingAlertMinute failed", "orderID", order.ID, "minute", minute, "error", err)
				break
			}
			if !ok {
				break
			}
			order.PendingAlertMinute = minute
			event := models.DataEventOrderPendingUnassigned{}
			event.BuildFromOrder(order, minute)
			if err := o.stream.PublisherEvent(ctx, event); err != nil {
				slog.Error("NotifyPendingUnassigned publish failed", "orderID", order.ID, "minute", minute, "error", err)
				continue
			}
			slog.Info("NotifyPendingUnassigned published", "orderID", order.ID, "shopID", order.ShopID, "minutesPending", minute)
		}
	}
	return nil
}

func resolveAutoTransition(method models.PaymentMethod, status models.OrderStatus) models.OrderStatus {
	if methodRules, ok := paymentAutoTransitions[method]; ok {
		if auto, ok := methodRules[status]; ok {
			return auto
		}
	}
	return status
}

func (o *orderApp) handleStatusTransition(ctx context.Context, status models.OrderStatus, order *models.Order) {
	deliveryfee.ApplyBsToOrder(ctx, o.bcv, order)
	if h, ok := o.statusHandlers[status]; ok {
		if err := h(ctx, *order); err != nil {
			slog.Error("status transition handler failed", "status", status, "orderID", order.ID, "error", err)
		}
		return
	}
	o.riderMgr.BroadcastOrderUpdate(ctx, *order)
}

func (o *orderApp) notifyRiderUpdate(ctx context.Context, order *models.Order) {
	deliveryfee.ApplyBsToOrder(ctx, o.bcv, order)
	o.riderMgr.BroadcastOrderUpdate(ctx, *order)
}

func (o *orderApp) generateAndSavePickupCode(ctx context.Context, orderID uuid.UUID) {
	code, err := generateCode()
	if err != nil {
		slog.Error("error generating pickup code", "orderID", orderID, "error", err)
		return
	}
	if err := o.repo.SavePickupCode(ctx, orderID, code); err != nil {
		slog.Error("error saving pickup code", "orderID", orderID, "error", err)
	}
}

func (o *orderApp) ConfirmPickup(ctx context.Context, orderID uuid.UUID, code string, callerShopID uuid.UUID) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	if order.ShopID != callerShopID {
		return models.Order{}, errcustom.ErrUnautorized
	}

	if order.Status != string(models.StatusInPreparation) && order.Status != string(models.StatusReady) {
		return models.Order{}, errcustom.ErrInvalidTransition
	}

	if order.PickupCode == nil || *order.PickupCode != code {
		return models.Order{}, errcustom.ErrInvalidCode
	}

	if order.PickupCodeUsedAt != nil {
		return models.Order{}, errcustom.ErrCodeAlreadyUsed
	}

	if err := o.repo.MarkPickupCodeUsed(ctx, orderID); err != nil {
		return models.Order{}, err
	}

	deliveryCode, err := generateCode()
	if err != nil {
		slog.Error("error generating delivery code", "orderID", orderID, "error", err)
		if err := o.repo.UpdateOrderStatus(ctx, orderID, string(models.StatusOnTheWay)); err != nil {
			return models.Order{}, err
		}
		order.Status = string(models.StatusOnTheWay)
		o.notifyRiderUpdate(ctx, order)
		return *order, nil
	}

	if err := o.repo.SaveDeliveryCode(ctx, orderID, deliveryCode); err != nil {
		slog.Error("error saving delivery code", "orderID", orderID, "error", err)
	}
	order.DeliveryCode = &deliveryCode

	if err := o.repo.UpdateOrderStatus(ctx, orderID, string(models.StatusOnTheWay)); err != nil {
		return models.Order{}, err
	}

	order.Status = string(models.StatusOnTheWay)
	o.notifyRiderUpdate(ctx, order)
	return *order, nil
}

func (o *orderApp) ConfirmDelivery(ctx context.Context, orderID uuid.UUID, code string, callerRiderID uuid.UUID) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	if order.RiderID == nil || *order.RiderID != callerRiderID {
		return models.Order{}, errcustom.ErrUnautorized
	}

	if order.Status != string(models.StatusOnTheWay) {
		return models.Order{}, errcustom.ErrInvalidTransition
	}

	if order.DeliveryCode == nil || *order.DeliveryCode != code {
		return models.Order{}, errcustom.ErrInvalidCode
	}

	if order.DeliveryCodeUsedAt != nil {
		return models.Order{}, errcustom.ErrCodeAlreadyUsed
	}

	if err := o.repo.MarkDeliveryCodeUsed(ctx, orderID); err != nil {
		return models.Order{}, err
	}

	if err := o.repo.UpdateOrderStatus(ctx, orderID, string(models.StatusCompleted)); err != nil {
		return models.Order{}, err
	}

	order.Status = string(models.StatusCompleted)

	if order.RiderID == nil {
		slog.Error("order completed without rider assigned", "orderID", orderID)
		o.tryCloseShopBatch(ctx, order.ShopID)
		return *order, nil
	}
	// Sin tasa persistida por orden a esta altura del ciclo de vida (ver
	// refundCustomerOnShopRejectWithReason): se usa la tasa BCV vigente al
	// completar la entrega, más justa para el rider que la del quote inicial.
	if err := o.finance.RecordRideEarning(ctx, *order.RiderID, order.ID, order.DeliveryFee, order.TotalPrice, o.bcvRate(ctx), order.MethodPayment); err != nil {
		slog.Error("error recording ride earning", "orderID", orderID, "error", err)
	}

	if err := o.shopCommission.RecordSaleCommission(ctx, order.ShopID, order.ID, order.TotalPrice, order.DeliveryFee, order.MethodPayment, order.IsLaunchPromo, order.IsZeroCommission, order.PromoSubsidyAmount); err != nil {
		slog.Error("error recording shop commission", "orderID", orderID, "shopID", order.ShopID, "error", err)
	}

	o.notifyRiderUpdate(ctx, order)
	o.tryCloseShopBatch(ctx, order.ShopID)

	return *order, nil
}

func (o *orderApp) GetOrderHistory(ctx context.Context, customerID uuid.UUID) (dtos.OrderHistoryResponse, error) {
	orders, err := o.repo.SearchOrderHistoryByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	history := make(dtos.OrderHistoryResponse, len(orders))
	for i, ord := range orders {
		history[i] = dtos.OrderHistoryItem{
			ID:            ord.ID,
			Number:        ord.Number,
			Status:        ord.Status,
			TotalPrice:    ord.TotalPrice,
			DeliveryFee:   ord.DeliveryFee,
			MethodPayment: ord.MethodPayment,
			Address:       ord.Address,
			ShopID:        ord.ShopID,
			CreatedAt:     ord.CreatedAt,
		}
	}

	return history, nil
}

// addSoldItemsAsync incrementa el contador "sold" de cada item vendido en segundo
// plano, sin bloquear la respuesta ni la publicación del evento de la orden.
func (o *orderApp) addSoldItemsAsync(order models.Order) {
	items := order.ItemsOrders
	go func() {
		ctx := context.Background()
		for _, item := range items {
			soldItem := productsModels.DataProduceOrder{ProductID: item.ItemID, Quantity: item.Amount}
			if err := o.products.AddSoldItem(ctx, soldItem); err != nil {
				slog.Error("error incrementando sold del producto", "productID", item.ItemID, "error", err)
			}
		}
	}()
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}
