package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	DevicePlatformAndroid = "android"
	DevicePlatformIOS     = "ios"
)

// RiderDeviceToken es el token FCM del dispositivo del rider, usado para
// mandarle push cuando se le oferta un viaje o un pedido. Un solo token activo
// por rider — un login nuevo pisa el anterior.
type RiderDeviceToken struct {
	bun.BaseModel `bun:"table:riders.device_tokens"`

	RiderID   uuid.UUID `bun:"rider_id,pk,type:uuid"`
	FCMToken  string    `bun:"fcm_token,notnull"`
	Platform  string    `bun:"platform,notnull"`
	UpdatedAt time.Time `bun:"updated_at,default:now()"`
}
