package ports

import (
	"context"

	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"github.com/google/uuid"
)

// ZoneFareRulesApp administra las zonas de tarifa desde el CRM: crear,
// reemplazar polígono/precios, listar, consultar y desactivar. No decide
// tarifas de viajes — eso lo hace tripsApp.fareEstimate leyendo
// ZoneFareRulesRepo.FindActiveByCell directamente.
type ZoneFareRulesApp interface {
	CreateZoneFareRule(ctx context.Context, req dtos.ZoneFareRuleRequest) (dtos.ZoneFareRuleResponse, error)
	UpdateZoneFareRule(ctx context.Context, zoneID uuid.UUID, req dtos.ZoneFareRuleRequest) (dtos.ZoneFareRuleResponse, error)
	ListZoneFareRules(ctx context.Context) (dtos.ZoneFareRulesResponse, error)
	GetZoneFareRule(ctx context.Context, zoneID uuid.UUID) (dtos.ZoneFareRuleResponse, error)
	// DeactivateZoneFareRule es un soft delete: ver models.ZoneFareRule.Active.
	DeactivateZoneFareRule(ctx context.Context, zoneID uuid.UUID) error
}

// ZoneFareRulesRepo persiste zonas y sus celdas H3. FindActiveByCell es la
// única lectura caliente: tripsApp.fareEstimate la consulta en cada cotización.
type ZoneFareRulesRepo interface {
	Create(ctx context.Context, rule *models.ZoneFareRule, cells []string) error
	Update(ctx context.Context, rule *models.ZoneFareRule, cells []string) error
	GetByID(ctx context.Context, zoneID uuid.UUID) (*models.ZoneFareRule, error)
	// GetCellsByZoneID devuelve las celdas vigentes de la zona, para que el CRM pueda redibujar el polígono.
	GetCellsByZoneID(ctx context.Context, zoneID uuid.UUID) ([]string, error)
	List(ctx context.Context) ([]models.ZoneFareRule, error)
	Deactivate(ctx context.Context, zoneID uuid.UUID) error
	// FindActiveByCell resuelve la zona activa (si alguna) cuya celda cubre
	// ese punto. errcustom.ErrNotFound si ninguna celda activa coincide.
	FindActiveByCell(ctx context.Context, h3Cell string) (*models.ZoneFareRule, error)
}
