package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/models"
	items "Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
)

type OrdersApp interface {
	CreateQuote(ctx context.Context, req *dtos.CreateQuote) (*dtos.QuoteResponse, error)
	CreateOrder(ctx context.Context, order *dtos.CreateOrder) (*models.Order, error)
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error)
	SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error)
	// callerShopID solo se valida cuando isShop es true (contra order.ShopID); se ignora para customers.
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string, requesterID uuid.UUID, isShop bool, callerShopID uuid.UUID) (models.Order, error)
	// callerShopID debe coincidir con order.ShopID (el local que confirma la recogida).
	ConfirmPickup(ctx context.Context, orderID uuid.UUID, code string, callerShopID uuid.UUID) (models.Order, error)
	// callerRiderID debe coincidir con order.RiderID (el rider asignado que confirma la entrega).
	ConfirmDelivery(ctx context.Context, orderID uuid.UUID, code string, callerRiderID uuid.UUID) (models.Order, error)
	GetOrderHistory(ctx context.Context, customerID uuid.UUID) (dtos.OrderHistoryResponse, error)
	// AutoRejectStalePending rechaza órdenes Pending sin aceptación del local
	// tras PendingAcceptTimeout. Devuelve las órdenes efectivamente rechazadas
	// (para que el worker las broadcastée sin ciclo DI OrdersApp↔Manager).
	AutoRejectStalePending(ctx context.Context) ([]models.Order, error)
	// NotifyPendingUnassigned publica order_pending_unassigned en los minutos 4 y 8
	// mientras la orden sigue Pending sin aceptación del local.
	NotifyPendingUnassigned(ctx context.Context) error
}

type OrdersRepo interface {
	CreateOrder(ctx context.Context, order *models.Order) (*models.Order, error)
	SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error)
	SearchItemsByID(ctx context.Context, ItemsID []uuid.UUID) (items.Items, error)
	SearchShopByID(ctx context.Context, shopID uuid.UUID) (*models.ShopOrder, error)
	SearchAdditionsByID(ctx context.Context, additionID []uuid.UUID) ([]models.Addition, error)
	SearchFlavorsForItem(ctx context.Context, itemID uuid.UUID) ([]models.Flavor, error)
	SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) error
	SearchAvailableOrders(ctx context.Context) ([]models.Order, error)
	SavePickupCode(ctx context.Context, orderID uuid.UUID, code string) error
	SaveDeliveryCode(ctx context.Context, orderID uuid.UUID, code string) error
	MarkPickupCodeUsed(ctx context.Context, orderID uuid.UUID) error
	MarkDeliveryCodeUsed(ctx context.Context, orderID uuid.UUID) error
	AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error
	SearchActiveOrdersByRiderID(ctx context.Context, riderID uuid.UUID) ([]models.Order, error)
	SearchOrderHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Order, error)
	// ListStatusHistory devuelve las transiciones de estado por pedido, en orden cronológico.
	ListStatusHistory(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]models.StatusChange, error)
	UpdateOrderDetails(ctx context.Context, orderID uuid.UUID, details []byte) error
	MarkPendingAt(ctx context.Context, orderID uuid.UUID, at time.Time) error
	ListStalePending(ctx context.Context, olderThan time.Time) ([]models.Order, error)
	// RejectIfPending CAS Pending → Rejected. Returns false if already left Pending.
	RejectIfPending(ctx context.Context, orderID uuid.UUID) (bool, error)
	// ListPendingUnassignedForAlert: Pending past first alert minute, still due for Discord.
	ListPendingUnassignedForAlert(ctx context.Context, olderThan time.Time) ([]models.Order, error)
	// MarkPendingAlertMinute CAS advances pending_alert_minute if still Pending and behind.
	MarkPendingAlertMinute(ctx context.Context, orderID uuid.UUID, minute int) (bool, error)
}

// OrderQuotesRepo stores frozen checkout quotes (typically Redis with TTL).
type OrderQuotesRepo interface {
	Save(ctx context.Context, quote *models.OrderQuote, ttl time.Duration) error
	Get(ctx context.Context, quoteID uuid.UUID) (*models.OrderQuote, error)
	Delete(ctx context.Context, quoteID uuid.UUID) error
	// TryConsume marks the quote as used once. Returns false if already consumed.
	TryConsume(ctx context.Context, quoteID uuid.UUID, ttl time.Duration) (bool, error)
	// ReleaseConsume clears the consumed flag so checkout can retry after a failed create.
	ReleaseConsume(ctx context.Context, quoteID uuid.UUID) error
	IsConsumed(ctx context.Context, quoteID uuid.UUID) (bool, error)
	// Active quote pointer per customer (at most one reusable checkout quote).
	GetActiveID(ctx context.Context, customerID uuid.UUID) (uuid.UUID, error)
	SetActiveID(ctx context.Context, customerID, quoteID uuid.UUID, ttl time.Duration) error
	ClearActiveID(ctx context.Context, customerID uuid.UUID) error
}

// OrderRiderBroadcaster is implemented by the riders module to receive order lifecycle events.
// Defined here so orders/app has no import dependency on riders/domain/ports.
type OrderRiderBroadcaster interface {
	BroadcastNewOrder(ctx context.Context, order models.Order)
	BroadcastOrderUpdate(ctx context.Context, order models.Order)
}

// RideEarningRecorder is implemented by the finance module to record rider earnings on delivery.
// Defined here to avoid a cross-domain import from orders → riders.
type RideEarningRecorder interface {
	// deliveryFee está en USD; bcvRate es la tasa a aplicar para convertir y
	// acreditar en Bs (moneda nativa del balance de riders).
	RecordRideEarning(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, totalPrice, bcvRate float64, method string) error
	// RecordRideCashCollected liquida un domicilio con efectivo parcial.
	// El balance del rider se mueve por su pago menos el efectivo que cobró.
	RecordRideCashCollected(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, cashUSD, bcvRate float64) error
}

// ShopCommissionRecorder is implemented by shops/finance to record the shop's commission on delivery completion.
// Defined here to avoid a cross-domain import from orders → shops.
// isZeroCommission (días 1-8 desde la primera apertura) hace que el negocio reciba el 100% en vez del 90% habitual.
// isLaunchPromo + promoSubsidyAmount (solo día 1) registran el desembolso real que la plataforma puso para cubrir el 50% del cliente.
type ShopCommissionRecorder interface {
	RecordSaleCommission(ctx context.Context, shopID uuid.UUID, orderID uuid.UUID, totalPrice, deliveryFee float64, method string, isLaunchPromo, isZeroCommission bool, promoSubsidyAmount float64) error
}

// ManagerConn abstracts the WebSocket transport for order-manager connections.
type ManagerConn interface {
	Send(msg []byte)
	Close()
}

type AppManagerOrder interface {
	Register(ctx context.Context, connectionID uuid.UUID, conn ManagerConn, isShop bool) error
	Unregister(connectionID uuid.UUID, conn ManagerConn, isShop bool)
	BroadcastOrder(order models.Order) error
	BroadcastOrderToCustomer(order models.Order) error
	BroadcastShopStatus(shopID uuid.UUID, open bool) error
	// NotifyCustomer envía un payload arbitrario a las conexiones del customer (p. ej. refund_request_update).
	NotifyCustomer(customerID uuid.UUID, msg []byte) error
	// NotifyRiderOrderLocation envía {"type":"rider_location",...} al cliente de la orden
	// y guarda la última ubicación para reenviarla cuando el cliente se conecta.
	NotifyRiderOrderLocation(ctx context.Context, customerID, orderID uuid.UUID, lat, lng float64, updatedAt time.Time)
}
