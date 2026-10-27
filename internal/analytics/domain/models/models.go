package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ETLEntity string

const (
	EntityDimShop           ETLEntity = "dim_shop"
	EntityDimUser           ETLEntity = "dim_user"
	EntityDimRider          ETLEntity = "dim_rider"
	EntityCubeOrder         ETLEntity = "cube_order"
	EntityCubeTrip          ETLEntity = "cube_trip"
	EntityFactTrackingEvent ETLEntity = "fact_tracking_event"
	EntityCubeFareQuote     ETLEntity = "cube_fare_quote"
	EntityCubeDeliveryQuote ETLEntity = "cube_delivery_quote"
)

var DefaultETLEntities = []ETLEntity{
	EntityDimShop,
	EntityDimUser,
	EntityDimRider,
	EntityCubeOrder,
	EntityCubeTrip,
	EntityFactTrackingEvent,
	EntityCubeFareQuote,
	EntityCubeDeliveryQuote,
}

// CronETLEntities es lo que sincroniza el cron del worker. Cada entidad sale
// de esta lista a medida que gana su propio productor de eventos (ver
// ports.ETLApp.SyncXByID) y se resincroniza en tiempo real en vez de esperar
// al batch horario. EntityFactTrackingEvent se queda aquí a propósito: es
// telemetría genérica de alto volumen sin status de negocio, no amerita
// tiempo real ni inflar el stream compartido con notifications.
var CronETLEntities = []ETLEntity{
	EntityDimShop,
	EntityDimUser,
	EntityDimRider,
	EntityCubeTrip,
	EntityFactTrackingEvent,
	EntityCubeDeliveryQuote,
}

type DateRange struct {
	From time.Time
	To   time.Time
}

type SyncLog struct {
	bun.BaseModel `bun:"table:analytics.etl_sync_logs"`

	Entity       ETLEntity `bun:"entity,pk"`
	LastSyncedTo time.Time `bun:"last_synced_to,notnull"`
	UpdatedAt    time.Time `bun:"updated_at,notnull"`
}

type DimShop struct {
	bun.BaseModel `bun:"table:analytics.dim_shop"`

	ShopID        uuid.UUID `bun:"shop_id,pk,type:uuid"`
	Name          string    `bun:"name,notnull"`
	Slug          string    `bun:"slug"`
	Address       string    `bun:"address"`
	Type          string    `bun:"type"`
	LicenseStatus string    `bun:"license_status"`
	Opened        bool      `bun:"opened,notnull"`
	CreatedAt     time.Time `bun:"created_at,notnull"`
	UpdatedAt     time.Time `bun:"updated_at,notnull"`
	SyncedAt      time.Time `bun:"synced_at,notnull"`
	H3Cell        *string   `bun:"h3_cell"`
}

type DimUser struct {
	bun.BaseModel `bun:"table:analytics.dim_user"`

	UserID       uuid.UUID `bun:"user_id,pk,type:uuid"`
	Name         string    `bun:"name,notnull"`
	SurName      *string   `bun:"sur_name"`
	Phone        *string   `bun:"phone"`
	Email        *string   `bun:"email"`
	Roles        []string  `bun:"roles,array,notnull"`
	Enabled      bool      `bun:"enabled,notnull"`
	RegisteredAt time.Time `bun:"registered_at,notnull"`
	UpdatedAt    time.Time `bun:"updated_at,notnull"`
	SyncedAt     time.Time `bun:"synced_at,notnull"`
}

type DimRider struct {
	bun.BaseModel `bun:"table:analytics.dim_rider"`

	RiderID      uuid.UUID  `bun:"rider_id,pk,type:uuid"`
	Name         string     `bun:"name,notnull"`
	SurName      *string    `bun:"sur_name"`
	Phone        *string    `bun:"phone"`
	Email        *string    `bun:"email"`
	Status       string     `bun:"status,notnull"`
	VehicleType  *string    `bun:"vehicle_type"`
	RegisteredAt time.Time  `bun:"registered_at,notnull"`
	ActivatedAt  *time.Time `bun:"activated_at"`
	UpdatedAt    time.Time  `bun:"updated_at,notnull"`
	SyncedAt     time.Time  `bun:"synced_at,notnull"`
}

type FactTrackingEvent struct {
	bun.BaseModel `bun:"table:analytics.fact_tracking_event"`

	ID        uuid.UUID      `bun:"id,pk,type:uuid"`
	Type      string         `bun:"type,notnull"`
	Params    map[string]any `bun:"params,type:jsonb,notnull"`
	CreatedAt time.Time      `bun:"created_at,notnull"`
	SyncedAt  time.Time      `bun:"synced_at,notnull"`
}

