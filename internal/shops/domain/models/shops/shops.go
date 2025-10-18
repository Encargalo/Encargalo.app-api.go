package shops

import (
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Shops []Shop

type Shop struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID            uuid.UUID  `bun:"type:uuid,pk,default:uuid_generate_v4()"`
	OwnerID       uuid.UUID  `bun:"type:uuid,notnull"`
	LicenseID     uuid.UUID  `bun:"type:uuid,nullzero,default:'6e02bcef-0372-4e80-ad1b-ed5d348dde0e'"`
	Name          string     `bun:"type:varchar,notnull"`
	Tag           string     `bun:"type:varchar"`
	Address       string     `bun:"type:varchar"`
	HomePhone     string     `bun:"type:varchar,notnull"`
	Latitude      *float64   `bun:"type:float"`
	Longitude     *float64   `bun:"type:float"`
	LogoImage     string     `bun:"type:varchar"`
	Banner        string     `bun:"type:varchar"`
	Opened        bool       `bun:"default:false"`
	Type          string     `bun:"type:varchar,notnull"`
	Score         float64    `bun:"default:0"`
	LicenseStatus string     `bun:"type:varchar,default:'active'"`
	CreatedAt     time.Time  `bun:"default:current_timestamp"`
	UpdatedAt     time.Time  `bun:"default:current_timestamp"`
	DeletedAt     *time.Time `bun:"default:null"`
}

func (s *Shop) ToDomainDTO() dtos.ShopResponse {
	return dtos.ShopResponse{
		ID:        s.ID,
		Name:      s.Name,
		Tag:       s.Tag,
		Address:   s.Address,
		HomePhone: s.HomePhone,
		Logo:      s.LogoImage,
		Banner:    s.Banner,
		Opened:    s.Opened,
		Type:      s.Type,
		Score:     s.Score,
	}
}

func (s *Shops) ToDomainDTO() dtos.ShopsResponse {
	var shops dtos.ShopsResponse

	for _, v := range *s {
		shops.Add(v.ToDomainDTO())
	}

	return shops

}
