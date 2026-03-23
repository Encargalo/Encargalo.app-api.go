package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/users/domain/dto"
	portsCustomers "Encargalo.app-api.go/internal/users/domain/ports"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/AndresXLP/ravenTree"
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
	custo  portsCustomers.UsersRepo
	stream ports.RedisStream
}

func NewOrderApp(repo ports.OrdersRepo, shops shops.ShopsRepo, custo portsCustomers.UsersRepo, stream ports.RedisStream) ports.OrdersApp {
	return &orderApp{repo, shops, custo, stream}
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

	if newStatus == StatusOnTheWay {
		o.sendCreditDiscount(ctx, order)
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

func (o *orderApp) sendCreditDiscount(ctx context.Context, order *models.Order) {

	config := config.Get().CreditManager

	serviceName := "billing-service"
	requestID := uuid.New().String()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	body := calculateLoanAmount(*order)

	bodyAux, _ := json.Marshal(body)

	payload := serviceName + requestID + timestamp + string(bodyAux)

	h := hmac.New(sha256.New, []byte(config.SigningKey))
	h.Write([]byte(payload))
	signature := hex.EncodeToString(h.Sum(nil))

	tree := ravenTree.NewRavensTree()
	options := &ravenTree.Options{
		Headers: map[string]string{
			"X-Service-Name": serviceName,
			"X-Request-ID":   requestID,
			"X-Timestamp":    timestamp,
			"X-Signature":    signature,
		},
		Host:   config.Url,
		Path:   fmt.Sprintf("/credits/wallet/%s/deduct-commission", order.ShopID),
		Method: http.MethodPost,
		Body:   body,
	}

	resp, _ := tree.SendRaven(ctx, options)

	if resp.Status != "202" {
		var dataEventFail models.EventFailCreditDiscount
		dataEventFail.BuildFailCreditDiscount(resp.Status, resp.ParseBodyToString(), body.OrderID.String(), body.Amount)

		if err := o.stream.PublisherEvent(ctx, dataEventFail); err != nil {
			slog.Error("error producing event fail credits discounts to stream", "error", err)
		}
		slog.Info("event publisher successfully")
		return
	}

	slog.Info("credit discount sent successfully")
}

func calculateLoanAmount(order models.Order) models.CreditManager {

	amount := float64(order.TotalPrice-order.DeliveryFee) * 0.10

	data := models.CreditManager{
		Amount:  amount,
		OrderID: order.ID,
	}

	return data

}
