package ports

import (
	"context"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	tripsModels "Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

// PushNotifier manda push (FCM) a los riders candidatos de una oferta de
// viaje o pedido, en paralelo al WebSocket — nunca lo sustituye. Es
// best-effort: no devuelve error al llamador porque un fallo de push no debe
// romper ni frenar la oferta por WS. riderIDs ya viene filtrado por quien
// llama (en vivo + fallback de última ubicación conocida); este puerto solo
// resuelve tokens y envía.
type PushNotifier interface {
	NotifyTripOffer(ctx context.Context, riderIDs []uuid.UUID, trip tripsModels.Trip)
	NotifyNewOrder(ctx context.Context, riderIDs []uuid.UUID, order ordersModels.Order)
	// NotifyTripCancelled avisa al rider ya asignado: a diferencia de la oferta,
	// esto puede llegarle mientras maneja hacia el punto de recogida con la app
	// en segundo plano, donde el WS puede estar caído sin que se note.
	NotifyTripCancelled(ctx context.Context, riderIDs []uuid.UUID, trip tripsModels.Trip)
	// NotifyOrderCancelled es el mismo caso que NotifyTripCancelled pero para
	// pedidos: al rider ya asignado a quien la tienda rechaza o el cliente
	// cancela, no solo al que seguía ofreciéndosela.
	NotifyOrderCancelled(ctx context.Context, riderIDs []uuid.UUID, order ordersModels.Order)
}
