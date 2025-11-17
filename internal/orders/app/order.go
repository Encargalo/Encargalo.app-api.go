package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type orderApp struct {
	repo   ports.OrdersRepo
	shops  shops.ShopsRepo
	stream ports.RedisStream
}

func NewOrderApp(repo ports.OrdersRepo, shops shops.ShopsRepo, stream ports.RedisStream) ports.OrdersApp {
	return &orderApp{repo, shops, stream}
}

func (o *orderApp) CreateOrder(ctx context.Context, order *dtos.CreateOrder) (*models.Order, error) {

	ord, err := o.SearchOrdersByID(ctx, order.ID)
	if ord != nil {
		return &models.Order{}, errors.New("order already exists")
	}

	if err != nil {
		if err.Error() != "order not found" {
			return &models.Order{}, err
		}
	}

	batchID, err := o.shops.SearchBatchByShopID(ctx, order.ShopID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return &models.Order{}, errors.New("batch not open")
		}
	}

	orderModel := new(models.Order)
	orderModel.BuildDtoToModel(*order, batchID)

	items, err := o.repo.SearchItemsByID(ctx, orderModel.GetItemsID())
	if err != nil {
		return &models.Order{}, err
	}

	orderModel.SetPrices(items)

	for _, v := range orderModel.ItemsOrders {
		if v.Additions != nil {
			additions, err := o.repo.SearchAdditionsByID(ctx, v.GetAdditionsID())
			if err != nil {
				return &models.Order{}, err
			}

			orderModel.SetAdditionalsPrices(additions)
		}
	}

	err = o.repo.CreateOrder(ctx, orderModel)
	if err != nil {
		return &models.Order{}, err
	}

	if err := o.stream.Producer(ctx, buildToDataProduceOrder(*orderModel)); err != nil {
		slog.Error("error producing order to stream", "error", err)
	}

	eventOrderCreateModel := models.DataEventOrderCreated{}
	eventOrderCreateModel.BuidlToModelEvent(*orderModel)

	if err := o.stream.EventOrderCreated(ctx, eventOrderCreateModel); err != nil {
		slog.Error("error producing event order created to stream", "error", err)
	}

	return orderModel, nil
}

func (o *orderApp) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	return o.repo.SearchOrdersByID(ctx, orderID)
}

func (o *orderApp) SearchOrderByBatchIsOpen(ctx context.Context, shopID uuid.UUID) ([]models.Order, error) {
	return o.repo.SearchOrderByBatchIsOpen(ctx, shopID)
}

func (o *orderApp) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	status = strings.TrimSpace(status)
	method := strings.TrimSpace(order.MethodPayment)

	rules := map[string]map[string]string{
		"Nequi": {
			"Aceptado": "En Espera Del Comprobante De Pago",
		},
		"Efectivo": {
			"Aceptado": "En Preparacion",
		},
	}

	if methodRules, exists := rules[method]; exists {
		if newStatus, ok := methodRules[status]; ok {
			status = newStatus
		}
	}
	if err := o.repo.UpdateOrderStatus(ctx, orderID, status); err != nil {
		return models.Order{}, err
	}

	order.Status = status

	return *order, nil

}

func buildToDataProduceOrder(items models.Order) []models.DataProduceOrder {
	dataProduceOrder := []models.DataProduceOrder{}

	for _, item := range items.ItemsOrders {
		dataProduceOrder = append(dataProduceOrder, models.DataProduceOrder{
			MessageType: "add_sold_product",
			ProductID:   item.ItemID,
			Quantity:    item.Amount,
		})
	}

	return dataProduceOrder
}
