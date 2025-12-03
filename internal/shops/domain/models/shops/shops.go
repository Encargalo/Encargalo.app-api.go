package shops

import (
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/entity"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Shops []Shop

type Shop struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID              uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	OwnerID         uuid.UUID  `bun:"owner_id,type:uuid,notnull"`
	LicenseID       uuid.UUID  `bun:"license_id,type:uuid,nullzero,default:'6e02bcef-0372-4e80-ad1b-ed5d348dde0e'"`
	Name            string     `bun:"name,type:varchar,notnull"`
	Tag             string     `bun:"tag,type:varchar"`
	Address         string     `bun:"address,type:varchar"`
	HomePhone       string     `bun:"home_phone,type:varchar,notnull"`
	Latitude        *float64   `bun:"latitude,type:float"`
	Longitude       *float64   `bun:"longitude,type:float"`
	LogoImage       string     `bun:"logo_image,type:varchar"`
	Banner          string     `bun:"banner,type:varchar"`
	Opened          bool       `bun:"opened,default:false"`
	Type            string     `bun:"type,type:varchar,notnull"`
	Score           float64    `bun:"score,default:0"`
	Coverage        float32    `bun:"coverage,default:0"`
	MinimunDelivery int        `bun:"minimum_delivery,default:0"`
	MaximumDelivery int        `bun:"maximum_delivery,default:0"`
	LicenseStatus   string     `bun:"license_status,type:varchar,default:'active'"`
	CreatedAt       time.Time  `bun:"created_at,default:current_timestamp"`
	UpdatedAt       time.Time  `bun:"updated_at,default:current_timestamp"`
	DeletedAt       *time.Time `bun:"deleted_at,nullzero"`
}

func (s *Shop) ToDomainDTO(distance entity.ShopDistance) dtos.ShopResponse {
	return dtos.ShopResponse{
		ID:          s.ID,
		Name:        s.Name,
		Tag:         s.Tag,
		Address:     s.Address,
		Latitude:    *s.Latitude,
		Longitude:   *s.Longitude,
		HomePhone:   s.HomePhone,
		Logo:        s.LogoImage,
		Banner:      s.Banner,
		Opened:      s.Opened,
		Type:        s.Type,
		Score:       s.Score,
		DeliveryFee: calculateDeliveryFee(int(distance.Distance), s.Coverage*1000, s.MaximumDelivery, s.MinimunDelivery),
		Time:        int(distance.Duration) + 20,
	}
}

func (s *Shops) ToDomainDTO(distances []entity.ShopDistance) dtos.ShopsResponse {
	var shops dtos.ShopsResponse

	distMap := make(map[uuid.UUID]entity.ShopDistance)
	for _, d := range distances {
		distMap[d.ShopID] = d
	}

	for _, shop := range *s {
		distance, ok := distMap[shop.ID]
		if !ok {
			continue
		}
		shops.Add(shop.ToDomainDTO(distance))
	}

	return shops
}

func calculateDeliveryFee(distanciaMetros int, distanciaMaximaMetros float32, precioMaximo int, precioMinimo int) int {
	precioPorMetro := float32(precioMaximo) / distanciaMaximaMetros
	total := float32(distanciaMetros) * precioPorMetro

	if total < float32(precioMinimo) {
		return precioMinimo
	}

	return int(total)
}
