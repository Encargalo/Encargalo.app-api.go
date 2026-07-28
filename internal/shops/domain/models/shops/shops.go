package shops

import (
	"strings"
	"time"

	"Encargalo.app-api.go/internal/shared/deliveryfee"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	"Encargalo.app-api.go/internal/shops/domain/entity"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	CoverageRadiusKm      = 3.0 // km, radio para Restaurant/Supermarket
	TurboCoverageRadiusKm = 2.0 // km, radio para nodos Turbo
)

// ShopStatus es el estado de onboarding del shop (distinto de LicenseStatus,
// que representa vigencia/suspensión de licencia). Ver design.md de
// add-staff-shop-onboarding.
type ShopStatus string

const (
	ShopStatusDraft  ShopStatus = "draft"
	ShopStatusActive ShopStatus = "active"
)

type Shops []Shop

type Shop struct {
	bun.BaseModel `bun:"table:business.shops"`

	ID            uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	OwnerID       uuid.UUID  `bun:"owner_id,type:uuid,notnull"`
	LicenseID     uuid.UUID  `bun:"license_id,type:uuid,nullzero,default:'6e02bcef-0372-4e80-ad1b-ed5d348dde0e'"`
	Name          string     `bun:"name,type:varchar,notnull"`
	Slug          string     `bun:"slug,type:varchar"`
	Address       string     `bun:"address,type:varchar"`
	HomePhone     string     `bun:"home_phone,type:varchar"`
	Latitude      *float64   `bun:"latitude,type:float"`
	Longitude     *float64   `bun:"longitude,type:float"`
	LogoImage     string     `bun:"logo_image,type:varchar"`
	Banner        string     `bun:"banner,type:varchar"`
	Opened        bool       `bun:"opened,default:false"`
	Type          string     `bun:"type,type:varchar,notnull"`
	Score         float64    `bun:"score,default:0"`
	LicenseStatus string     `bun:"license_status,type:varchar,default:'active'"`
	CreatedAt     time.Time  `bun:"created_at,default:current_timestamp"`
	UpdatedAt     time.Time  `bun:"updated_at,default:current_timestamp"`
	DeletedAt     *time.Time `bun:"deleted_at,nullzero"`

	// FirstOpenedAt es el instante exacto en que el negocio se marcó como
	// abierto por primera vez; ancla las dos promos de lanzamiento (día 1 y
	// comisión 0% durante 8 días). Nil hasta la primera apertura.
	FirstOpenedAt        *time.Time `bun:"first_opened_at,nullzero"`
	LaunchPromoEndsAt    *time.Time `bun:"launch_promo_ends_at,nullzero"`
	ZeroCommissionEndsAt *time.Time `bun:"zero_commission_ends_at,nullzero"`

	// Status controla el onboarding hecho por staff: un shop en draft no
	// puede abrir (ver ports.ShopsApp.SetOpenClose). ActivatedBy audita qué
	// staff lo activó.
	Status      ShopStatus `bun:"status,notnull,default:'draft'"`
	ActivatedAt *time.Time `bun:"activated_at,nullzero"`
	ActivatedBy *uuid.UUID `bun:"activated_by,nullzero,type:uuid"`
}

// ShopCompletion es el checklist mínimo para poder activar un shop en draft.
type ShopCompletion struct {
	IsComplete    bool     `json:"is_complete"`
	Missing       []string `json:"missing"`
	FilledCount   int      `json:"filled_count"`
	RequiredCount int      `json:"required_count"`
}

// Completion calcula el checklist mínimo para activar (ver riders.RiderProfile.Completion
// para el mismo patrón). Name y Type ya son obligatorios desde la creación.
func (s *Shop) Completion() ShopCompletion {
	checks := []struct {
		key string
		ok  bool
	}{
		{"slug", strings.TrimSpace(s.Slug) != ""},
		{"logo_image", strings.TrimSpace(s.LogoImage) != ""},
		{"banner", strings.TrimSpace(s.Banner) != ""},
		{"address", strings.TrimSpace(s.Address) != ""},
		{"home_phone", strings.TrimSpace(s.HomePhone) != ""},
		{"coordinates", s.Latitude != nil && s.Longitude != nil},
	}

	missing := make([]string, 0)
	filled := 0
	for _, c := range checks {
		if c.ok {
			filled++
		} else {
			missing = append(missing, c.key)
		}
	}

	return ShopCompletion{
		IsComplete:    len(missing) == 0,
		Missing:       missing,
		FilledCount:   filled,
		RequiredCount: len(checks),
	}
}

// ToDTO convierte el checklist a su forma de respuesta HTTP.
func (c ShopCompletion) ToDTO() dtos.ShopCompletionDTO {
	missing := c.Missing
	if missing == nil {
		missing = []string{}
	}
	return dtos.ShopCompletionDTO{
		IsComplete:    c.IsComplete,
		Missing:       missing,
		FilledCount:   c.FilledCount,
		RequiredCount: c.RequiredCount,
	}
}

// HasActiveLaunchPromo indica si el descuento del 50% al cliente (día 1)
// sigue vigente para este negocio en este instante.
func (s *Shop) HasActiveLaunchPromo() bool {
	return s.LaunchPromoEndsAt != nil && time.Now().Before(*s.LaunchPromoEndsAt)
}

// ToStaffDTO expone el shop tal como lo ve el panel de staff durante el
// onboarding (incluye status/completion, no expuestos en ToDomainDTO).
func (s *Shop) ToStaffDTO() dtos.StaffShopResponse {
	resp := dtos.StaffShopResponse{
		ID:         s.ID,
		OwnerID:    s.OwnerID,
		Name:       s.Name,
		Slug:       s.Slug,
		Address:    s.Address,
		HomePhone:  s.HomePhone,
		Latitude:   s.Latitude,
		Longitude:  s.Longitude,
		LogoImage:  s.LogoImage,
		Banner:     s.Banner,
		Type:       s.Type,
		Status:     string(s.Status),
		Opened:     s.Opened,
		Completion: s.Completion().ToDTO(),
	}
	if !s.CreatedAt.IsZero() {
		resp.CreatedAt = s.CreatedAt.Format(time.RFC3339)
	}
	if !s.UpdatedAt.IsZero() {
		resp.UpdatedAt = s.UpdatedAt.Format(time.RFC3339)
	}
	if s.ActivatedAt != nil {
		activated := s.ActivatedAt.Format(time.RFC3339)
		resp.ActivatedAt = &activated
	}
	return resp
}

func (s *Shop) ToDomainDTO(distance entity.ShopDistance) dtos.ShopResponse {
	return dtos.ShopResponse{
		ID:             s.ID,
		Name:           s.Name,
		Slug:           s.Slug,
		Address:        s.Address,
		Latitude:       *s.Latitude,
		Longitude:      *s.Longitude,
		HomePhone:      s.HomePhone,
		Logo:           s.LogoImage,
		Banner:         s.Banner,
		Opened:         s.Opened,
		Type:           s.Type,
		Score:          s.Score,
		DeliveryFee:    deliveryfee.FromDistance(distance.Distance),
		Time:           prepTimeMinutes(s.Type, distance.Duration),
		HasLaunchPromo: s.HasActiveLaunchPromo(),
	}
}

// prepTimeMinutes calcula el tiempo mostrado al cliente a partir de la
// duración de manejo. Los shops Turbo no suman los 20 minutos de preparación
// que sí aplican a Restaurant/Supermarket (ver design.md de add-turbo-modality).
func prepTimeMinutes(shopType string, durationMin int64) int {
	if shopType == ShopTypeTurbo {
		return int(durationMin)
	}
	return int(durationMin) + 20
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
