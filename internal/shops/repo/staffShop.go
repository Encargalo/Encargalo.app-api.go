package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"Encargalo.app-api.go/internal/shops/domain/ports"
	userModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type staffShopRepo struct {
	db *bun.DB
}

func NewStaffShopRepo(db *bun.DB) ports.StaffShopRepo {
	return &staffShopRepo{db: db}
}

func (r *staffShopRepo) FindOwnerCandidateByPhone(ctx context.Context, phone string) (uuid.UUID, string, string, string, string, error) {
	phone = strings.TrimSpace(phone)
	var acc userModels.Account
	err := r.db.NewSelect().
		Model(&acc).
		Where("phone = ?", phone).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, "", "", "", "", errcustom.ErrNotFound
		}
		slog.Error("staffShop FindOwnerCandidateByPhone failed", "phone", phone, "error", err)
		return uuid.Nil, "", "", "", "", errcustom.Unexpected(err)
	}
	sur, ph, em := "", "", ""
	if acc.SurName != nil {
		sur = *acc.SurName
	}
	if acc.Phone != nil {
		ph = *acc.Phone
	}
	if acc.Email != nil {
		em = *acc.Email
	}
	return acc.ID, acc.Name, sur, ph, em, nil
}

func (r *staffShopRepo) HasAnyOwnerRole(ctx context.Context, userID uuid.UUID) (bool, error) {
	exists, err := r.db.NewSelect().
		TableExpr("users.user_roles ur").
		Join("JOIN users.roles ro ON ro.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Where("ro.name = ?", "Owner").
		Where("ur.enabled = TRUE").
		Exists(ctx)
	if err != nil {
		slog.Error("staffShop HasAnyOwnerRole failed", "userID", userID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return exists, nil
}

// EnsureOwnerRole agrega (o reactiva) la fila Owner+shop_id de la cuenta sin
// tocar sus demás roles/shops. users.user_roles tiene UNIQUE(user_id, role_id,
// shop_id), así que una misma cuenta puede ser Owner de varios shops.
func (r *staffShopRepo) EnsureOwnerRole(ctx context.Context, userID, shopID uuid.UUID) error {
	_, err := r.db.NewRaw(`
		INSERT INTO users.user_roles (user_id, role_id, shop_id, enabled)
		SELECT ?, id, ?, TRUE FROM users.roles WHERE name = 'Owner'
		ON CONFLICT (user_id, role_id, shop_id) DO UPDATE SET enabled = TRUE
	`, userID, shopID).Exec(ctx)
	if err != nil {
		slog.Error("staffShop EnsureOwnerRole failed", "userID", userID, "shopID", shopID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *staffShopRepo) CreateDraftShop(ctx context.Context, shop *shopsmodels.Shop) error {
	shop.Status = shopsmodels.ShopStatusDraft
	if _, err := r.db.NewInsert().Model(shop).Exec(ctx); err != nil {
		slog.Error("staffShop CreateDraftShop failed", "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *staffShopRepo) GetShopByID(ctx context.Context, shopID uuid.UUID) (*shopsmodels.Shop, error) {
	var shop shopsmodels.Shop
	err := r.db.NewSelect().
		Model(&shop).
		Where("id = ?", shopID).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("staffShop GetShopByID failed", "shopID", shopID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return &shop, nil
}

const maxListShopsLimit = 50

func (r *staffShopRepo) ListShopsByStatus(ctx context.Context, status shopsmodels.ShopStatus, page, limit int) (shopsmodels.Shops, int, error) {
	if limit > maxListShopsLimit {
		limit = maxListShopsLimit
	}
	offset := (page - 1) * limit

	var shops shopsmodels.Shops
	total, err := r.db.NewSelect().
		Model(&shops).
		Where("status = ?", status).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		slog.Error("staffShop ListShopsByStatus failed", "status", status, "error", err)
		return nil, 0, errcustom.Unexpected(err)
	}
	return shops, total, nil
}

// UpdateDraftShop reescribe los campos editables del shop (el caller ya
// mergeó los cambios parciales sobre el modelo obtenido de GetDraftShopByID).
func (r *staffShopRepo) UpdateDraftShop(ctx context.Context, shop *shopsmodels.Shop) error {
	_, err := r.db.NewUpdate().
		Model(shop).
		Column("name", "slug", "address", "home_phone", "latitude", "longitude", "h3_cell", "logo_image", "banner").
		Set("updated_at = now()").
		WherePK().
		Exec(ctx)
	if err != nil {
		slog.Error("staffShop UpdateDraftShop failed", "shopID", shop.ID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *staffShopRepo) ActivateShop(ctx context.Context, shopID, staffID uuid.UUID, activatedAt time.Time) error {
	_, err := r.db.NewUpdate().
		TableExpr("business.shops").
		Set("status = ?", shopsmodels.ShopStatusActive).
		Set("activated_at = ?", activatedAt).
		Set("activated_by = ?", staffID).
		Set("updated_at = now()").
		Where("id = ?", shopID).
		Exec(ctx)
	if err != nil {
		slog.Error("staffShop ActivateShop failed", "shopID", shopID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}
