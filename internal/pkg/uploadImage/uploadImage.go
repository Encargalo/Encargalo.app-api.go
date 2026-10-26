package uploadimage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
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

// allowedImageMIMEs son los únicos media types que se aceptan en el data-URL.
var allowedImageMIMEs = map[string]struct{}{
	"image/jpeg": {},
	"image/jpg":  {},
	"image/png":  {},
	"image/webp": {},
	"image/heic": {},
	"image/heif": {},
}

// allowedUploadFormats repite la lista del lado de Cloudinary: si el payload no es una imagen
// real de estos formatos, el upload se rechaza allá aunque el data-URL dijera lo contrario.
var allowedUploadFormats = api.CldAPIArray{"jpg", "jpeg", "png", "webp", "heic", "heif"}

// incomingTransformation acota la foto de teléfono antes de guardarla; q_auto:good no degrada
// la referencia bancaria de un comprobante, que es evidencia que el staff tiene que leer.
// La conversión a WebP va encadenada aquí (/f_webp) y no en UploadParams.Format: Cloudinary
// ignora Format en silencio cuando también se manda AllowedFormats (comprobado empíricamente,
// no está documentado), así que subía el formato original pese a este campo.
const incomingTransformation = "c_limit,w_1600,q_auto:good/f_webp"

// ValidateImageDataURL exige un data-URL base64 de imagen. Es obligatorio antes de llamar a
// Cloudinary: el SDK decide por el formato del string, así que una URL la descargaría Cloudinary
// y cualquier otro string lo trataría como una ruta del filesystem del servidor.
func ValidateImageDataURL(raw string) error {
	s := strings.TrimSpace(raw)

	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return errcustom.ErrUploadingImage
	}

	meta := strings.ToLower(s[:comma])
	if !strings.HasPrefix(meta, "data:") || !strings.HasSuffix(meta, ";base64") {
		return errcustom.ErrUploadingImage
	}

	mime := strings.TrimSpace(strings.Split(strings.TrimSuffix(meta[len("data:"):], ";base64"), ";")[0])
	if _, ok := allowedImageMIMEs[mime]; !ok {
		return errcustom.ErrUploadingImage
	}

	return validateBase64Payload(s[comma+1:])
}

// validateBase64Payload decodifica en streaming para no duplicar en memoria un comprobante
// que puede pesar megas; los decoders ignoran los saltos de línea que meten algunos clientes.
func validateBase64Payload(payload string) error {
	if strings.TrimSpace(payload) == "" {
		return errcustom.ErrUploadingImage
	}

	if _, err := io.Copy(io.Discard, base64.NewDecoder(base64.StdEncoding, strings.NewReader(payload))); err == nil {
		return nil
	}

	// Algunos clientes mandan el base64 sin padding.
	if _, err := io.Copy(io.Discard, base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(payload))); err == nil {
		return nil
	}

	return errcustom.ErrUploadingImage
}

func (repo uploadImage) UploadImage(ctx context.Context, imageB64 string) (*uploader.UploadResult, error) {
	ctx, span := otel.Tracer(reqctx.ServiceName).Start(ctx, "cloudinary.Upload")
	span.SetAttributes(attribute.Int("image_bytes", len(imageB64)))
	defer span.End()

	if err := ValidateImageDataURL(imageB64); err != nil {
		span.SetStatus(codes.Error, "formato de imagen inválido")
		slog.Error("imagen rechazada antes de subir a Cloudinary")
		return nil, err
	}

	folder := config.Get().Cloudinary.Folder
	resp, err := repo.cld.Upload.Upload(ctx, imageB64, uploader.UploadParams{
		Folder:         folder,
		ResourceType:   "image",
		AllowedFormats: allowedUploadFormats,
		Transformation: incomingTransformation,
	})
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
