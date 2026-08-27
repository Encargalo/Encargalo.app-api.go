package uploadimage

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type UploadImage interface {
	UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error)
	// DeleteImage elimina un asset por SecureURL (best-effort parse de public_id).
	DeleteImage(ctx context.Context, secureURL string) error
}

type uploadImage struct {
	cld *cloudinary.Cloudinary
}

func NewUploadImage(cld *cloudinary.Cloudinary) UploadImage {
	return &uploadImage{cld}
}

func (repo uploadImage) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	ctx, span := otel.Tracer(reqctx.ServiceName).Start(ctx, "cloudinary.Upload")
	span.SetAttributes(attribute.Int("image_bytes", len(imageB64)))
	defer span.End()

	folder := config.Get().Cloudinary.Folder
	resp, err := repo.cld.Upload.Upload(ctx, imageB64, uploader.UploadParams{Folder: folder})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		slog.Error("error uploading image", "error", err)
		return nil, err
	}
	// Cloudinary a veces responde sin error de transporte pero con Error/SecureURL vacío
	// (p. ej. base64 mal formado). Tratarlo como fallo para no crear órdenes huérfanas.
	if resp == nil || resp.SecureURL == "" || resp.Error.Message != "" {
		msg := "upload sin SecureURL"
		if resp != nil && resp.Error.Message != "" {
			msg = resp.Error.Message
		}
		span.SetStatus(codes.Error, msg)
		slog.Error("error uploading image", "error", msg)
		return nil, fmt.Errorf("cloudinary upload failed: %s", msg)
	}
	return resp, nil
}

// ValidateCloudinaryURL verifica que la URL sea HTTPS de Cloudinary.
func ValidateCloudinaryURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path == "" {
		return errcustom.ErrInvalidProofURL
	}
	host := strings.ToLower(u.Hostname())
	if host == "res.cloudinary.com" || strings.HasSuffix(host, ".cloudinary.com") {
		return nil
	}
	return errcustom.ErrInvalidProofURL
}

func (repo uploadImage) DeleteImage(ctx context.Context, secureURL string) error {
	secureURL = strings.TrimSpace(secureURL)
	if secureURL == "" {
		return nil
	}
	publicID, ok := PublicIDFromSecureURL(secureURL)
	if !ok {
		slog.Warn("cloudinary delete: no se pudo parsear public_id", "url", secureURL)
		return fmt.Errorf("public_id inválido desde url")
	}
	resp, err := repo.cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID})
	if err != nil {
		slog.Error("error deleting image", "publicID", publicID, "error", err)
		return err
	}
	if resp != nil && resp.Error.Message != "" {
		slog.Error("error deleting image", "publicID", publicID, "error", resp.Error.Message)
		return fmt.Errorf("cloudinary destroy failed: %s", resp.Error.Message)
	}
	return nil
}

// PublicIDFromSecureURL extrae el public_id de una SecureURL de Cloudinary.
// Ej: .../image/upload/v1234/folder/abc.jpg → folder/abc
func PublicIDFromSecureURL(secureURL string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(secureURL))
	if err != nil || u.Path == "" {
		return "", false
	}
	const marker = "/upload/"
	idx := strings.Index(u.Path, marker)
	if idx < 0 {
		return "", false
	}
	rest := u.Path[idx+len(marker):]
	parts := strings.Split(rest, "/")
	// Saltar transformaciones (contienen ',') y versión vN.
	i := 0
	for i < len(parts) {
		p := parts[i]
		if p == "" {
			i++
			continue
		}
		if strings.Contains(p, ",") {
			i++
			continue
		}
		if len(p) > 1 && p[0] == 'v' && isAllDigits(p[1:]) {
			i++
			break
		}
		break
	}
	if i >= len(parts) {
		return "", false
	}
	idParts := parts[i:]
	joined := strings.Join(idParts, "/")
	joined = strings.TrimSuffix(joined, path.Ext(joined))
	joined = strings.Trim(joined, "/")
	if joined == "" {
		return "", false
	}
	return joined, true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
