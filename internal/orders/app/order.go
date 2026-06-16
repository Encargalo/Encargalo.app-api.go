package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shops "Encargalo.app-api.go/internal/shops/domain/ports"
	"Encargalo.app-api.go/internal/users/domain/dto"
	portsCustomers "Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

type PaymentMethod string

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
	riderMgr    ports.OrderRiderBroadcaster
	finance     ports.RideEarningRecorder
}

func NewOrderApp(repo ports.OrdersRepo, shops shops.ShopsRepo, custo portsCustomers.UsersRepo, stream ports.RedisStream, uploadImage uploadimage.UploadImage, riderMgr ports.OrderRiderBroadcaster, finance ports.RideEarningRecorder) ports.OrdersApp {
	return &orderApp{repo, shops, custo, stream, uploadImage, riderMgr, finance}
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

	fee, err := deliveryfee.Calculate(shop.Latitude, shop.Longitude, order.Address.Latitude, order.Address.Longitude)
	if err != nil {
		slog.Error("calcDeliveryFee: google maps failed", "shopID", shop.ID, "error", err)
		return &models.Order{}, errcustom.ErrUnexpectedError
	}
	orderModel.DeliveryFee = fee

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

	newStatus := models.OrderStatus(strings.TrimSpace(status))
	method := PaymentMethod(strings.TrimSpace(order.MethodPayment))

	var allowedStatuses = map[models.OrderStatus]bool{
		models.StatusAccepted:            true,
		models.StatusWaitingPaymentProof: true,
		models.StatusInPreparation:       true,
		models.StatusReady:               true,
		models.StatusOnTheWay:            true,
		models.StatusCompleted:           true,
		models.StatusCancelled:           true,
		models.StatusRejected:            true,
	}

	if !allowedStatuses[newStatus] {
		return models.Order{}, fmt.Errorf("invalid status: %s", newStatus)
	}

	if !isShop && newStatus != models.StatusCancelled {
		return models.Order{}, errcustom.ErrUnautorized
	}

	if isShop && newStatus == models.StatusCompleted {
		return models.Order{}, errcustom.ErrUnautorized
	}

	var transitionRules = map[PaymentMethod]map[models.OrderStatus]models.OrderStatus{
		MethodCashBs:    {models.StatusAccepted: models.StatusInPreparation},
		MethodCashUSD:   {models.StatusAccepted: models.StatusInPreparation},
		MethodPagoMovil: {models.StatusAccepted: models.StatusInPreparation},
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

	if newStatus == models.StatusInPreparation {
		o.generateAndSavePickupCode(ctx, orderID)
		o.riderMgr.BroadcastNewOrder(*order)
	}

	if newStatus != models.StatusInPreparation {
		o.riderMgr.BroadcastOrderUpdate(*order)
	}

	return *order, nil
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

func (o *orderApp) ConfirmPickup(ctx context.Context, orderID uuid.UUID, code string) (models.Order, error) {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
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
		return *order, nil
	}

	if err := o.repo.SaveDeliveryCode(ctx, orderID, deliveryCode); err != nil {
		slog.Error("error saving delivery code", "orderID", orderID, "error", err)
	}

	if err := o.repo.UpdateOrderStatus(ctx, orderID, string(models.StatusOnTheWay)); err != nil {
		return models.Order{}, err
	}

	order.Status = string(models.StatusOnTheWay)
	return *order, nil
}

func (o *orderApp) ConfirmDelivery(ctx context.Context, orderID uuid.UUID, code string) error {
	order, err := o.SearchOrdersByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != string(models.StatusOnTheWay) {
		return errcustom.ErrInvalidTransition
	}

	if order.DeliveryCode == nil || *order.DeliveryCode != code {
		return errcustom.ErrInvalidCode
	}

	if order.DeliveryCodeUsedAt != nil {
		return errcustom.ErrCodeAlreadyUsed
	}

	if err := o.repo.MarkDeliveryCodeUsed(ctx, orderID); err != nil {
		return err
	}

	if err := o.repo.UpdateOrderStatus(ctx, orderID, string(models.StatusCompleted)); err != nil {
		return err
	}

	if err := o.finance.RecordRideEarning(ctx, *order.RiderID, order.ID, order.DeliveryFee, order.TotalPrice, order.MethodPayment); err != nil {
		slog.Error("error recording ride earning", "orderID", orderID, "error", err)
	}

	return nil
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

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
