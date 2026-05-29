package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"Encargalo.app-api.go/internal/users/domain/dto"
	portsCustomers "Encargalo.app-api.go/internal/users/domain/ports"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
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
	MethodCashBs    PaymentMethod = "EfectivoBs"
	MethodCashUSD   PaymentMethod = "EfectivoDolar"
	MethodPagoMovil PaymentMethod = "PagoMovil"
)

type orderApp struct {
	repo        ports.OrdersRepo
	shops       shops.ShopsRepo
	custo       portsCustomers.UsersRepo
	stream      ports.RedisStream
	uploadImage uploadimage.UploadImage
}

func NewOrderApp(repo ports.OrdersRepo, shops shops.ShopsRepo, custo portsCustomers.UsersRepo, stream ports.RedisStream, uploadImage uploadimage.UploadImage) ports.OrdersApp {
	return &orderApp{repo, shops, custo, stream, uploadImage}
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

	batchID, err := o.shops.SearchBatchByShopID(ctx, order.ShopID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return &models.Order{}, errcustom.ErrBatchNotOpen
		}
	}

	shop, err := o.repo.SearchShopByID(ctx, order.ShopID)
	if err != nil {
		return &models.Order{}, err
	}

	orderModel := new(models.Order)
	orderModel.BuildDtoToModel(*order, batchID)

	customer, err := o.custo.SearchCustomerBy(ctx, dto.SearchCustomerBy{ID: order.CustomerID})
	if err != nil {
		return &models.Order{}, err
	}

	items, err := o.repo.SearchItemsByID(ctx, orderModel.GetItemsID())
	if err != nil {
		return &models.Order{}, err
	}

	orderModel.SetPrices(items)
	orderModel.SetItemName(items)
	orderModel.SetItemImage(items)

	for _, v := range orderModel.ItemsOrders {
		if v.Additions != nil {
			additions, err := o.repo.SearchAdditionsByID(ctx, v.GetAdditionsID())
			if err != nil {
				return &models.Order{}, err
			}

			orderModel.SetAdditionalsPrices(additions)
			orderModel.SetAdditionalsName(additions)
		}
	}

	DetailsModel := new(models.Details)
	DetailsModel.BuildDetails(orderModel, shop, customer)

	if order.PagoMovilData != nil {
		proofUpload, err := o.uploadImage.UploadImage(ctx, order.PaymentProofImage)
		if err != nil {
			return &models.Order{}, fmt.Errorf("error subiendo comprobante de pago: %w", err)
		}
		DetailsModel.PaymentDetails = &models.PaymentDetails{
			Nombre:          order.PagoMovilData.Nombre,
			Cedula:          order.PagoMovilData.Cedula,
			Telefono:        order.PagoMovilData.Telefono,
			Banco:           order.PagoMovilData.Banco,
			PaymentProofURL: proofUpload.SecureURL,
		}
	}

	detailsData, err := json.Marshal(DetailsModel)
	if err != nil {
		return &models.Order{}, err
	}

	orderModel.Details = detailsData

	orderDB, err := o.repo.CreateOrder(ctx, orderModel)
	if err != nil {
		return &models.Order{}, err
	}

	if err := o.stream.PublisherEvent(ctx, buildToDataProduceOrder(*orderDB)); err != nil {
		slog.Error("error producing order to stream", "error", err)
	}

	eventOrderCreateModel := models.DataEventOrderCreated{}
	eventOrderCreateModel.BuidlToModelEvent(*orderDB)

	if err := o.stream.PublisherEvent(ctx, eventOrderCreateModel); err != nil {
		slog.Error("error producing event order created to stream", "error", err)
	}

	return orderDB, nil
}

func (o *orderApp) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	return o.repo.SearchOrdersByID(ctx, orderID)
}

func (o *orderApp) SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error) {
	return o.repo.SearchActiveOrdersByCriteria(ctx, criteria, isShop)
}

func (o *orderApp) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string, requesterID uuid.UUID, isShop bool) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	if !isShop && order.CustomerID != requesterID {
		return models.Order{}, errcustom.ErrUnautorized
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

	if !isShop && newStatus != StatusCancelled {
		return models.Order{}, errcustom.ErrUnautorized
	}

	var transitionRules = map[PaymentMethod]map[OrderStatus]OrderStatus{
		MethodCashBs:    {StatusAccepted: StatusInPreparation},
		MethodCashUSD:   {StatusAccepted: StatusInPreparation},
		MethodPagoMovil: {StatusAccepted: StatusInPreparation},
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

