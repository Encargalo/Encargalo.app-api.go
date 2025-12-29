package uploadimage

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type UploadImage interface {
	UploadImage(ctx context.Context, imageB64 string) (string, error)
}

type uploadImage struct {
	cld *cloudinary.Cloudinary
}

func NewUploadImage(cld *cloudinary.Cloudinary) UploadImage {
	return &uploadImage{cld}
}

func (repo uploadImage) UploadImage(ctx context.Context, imageB64 string) (string, error) {
	folder := config.Get().Cloudinary.Folder
	resp, err := repo.cld.Upload.Upload(ctx, imageB64, uploader.UploadParams{Folder: folder})
	if err != nil {
		slog.Error("error uploading image", "error", err)
		return "", err
	}
	return resp.URL, nil
}
