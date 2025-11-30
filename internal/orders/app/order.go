package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type OrderStatus string
type PaymentMethod string

const (
	StatusAccepted            OrderStatus = "Accepted"
	StatusWaitingPaymentProof OrderStatus = "Waiting For Payment Proof"
	StatusInPreparation       OrderStatus = "In Preparation"
	StatusOnTheWay            OrderStatus = "On The Way"
	StatusCompleted           OrderStatus = "Completed"
	StatusCancelled           OrderStatus = "Cancelled"
	StatusRejected            OrderStatus = "Rejected"
)

const (
	MethodNequi PaymentMethod = "Nequi"
	MethodCash  PaymentMethod = "Efectivo"
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

func (o *orderApp) SearchActiveOrdersByCriteria(ctx context.Context, shopID uuid.UUID) ([]models.Order, error) {
	return o.repo.SearchActiveOrdersByCriteria(ctx, shopID)
}

func (o *orderApp) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	newStatus := OrderStatus(strings.TrimSpace(status))
	method := PaymentMethod(strings.TrimSpace(order.MethodPayment))

	var allowedStatuses = map[OrderStatus]bool{
		StatusAccepted:            true,
		StatusWaitingPaymentProof: true,
		StatusInPreparation:       true,
		StatusOnTheWay:            true,
		StatusCompleted:           true,
		StatusCancelled:           true,
		StatusRejected:            true,
	}

	if !allowedStatuses[newStatus] {
		return models.Order{}, fmt.Errorf("invalid status: %s", newStatus)
	}

	var transitionRules = map[PaymentMethod]map[OrderStatus]OrderStatus{
		MethodNequi: {
			StatusAccepted: StatusWaitingPaymentProof,
		},
		MethodCash: {
			StatusAccepted: StatusInPreparation,
		},
	}

	if methodRules, ok := transitionRules[method]; ok {
		if autoStatus, ok := methodRules[newStatus]; ok {
			newStatus = autoStatus
		}
	}

	if err := o.repo.UpdateOrderStatus(ctx, orderID, string(newStatus)); err != nil {
		return models.Order{}, err
	}

	order.Status = string(newStatus)
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