type CubeOrder struct {
	bun.BaseModel `bun:"table:analytics.cube_order"`

	OrderID            uuid.UUID  `bun:"order_id,pk,type:uuid"`
	QuoteID            *uuid.UUID `bun:"quote_id,type:uuid"`
	ShopID             uuid.UUID  `bun:"shop_id,type:uuid,notnull"`
	CustomerID         uuid.UUID  `bun:"customer_id,type:uuid,notnull"`
	RiderID            *uuid.UUID `bun:"rider_id,type:uuid"`
	Status             string     `bun:"status,notnull"`
	MethodPayment      string     `bun:"method_payment"`
	TotalPrice         float64    `bun:"total_price,notnull"`
	DeliveryFee        float64    `bun:"delivery_fee,notnull"`
	GMV                float64    `bun:"gmv,notnull"`
	PromoSubsidyAmount float64    `bun:"promo_subsidy_amount,notnull"`
	IsLaunchPromo      bool       `bun:"is_launch_promo,notnull"`
	IsZeroCommission   bool       `bun:"is_zero_commission,notnull"`
	OrderDate          time.Time  `bun:"order_date,notnull"`
	CreatedAt          time.Time  `bun:"created_at,notnull"`
	SyncedAt           time.Time  `bun:"synced_at,notnull"`
	H3Cell             *string    `bun:"h3_cell"`
}

// Fare es nulo hasta completar el trip; PickupH3Cell/DropoffH3Cell son resolución 7 (zona), no la 8 de trips.trips.pickup_h3_cell.
type CubeTrip struct {
	bun.BaseModel `bun:"table:analytics.cube_trip"`

	TripID        uuid.UUID  `bun:"trip_id,pk,type:uuid"`
	QuoteID       *uuid.UUID `bun:"quote_id,type:uuid"`
	CustomerID    uuid.UUID  `bun:"customer_id,type:uuid,notnull"`
	RiderID       *uuid.UUID `bun:"rider_id,type:uuid"`
	Status        string     `bun:"status,notnull"`
	Fare          *float64   `bun:"fare"`
	PickupH3Cell  *string    `bun:"pickup_h3_cell"`
	DropoffH3Cell *string    `bun:"dropoff_h3_cell"`
	TripDate      time.Time  `bun:"trip_date,notnull"`
	CreatedAt     time.Time  `bun:"created_at,notnull"`
	SyncedAt      time.Time  `bun:"synced_at,notnull"`
}

// CubeFareQuote es la cotización mostrada al cliente, haya convertido o no.
// QuoteDate/QuoteHour vienen en hora de negocio, no UTC: la hora del día solo
// significa algo contra el reloj local del cliente.
type CubeFareQuote struct {
	bun.BaseModel `bun:"table:analytics.cube_fare_quote"`

	QuoteID        uuid.UUID `bun:"quote_id,pk,type:uuid"`
	CustomerID     uuid.UUID `bun:"customer_id,type:uuid,notnull"`
	DistanceMeters float64   `bun:"distance_meters,notnull"`
	DurationMin    int64     `bun:"duration_min,notnull"`
	FareUSD        float64   `bun:"fare_usd,notnull"`
	FareBand       string    `bun:"fare_band,notnull"`
	PickupH3Cell   *string   `bun:"pickup_h3_cell"`
	DropoffH3Cell  *string   `bun:"dropoff_h3_cell"`
	QuoteDate      time.Time `bun:"quote_date,notnull"`
	QuoteHour      int16     `bun:"quote_hour,notnull"`
	CreatedAt      time.Time `bun:"created_at,notnull"`
	SyncedAt       time.Time `bun:"synced_at,notnull"`
}

// CubeDeliveryQuote es la cotización de delivery mostrada al cliente, haya
// convertido o no. QuoteDate/QuoteHour vienen en hora de negocio, no UTC.
type CubeDeliveryQuote struct {
	bun.BaseModel `bun:"table:analytics.cube_delivery_quote"`

	QuoteID        uuid.UUID `bun:"quote_id,pk,type:uuid"`
	CustomerID     uuid.UUID `bun:"customer_id,type:uuid,notnull"`
	ShopID         uuid.UUID `bun:"shop_id,type:uuid,notnull"`
	DistanceMeters float64   `bun:"distance_meters,notnull"`
	FeeUSD         float64   `bun:"fee_usd,notnull"`
	FareBand       string    `bun:"fare_band,notnull"`
	QuoteDate      time.Time `bun:"quote_date,notnull"`
	QuoteHour      int16     `bun:"quote_hour,notnull"`
	CreatedAt      time.Time `bun:"created_at,notnull"`
	SyncedAt       time.Time `bun:"synced_at,notnull"`
}

type ETLEntityResult struct {
	Entity        ETLEntity `json:"entity"`
	Success       bool      `json:"success"`
	Rows          int       `json:"rows"`
	Error         string    `json:"error,omitempty"`
	ExecutionTime int64     `json:"execution_time_ms"`
}

type BatchResult struct {
	From    time.Time         `json:"from"`
	To      time.Time         `json:"to"`
	Results []ETLEntityResult `json:"results"`
	Message string            `json:"message"`
}

// GMV es el valor de mercancía: total del pedido menos el delivery fee (nunca negativo).
func GMV(totalPrice, deliveryFee float64) float64 {
	gmv := totalPrice - deliveryFee
	if gmv < 0 {
		return 0
	}
	return gmv
}
