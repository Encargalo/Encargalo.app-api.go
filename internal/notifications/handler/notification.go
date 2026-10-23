package handler

import (
	"Encargalo.app-api.go/internal/notifications/app"
	"Encargalo.app-api.go/internal/notifications/domain/models"
	"github.com/redis/go-redis/v9"
)

type Notification interface {
	OrderCreated(msg redis.XMessage) error
	OrderCompleted(msg redis.XMessage) error
	OrderRejected(msg redis.XMessage) error
	OrderPendingUnassigned(msg redis.XMessage) error
	OrderAutoRejected(msg redis.XMessage) error
	OrderNoRider(msg redis.XMessage) error
	PaymentProofPending(msg redis.XMessage) error
	BalanceRefundPending(msg redis.XMessage) error
	TripCreated(msg redis.XMessage) error
	TripNoDriver(msg redis.XMessage) error
	WithdrawalRequested(msg redis.XMessage) error
}

type notification struct {
	svc app.Notifications
}

func NewNotification(svc app.Notifications) Notification {
	return &notification{svc}
}

func (n *notification) OrderCreated(msg redis.XMessage) error {
	var data models.DataEventOrderCreated
	data.BuidlToModelEvent(msg)

	return n.svc.OrderCreated(data.OrderID)
}

func (n *notification) OrderCompleted(msg redis.XMessage) error {
	var data models.DataEventOrderCreated
	data.BuidlToModelEvent(msg)

	return n.svc.OrderCompleted(data.OrderID)
}

func (n *notification) OrderRejected(msg redis.XMessage) error {
	var data models.DataEventOrderCreated
	data.BuidlToModelEvent(msg)

	return n.svc.OrderRejected(data.OrderID)
}

func (n *notification) OrderPendingUnassigned(msg redis.XMessage) error {
	var data models.DataEventOrderPendingUnassigned
	data.BuildFromXMessage(msg)
	return n.svc.OrderPendingUnassigned(data.OrderID, data.MinutesPending)
}

func (n *notification) OrderAutoRejected(msg redis.XMessage) error {
	var data models.DataEventOrderAutoRejected
	data.BuildFromXMessage(msg)
	return n.svc.OrderAutoRejected(data.OrderID, data.MinutesPending)
}

func (n *notification) PaymentProofPending(msg redis.XMessage) error {
	var data models.DataEventPaymentProofPending
	data.BuildFromXMessage(msg)
	return n.svc.PaymentProofPending(data.ReviewID, data.MinutesPending)
}

func (n *notification) BalanceRefundPending(msg redis.XMessage) error {
	var data models.DataEventBalanceRefundPending
	data.BuildFromXMessage(msg)
	return n.svc.BalanceRefundPending(data.RequestID, data.MinutesPending)
}

func (n *notification) TripCreated(msg redis.XMessage) error {
	var data models.DataEventTrip
	data.BuildFromXMessage(msg)
	return n.svc.TripCreated(data.TripID)
}

func (n *notification) TripNoDriver(msg redis.XMessage) error {
	var data models.DataEventTrip
	data.BuildFromXMessage(msg)
	return n.svc.TripNoDriver(data.TripID)
}

func (n *notification) OrderNoRider(msg redis.XMessage) error {
	var data models.DataEventOrderNoRider
	data.BuildFromXMessage(msg)
	return n.svc.OrderNoRider(data.OrderID)
}

func (n *notification) WithdrawalRequested(msg redis.XMessage) error {
	var data models.DataEventWithdrawalRequested
	data.BuildFromXMessage(msg)
	return n.svc.WithdrawalRequested(data.RequestID)
}
