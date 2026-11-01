package repo

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/fcm"
	shipmentsModels "Encargalo.app-api.go/internal/shipments/domain/models"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

const (
	pushTripOfferTitle         = "¡Nuevo viaje disponible!"
	pushTripOfferBody          = "Hay un viaje cerca de ti. Abre la app para aceptarlo."
	pushNewOrderTitle          = "¡Nuevo pedido disponible!"
	pushNewOrderBody           = "Hay un pedido cerca de ti. Abre la app para tomarlo."
	pushTripCancelledTitle     = "Viaje cancelado"
	pushTripCancelledBody      = "El cliente canceló el viaje. Ya puedes recibir nuevas ofertas."
	pushOrderCancelledTitle    = "Pedido cancelado"
	pushOrderCancelledBody     = "La tienda canceló el pedido. Ya puedes recibir nuevos pedidos."
	pushNewShipmentTitle       = "¡Nuevo envío disponible!"
	pushNewShipmentBody        = "Hay un envío cerca de ti. Abre la app para aceptarlo."
	pushShipmentCancelledTitle = "Envío cancelado"
	pushShipmentCancelledBody  = "El cliente canceló el envío. Ya puedes recibir nuevas ofertas."
	// pushFanOutLimit acota cuántos envíos a FCM corren en paralelo por ronda —
	// una oferta nunca llega a cientos de candidatos, pero evita una fuga de
	// goroutines si algún día lo hace.
	pushFanOutLimit = 20
)

type pushFCMNotifier struct {
	client *fcm.Client
	tokens ports.RiderProfileRepo
}

func NewPushNotifier(client *fcm.Client, tokens ports.RiderProfileRepo) ports.PushNotifier {
	return &pushFCMNotifier{client: client, tokens: tokens}
}

func (p *pushFCMNotifier) NotifyTripOffer(ctx context.Context, riderIDs []uuid.UUID, trip tripsModels.Trip) {
	data := map[string]string{"type": "trip_offer", "trip_id": trip.ID.String()}
	p.notify(ctx, riderIDs, pushTripOfferTitle, pushTripOfferBody, data)
}

func (p *pushFCMNotifier) NotifyNewOrder(ctx context.Context, riderIDs []uuid.UUID, order ordersModels.Order) {
	data := map[string]string{"type": "new_order", "order_id": order.ID.String()}
	p.notify(ctx, riderIDs, pushNewOrderTitle, pushNewOrderBody, data)
}

func (p *pushFCMNotifier) NotifyTripCancelled(ctx context.Context, riderIDs []uuid.UUID, trip tripsModels.Trip) {
	data := map[string]string{"type": "trip_cancelled", "trip_id": trip.ID.String()}
	p.notify(ctx, riderIDs, pushTripCancelledTitle, pushTripCancelledBody, data)
}

func (p *pushFCMNotifier) NotifyOrderCancelled(ctx context.Context, riderIDs []uuid.UUID, order ordersModels.Order) {
	data := map[string]string{"type": "order_cancelled", "order_id": order.ID.String()}
	p.notify(ctx, riderIDs, pushOrderCancelledTitle, pushOrderCancelledBody, data)
}

func (p *pushFCMNotifier) NotifyNewShipment(ctx context.Context, riderIDs []uuid.UUID, shipment shipmentsModels.Shipment) {
	data := map[string]string{"type": "new_shipment", "shipment_id": shipment.ID.String()}
	p.notify(ctx, riderIDs, pushNewShipmentTitle, pushNewShipmentBody, data)
}

func (p *pushFCMNotifier) NotifyShipmentCancelled(ctx context.Context, riderIDs []uuid.UUID, shipment shipmentsModels.Shipment) {
	data := map[string]string{"type": "shipment_cancelled", "shipment_id": shipment.ID.String()}
	p.notify(ctx, riderIDs, pushShipmentCancelledTitle, pushShipmentCancelledBody, data)
}

func (p *pushFCMNotifier) notify(ctx context.Context, riderIDs []uuid.UUID, title, body string, data map[string]string) {
	if p.client == nil || len(riderIDs) == 0 {
		return
	}
	tokens, err := p.tokens.GetDeviceTokensByRiderIDs(ctx, riderIDs)
	if err != nil {
		slog.Error("error leyendo device tokens para push", "error", err)
		return
	}

	sem := make(chan struct{}, pushFanOutLimit)
	var wg sync.WaitGroup
	for _, t := range tokens {
		wg.Add(1)
		sem <- struct{}{}
		go func(riderID uuid.UUID, fcmToken string) {
			defer wg.Done()
			defer func() { <-sem }()
			p.send(ctx, riderID, fcmToken, title, body, data)
		}(t.RiderID, t.FCMToken)
	}
	wg.Wait()
}

func (p *pushFCMNotifier) send(ctx context.Context, riderID uuid.UUID, token, title, body string, data map[string]string) {
	err := p.client.Send(ctx, token, title, body, data)
	if err == nil {
		return
	}
	if errors.Is(err, fcm.ErrInvalidToken) {
		if delErr := p.tokens.DeleteDeviceToken(context.Background(), riderID); delErr != nil {
			slog.Error("error borrando device token inválido", "riderID", riderID, "error", delErr)
		}
		return
	}
	slog.Error("error enviando push FCM", "riderID", riderID, "error", err)
}
