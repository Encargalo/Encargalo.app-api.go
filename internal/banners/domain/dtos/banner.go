package dtos

import (
	jsonutil "Encargalo.app-api.go/internal/pkg/json"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var bannerValidate = validator.New()

// BannerResponse se reutiliza tanto para el panel de staff (lista/detalle
// completos) como para el endpoint público: éste último ya llega filtrado y
// ordenado desde el repo, así que exponer position/active de más no es un
// problema y evita una segunda DTO idéntica.
type BannerResponse struct {
	ID       uuid.UUID `json:"id"`
	ImageURL string    `json:"image_url"`
	LinkURL  *string   `json:"link_url,omitempty"`
	Position int       `json:"position"`
	Active   bool      `json:"active"`
}

type BannersResponse []BannerResponse

// CreateBannerRequest crea un banner. Image es un data-URL base64 (igual que
// logo_image/banner en shops): se sube a Cloudinary en el app layer, nunca se
// persiste tal cual.
type CreateBannerRequest struct {
	Image    string  `json:"image" validate:"required"`
	LinkURL  *string `json:"link_url,omitempty" validate:"omitempty,url"`
	Position *int    `json:"position,omitempty"`
}

func (r *CreateBannerRequest) Validate() error {
	return bannerValidate.Struct(r)
}

// UpdateBannerRequest es una actualización parcial: los campos omitidos no se
// tocan. Image es un *string simple (no Nullable) porque un banner siempre
// necesita una imagen — "limpiarla" no tiene un estado válido, a diferencia
// de LinkURL, donde `null` sí remueve el link del todo.
type UpdateBannerRequest struct {
	Image    *string                   `json:"image,omitempty" example:"data:image/jpeg;base64,/9j/4AAQSkZJRg..."`
	LinkURL  jsonutil.Nullable[string] `json:"link_url" swaggertype:"string"`
	Position *int                      `json:"position,omitempty"`
	Active   *bool                     `json:"active,omitempty"`
}
