package app

import (
	"context"
	"errors"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"github.com/google/uuid"
)

type orderApp struct {
	repo ports.OrdersRepo
}

func NewOrderApp(repo ports.OrdersRepo) ports.OrdersApp {
	return &orderApp{repo}
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

	orderModel := new(models.Order)
	orderModel.BuildDtoToModel(*order)

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

	return nil
}

func (o *orderApp) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	return o.repo.SearchOrdersByID(ctx, orderID)
}
