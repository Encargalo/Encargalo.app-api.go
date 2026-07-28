package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	jsonutil "Encargalo.app-api.go/internal/pkg/json"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	portsShops "Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
)

type staffShopApp struct {
	repo             portsShops.StaffShopRepo
	categoriesSeeder portsShops.ShopCategoriesSeeder
	uploader         uploadimage.UploadImage
}

func NewStaffShopApp(repo portsShops.StaffShopRepo, categoriesSeeder portsShops.ShopCategoriesSeeder, uploader uploadimage.UploadImage) portsShops.StaffShopApp {
	return &staffShopApp{repo: repo, categoriesSeeder: categoriesSeeder, uploader: uploader}
}

// resolveImageField sube a Cloudinary el data-URL recibido en un campo
// Nullable y devuelve la SecureURL a persistir. Un valor null o vacío limpia
// el campo (string vacío); un campo omitido no debe llegar aquí (ver Set).
func (a *staffShopApp) resolveImageField(ctx context.Context, n jsonutil.Nullable[string]) (string, error) {
	if !n.Valid {
		return "", nil
	}
	trimmed := strings.TrimSpace(n.Value)
	if trimmed == "" {
		return "", nil
	}
	if a.uploader == nil {
		return "", errcustom.ErrUploadingImage
	}
	resp, err := a.uploader.UploadImage(ctx, trimmed)
	if err != nil || resp == nil || resp.SecureURL == "" {
		return "", errcustom.ErrUploadingImage
	}
	return resp.SecureURL, nil
}

func (a *staffShopApp) FindOwnerCandidate(ctx context.Context, phone string) (dtos.OwnerCandidateResponse, error) {
	userID, name, sur, ph, email, err := a.repo.FindOwnerCandidateByPhone(ctx, strings.TrimSpace(phone))
	if err != nil {
		return dtos.OwnerCandidateResponse{}, err
	}
	hasRole, err := a.repo.HasAnyOwnerRole(ctx, userID)
	if err != nil {
		return dtos.OwnerCandidateResponse{}, err
	}
	return dtos.OwnerCandidateResponse{
		UserID:       userID.String(),
		Name:         name,
		SurName:      sur,
		Phone:        ph,
		Email:        email,
		HasOwnerRole: hasRole,
	}, nil
}

func (a *staffShopApp) CreateDraftShop(ctx context.Context, req dtos.CreateDraftShopRequest) (dtos.StaffShopResponse, error) {
	shop := &shopsmodels.Shop{
		OwnerID: req.OwnerID,
		Name:    strings.TrimSpace(req.Name),
		Type:    req.Type,
		Status:  shopsmodels.ShopStatusDraft,
	}
	if err := a.repo.CreateDraftShop(ctx, shop); err != nil {
		return dtos.StaffShopResponse{}, err
	}
	// El owner ya puede entrar a ver su negocio en draft, aunque no pueda
	// abrirlo (gate en ShopsApp.SetOpenClose).
	if err := a.repo.EnsureOwnerRole(ctx, req.OwnerID, shop.ID); err != nil {
		return dtos.StaffShopResponse{}, err
	}
	// Best-effort: el shop y el rol ya quedaron creados; si el sembrado de
	// categorías falla, el staff puede agregarlas a mano después.
	if a.categoriesSeeder != nil {
		if err := a.categoriesSeeder.SeedDefaultCategories(ctx, shop.ID, shop.Type); err != nil {
			slog.Error("error al sembrar categorías default del shop", "error", err, "shopID", shop.ID, "shopType", shop.Type)
		}
	}
	return a.GetShop(ctx, shop.ID)
}

func (a *staffShopApp) GetShop(ctx context.Context, shopID uuid.UUID) (dtos.StaffShopResponse, error) {
	shop, err := a.repo.GetShopByID(ctx, shopID)
	if err != nil {
		return dtos.StaffShopResponse{}, err
	}
	return shop.ToStaffDTO(), nil
}

func (a *staffShopApp) ListShops(ctx context.Context, status shopsmodels.ShopStatus, page, limit int) (dtos.StaffShopsResponse, error) {
	if status == "" {
		status = shopsmodels.ShopStatusDraft
	}
	shops, total, err := a.repo.ListShopsByStatus(ctx, status, page, limit)
	if err != nil {
		return dtos.StaffShopsResponse{}, err
	}
	out := make([]dtos.StaffShopResponse, 0, len(shops))
	for i := range shops {
		out = append(out, shops[i].ToStaffDTO())
	}
	return dtos.StaffShopsResponse{Shops: out, Total: total, Page: page, Limit: limit}, nil
}

// UpdateDraftShop opera sobre el shop sin importar su status: además del
// onboarding en draft, staff usa este mismo endpoint para corregir datos de
// shops ya activos (nombre, dirección, logo, etc.) mientras estén cerrados —
// el único guard es Opened, no Status. Ver shopsmodels.Shop.Status.
func (a *staffShopApp) UpdateDraftShop(ctx context.Context, shopID uuid.UUID, req dtos.UpdateDraftShopRequest) (dtos.StaffShopResponse, error) {
	shop, err := a.repo.GetShopByID(ctx, shopID)
	if err != nil {
		return dtos.StaffShopResponse{}, err
	}

	if shop.Opened {
		return dtos.StaffShopResponse{}, errcustom.ErrShopOpen
	}

	if req.Name != nil {
		shop.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		shop.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Address != nil {
		shop.Address = strings.TrimSpace(*req.Address)
	}
	if req.HomePhone != nil {
		shop.HomePhone = strings.TrimSpace(*req.HomePhone)
	}
	if req.Latitude.Set {
		shop.Latitude = nil
		if req.Latitude.Valid {
			lat := req.Latitude.Value
			shop.Latitude = &lat
		}
	}
	if req.Longitude.Set {
		shop.Longitude = nil
		if req.Longitude.Valid {
			lon := req.Longitude.Value
			shop.Longitude = &lon
		}
	}
	if req.LogoImage.Set {
		logoURL, err := a.resolveImageField(ctx, req.LogoImage)
		if err != nil {
			return dtos.StaffShopResponse{}, err
		}
		shop.LogoImage = logoURL
	}
	if req.Banner.Set {
		bannerURL, err := a.resolveImageField(ctx, req.Banner)
		if err != nil {
			return dtos.StaffShopResponse{}, err
		}
		shop.Banner = bannerURL
	}

	if err := a.repo.UpdateDraftShop(ctx, shop); err != nil {
		return dtos.StaffShopResponse{}, err
	}
	return a.GetShop(ctx, shopID)
}

func (a *staffShopApp) ActivateShop(ctx context.Context, shopID, staffID uuid.UUID) (dtos.StaffShopResponse, error) {
	shop, err := a.repo.GetShopByID(ctx, shopID)
	if err != nil {
		return dtos.StaffShopResponse{}, err
	}
	if shop.Status == shopsmodels.ShopStatusActive {
		return shop.ToStaffDTO(), nil
	}
	if !shop.Completion().IsComplete {
		return dtos.StaffShopResponse{}, errcustom.ErrShopOnboardingIncomplete
	}
	if err := a.repo.ActivateShop(ctx, shopID, staffID, time.Now()); err != nil {
		return dtos.StaffShopResponse{}, err
	}
	return a.GetShop(ctx, shopID)
}
