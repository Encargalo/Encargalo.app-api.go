package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
)

// StaffShopRepo persiste el ciclo de onboarding de un shop dado de alta por
// staff (draft → active). Vive separado de ShopsRepo (cara pública/negocio)
// igual que riders separa RiderProfileRepo del resto de riders.
type StaffShopRepo interface {
	FindOwnerCandidateByPhone(ctx context.Context, phone string) (userID uuid.UUID, name, surName, phoneOut, email string, err error)
	// HasAnyOwnerRole indica si la cuenta ya es Owner de algún shop (informativo
	// para el staff antes de vincularla a uno nuevo).
	HasAnyOwnerRole(ctx context.Context, userID uuid.UUID) (bool, error)
	// EnsureOwnerRole agrega el rol Owner (con este shop_id) a la cuenta sin
	// remover los roles que ya tenga.
	EnsureOwnerRole(ctx context.Context, userID, shopID uuid.UUID) error
	CreateDraftShop(ctx context.Context, shop *shopsmodels.Shop) error
	// GetShopByID busca el shop sin filtrar por status: lo usan tanto el
	// onboarding (drafts) como la edición/activación de shops ya activos.
	GetShopByID(ctx context.Context, shopID uuid.UUID) (*shopsmodels.Shop, error)
	ListShopsByStatus(ctx context.Context, status shopsmodels.ShopStatus, page, limit int) (shops shopsmodels.Shops, total int, err error)
	UpdateDraftShop(ctx context.Context, shop *shopsmodels.Shop) error
	ActivateShop(ctx context.Context, shopID, staffID uuid.UUID, activatedAt time.Time) error
	// GetShopSchedule/ReplaceShopSchedule gestionan el horario semanal
	// (business.schedules) de cualquier shop por id, para que staff pueda
	// configurarlo en nombre del owner.
	GetShopSchedule(ctx context.Context, shopID uuid.UUID) ([]dtos.ScheduleResponse, error)
	ReplaceShopSchedule(ctx context.Context, shopID uuid.UUID, schedules []dtos.ScheduleInput) error
}

// ShopCategoriesSeeder crea las categorías base de un shop nuevo (una por
// cada macro-categoría de su tipo), implementado por products/app — shops no
// depende del paquete products directamente, solo de esta interfaz pequeña.
type ShopCategoriesSeeder interface {
	SeedDefaultCategories(ctx context.Context, shopID uuid.UUID, shopType string) error
}

// StaffShopApp orquesta el onboarding: crear en draft, completar por partes,
// activar. La creación ya vincula el rol Owner (para que el owner pueda
// entrar y ver su negocio en draft, aunque no pueda abrirlo) y siembra las
// categorías base según el tipo de negocio.
type StaffShopApp interface {
	FindOwnerCandidate(ctx context.Context, phone string) (dtos.OwnerCandidateResponse, error)
	CreateDraftShop(ctx context.Context, req dtos.CreateDraftShopRequest) (dtos.StaffShopResponse, error)
	GetShop(ctx context.Context, shopID uuid.UUID) (dtos.StaffShopResponse, error)
	ListShops(ctx context.Context, status shopsmodels.ShopStatus, page, limit int) (dtos.StaffShopsResponse, error)
	UpdateDraftShop(ctx context.Context, shopID uuid.UUID, req dtos.UpdateDraftShopRequest) (dtos.StaffShopResponse, error)
	ActivateShop(ctx context.Context, shopID, staffID uuid.UUID) (dtos.StaffShopResponse, error)
	GetShopSchedule(ctx context.Context, shopID uuid.UUID) ([]dtos.ScheduleResponse, error)
	SetShopSchedule(ctx context.Context, shopID uuid.UUID, schedules []dtos.ScheduleInput) error
}
