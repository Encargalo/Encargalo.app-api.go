package app

import (
	"context"
	"log/slog"
	"strings"

	"Encargalo.app-api.go/internal/banners/domain/dtos"
	"Encargalo.app-api.go/internal/banners/domain/models"
	"Encargalo.app-api.go/internal/banners/domain/ports"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

type bannersApp struct {
	repo     ports.BannersRepo
	uploader uploadimage.UploadImage
}

func NewBannersApp(repo ports.BannersRepo, uploader uploadimage.UploadImage) ports.BannersApp {
	return &bannersApp{repo: repo, uploader: uploader}
}

func (a *bannersApp) uploadImage(ctx context.Context, imageB64 string) (string, error) {
	if a.uploader == nil {
		return "", errcustom.ErrUploadingImage
	}
	resp, err := a.uploader.UploadImage(ctx, strings.TrimSpace(imageB64))
	if err != nil || resp == nil || resp.SecureURL == "" {
		return "", errcustom.ErrUploadingImage
	}
	return resp.SecureURL, nil
}

func (a *bannersApp) CreateBanner(ctx context.Context, req dtos.CreateBannerRequest) (dtos.BannerResponse, error) {
	imageURL, err := a.uploadImage(ctx, req.Image)
	if err != nil {
		return dtos.BannerResponse{}, err
	}

	position := 0
	if req.Position != nil {
		position = *req.Position
	}

	banner := &models.Banner{
		ImageURL: imageURL,
		LinkURL:  req.LinkURL,
		Position: position,
		Active:   true,
	}
	if err := a.repo.CreateBanner(ctx, banner); err != nil {
		return dtos.BannerResponse{}, err
	}
	return banner.ToDTO(), nil
}

func (a *bannersApp) ListBanners(ctx context.Context) (dtos.BannersResponse, error) {
	banners, err := a.repo.ListBanners(ctx)
	if err != nil {
		return nil, err
	}
	return banners.ToDTO(), nil
}

func (a *bannersApp) GetBanner(ctx context.Context, id uuid.UUID) (dtos.BannerResponse, error) {
	banner, err := a.repo.GetBannerByID(ctx, id)
	if err != nil {
		return dtos.BannerResponse{}, err
	}
	return banner.ToDTO(), nil
}

func (a *bannersApp) UpdateBanner(ctx context.Context, id uuid.UUID, req dtos.UpdateBannerRequest) (dtos.BannerResponse, error) {
	banner, err := a.repo.GetBannerByID(ctx, id)
	if err != nil {
		return dtos.BannerResponse{}, err
	}

	previousImageURL := banner.ImageURL
	if req.Image != nil {
		imageURL, err := a.uploadImage(ctx, *req.Image)
		if err != nil {
			return dtos.BannerResponse{}, err
		}
		banner.ImageURL = imageURL
	}
	if req.LinkURL.Set {
		banner.LinkURL = nil
		if req.LinkURL.Valid {
			link := req.LinkURL.Value
			banner.LinkURL = &link
		}
	}
	if req.Position != nil {
		banner.Position = *req.Position
	}
	if req.Active != nil {
		banner.Active = *req.Active
	}

	if err := a.repo.UpdateBanner(ctx, banner); err != nil {
		return dtos.BannerResponse{}, err
	}

	// Best-effort: el banner y su imagen nueva ya quedaron guardados: dejar el
	// asset viejo huérfano en Cloudinary no es motivo para fallar la
	// actualización, solo para avisarlo.
	if req.Image != nil && previousImageURL != "" && previousImageURL != banner.ImageURL {
		if err := a.uploader.DeleteImage(ctx, previousImageURL); err != nil {
			slog.Error("error al borrar la imagen anterior del banner en Cloudinary", "error", err, "bannerID", id)
		}
	}

	return banner.ToDTO(), nil
}

func (a *bannersApp) DeleteBanner(ctx context.Context, id uuid.UUID) error {
	banner, err := a.repo.GetBannerByID(ctx, id)
	if err != nil {
		return err
	}
	if err := a.repo.DeleteBanner(ctx, id); err != nil {
		return err
	}
	if banner.ImageURL != "" && a.uploader != nil {
		if err := a.uploader.DeleteImage(ctx, banner.ImageURL); err != nil {
			slog.Error("error al borrar la imagen del banner en Cloudinary", "error", err, "bannerID", id)
		}
	}
	return nil
}

func (a *bannersApp) ListActiveBanners(ctx context.Context) (dtos.BannersResponse, error) {
	banners, err := a.repo.ListActiveBanners(ctx)
	if err != nil {
		return nil, err
	}
	return banners.ToDTO(), nil
}
