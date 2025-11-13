package app

import (
	"context"
	"errors"
	"log/slog"

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
	socket AppManagerOrder
}

func NewOrderApp(repo ports.OrdersRepo, shops shops.ShopsRepo, stream ports.RedisStream, socket AppManagerOrder) ports.OrdersApp {
	return &orderApp{repo, shops, stream, socket}
}

func (o *orderApp) CreateOrder(ctx context.Context, order *dtos.CreateOrder) error {

	ord, err := o.SearchOrdersByID(ctx, order.ID)
	if ord != nil {
		return errors.New("order already exists")
	}

	if err != nil {
		if err.Error() != "order not found" {
			return err
		}
	}

	batchID, err := o.shops.SearchBatchByShopID(ctx, order.ShopID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return errors.New("batch not open")
		}
	}

	orderModel := new(models.Order)
	orderModel.BuildDtoToModel(*order, batchID)

	items, err := o.repo.SearchItemsByID(ctx, orderModel.GetItemsID())
	if err != nil {
		return err
	}

	orderModel.SetPrices(items)

	for _, v := range orderModel.ItemsOrders {
		if v.Additions != nil {
			additions, err := o.repo.SearchAdditionsByID(ctx, v.GetAdditionsID())
			if err != nil {
				return err
			}

			orderModel.SetAdditionalsPrices(additions)
		}
	}

	err = o.repo.CreateOrder(ctx, orderModel)
	if err != nil {
		return err
	}

	err = o.socket.BroadcastOrder(*orderModel)
	if err != nil {
		slog.Error("error broadcasting order to websockets", "error", err)
	}

	if err := o.stream.Producer(ctx, buildToDataProduceOrder(*orderModel)); err != nil {
		slog.Error("error producing order to stream", "error", err)
	}

	eventOrderCreateModel := models.DataEventOrderCreated{}
	eventOrderCreateModel.BuidlToModelEvent(*orderModel)

	if err := o.stream.EventOrderCreated(ctx, eventOrderCreateModel); err != nil {
		slog.Error("error producing event order created to stream", "error", err)
	}

	return nil
}

func (o *orderApp) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	return o.repo.SearchOrdersByID(ctx, orderID)
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
