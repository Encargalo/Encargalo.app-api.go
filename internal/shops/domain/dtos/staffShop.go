package dtos

import (
	"context"

	jsonutil "Encargalo.app-api.go/internal/pkg/json"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var staffShopValidate = validator.New()

type ShopCompletionDTO struct {
	IsComplete    bool     `json:"is_complete"`
	Missing       []string `json:"missing"`
	FilledCount   int      `json:"filled_count"`
	RequiredCount int      `json:"required_count"`
}

// OwnerCandidateResponse es la cuenta encontrada por teléfono al buscar a
// quién vincular como owner de un shop nuevo.
type OwnerCandidateResponse struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	SurName      string `json:"sur_name,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Email        string `json:"email,omitempty"`
	HasOwnerRole bool   `json:"has_owner_role"`
}

// StaffShopResponse es la vista de un shop desde el panel de staff (incluye
// campos de onboarding que no son públicos, como status/completion).
type StaffShopResponse struct {
	ID          uuid.UUID         `json:"id"`
	OwnerID     uuid.UUID         `json:"owner_id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug,omitempty"`
	Address     string            `json:"address,omitempty"`
	HomePhone   string            `json:"home_phone,omitempty"`
	Latitude    *float64          `json:"latitude,omitempty"`
	Longitude   *float64          `json:"longitude,omitempty"`
	LogoImage   string            `json:"logo_image,omitempty"`
	Banner      string            `json:"banner,omitempty"`
	Type        string            `json:"type"`
	Specialty   string            `json:"specialty"`
	Status      string            `json:"status"`
	Opened      bool              `json:"opened"`
	ActivatedAt *string           `json:"activated_at,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
	Completion  ShopCompletionDTO `json:"completion"`
}

// CreateDraftShopRequest crea un shop en estado draft. El owner ya debe
// existir como cuenta (se resuelve por OwnerID, obtenido vía el endpoint de
// candidatos) y gana el rol Owner sobre este shop como parte de la creación.
type CreateDraftShopRequest struct {
	Name      string    `json:"name" validate:"required" example:"Pizzería Don Luigi"`
	Type      string    `json:"type" validate:"required,oneof=Restaurant Supermarket Turbo" example:"Restaurant"`
	Specialty string    `json:"specialty" validate:"omitempty,oneof=General Panadería Carnicería Charcutería Frutería Licorería Pizzería 'Comida Rápida' 'Comida China' Heladería Cafetería" example:"Pizzería"`
	OwnerID   uuid.UUID `json:"owner_id" validate:"required" example:"d33aaf08-2c43-41c4-b2e7-882b019edb1e"`
}

func (r *CreateDraftShopRequest) Validate() error {
	return staffShopValidate.StructCtx(context.Background(), r)
}

// UpdateDraftShopRequest es una actualización parcial: los campos omitidos
// no se tocan. Permite completar el shop por partes, en cualquier orden.
// Latitude/Longitude/LogoImage/Banner son Nullable: omitir la clave no
// toca el campo, pero enviarla explícitamente en `null` lo limpia — un
// *T normal no puede distinguir "omitido" de "null" (ambos son nil).
type UpdateDraftShopRequest struct {
	Name      *string                    `json:"name"`
	Slug      *string                    `json:"slug"`
	Address   *string                    `json:"address"`
	HomePhone *string                    `json:"home_phone"`
	Specialty *string                    `json:"specialty" validate:"omitempty,oneof=General Panadería Carnicería Charcutería Frutería Licorería Pizzería 'Comida Rápida' 'Comida China' Heladería Cafetería" example:"Panadería"`
	Latitude  jsonutil.Nullable[float64] `json:"latitude" swaggertype:"number"`
	Longitude jsonutil.Nullable[float64] `json:"longitude" swaggertype:"number"`
	LogoImage jsonutil.Nullable[string]  `json:"logo_image" swaggertype:"string" example:"data:image/jpeg;base64,/9j/4AAQSkZJRg..."`
	Banner    jsonutil.Nullable[string]  `json:"banner" swaggertype:"string" example:"data:image/jpeg;base64,/9j/4AAQSkZJRg..."`
}

func (r *UpdateDraftShopRequest) Validate() error {
	return staffShopValidate.StructCtx(context.Background(), r)
}

// StaffShopsResponse es la respuesta paginada de GET /shops/staff.
type StaffShopsResponse struct {
	Shops []StaffShopResponse `json:"shops"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// ShopActivateErrorResponse se devuelve cuando falta completar el checklist
// mínimo para activar (mismo shape que riders.ActivateErrorResponse).
type ShopActivateErrorResponse struct {
	Message string   `json:"message"`
	Missing []string `json:"missing"`
}
