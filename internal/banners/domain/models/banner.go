package models

import (
	"time"

	"Encargalo.app-api.go/internal/banners/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Banners []Banner

type Banner struct {
	bun.BaseModel `bun:"table:marketing.banners" swaggerignore:"true"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ImageURL  string    `bun:"image_url,notnull" json:"image_url"`
	LinkURL   *string   `bun:"link_url" json:"link_url,omitempty"`
	Position  int       `bun:"position,notnull,default:0" json:"position"`
	Active    bool      `bun:"active,notnull,default:true" json:"active"`
	CreatedAt time.Time `bun:"created_at,default:now()" json:"-"`
	UpdatedAt time.Time `bun:"updated_at,default:now()" json:"-"`
}

func (b *Banner) ToDTO() dtos.BannerResponse {
	return dtos.BannerResponse{
		ID:       b.ID,
		ImageURL: b.ImageURL,
		LinkURL:  b.LinkURL,
		Position: b.Position,
		Active:   b.Active,
	}
}

func (bs Banners) ToDTO() dtos.BannersResponse {
	out := make(dtos.BannersResponse, 0, len(bs))
	for i := range bs {
		out = append(out, bs[i].ToDTO())
	}
	return out
}
