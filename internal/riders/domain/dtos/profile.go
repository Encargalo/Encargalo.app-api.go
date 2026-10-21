package dtos

import (
	"context"

	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/go-playground/validator/v10"
)

var applicationValidate = validator.New()

type RiderCompletionDTO struct {
	IsComplete    bool     `json:"is_complete"`
	Missing       []string `json:"missing"`
	FilledCount   int      `json:"filled_count"`
	RequiredCount int      `json:"required_count"`
}

type RiderProfileResponse struct {
	UserID             string             `json:"user_id"`
	Status             string             `json:"status"`
	Name               string             `json:"name,omitempty"`
	SurName            string             `json:"sur_name,omitempty"`
	Phone              string             `json:"phone,omitempty"`
	Email              string             `json:"email,omitempty"`
	HasRiderRole       bool               `json:"has_rider_role"`
	ProfilePhotoURL    *string            `json:"profile_photo_url,omitempty"`
	VehicleType        *string            `json:"vehicle_type,omitempty"`
	VehiclePlate       *string            `json:"vehicle_plate,omitempty"`
	VehicleBrand       *string            `json:"vehicle_brand,omitempty"`
	VehicleModel       *string            `json:"vehicle_model,omitempty"`
	VehicleColor       *string            `json:"vehicle_color,omitempty"`
	IDPhotoURL         *string            `json:"id_photo_url,omitempty"`
	CarnetPhotoURL     *string            `json:"carnet_photo_url,omitempty"`
	LicensePhotoURL    *string            `json:"license_photo_url,omitempty"`
	CirculationCardURL *string            `json:"circulation_card_url,omitempty"`
	VehiclePhoto1URL   *string            `json:"vehicle_photo_1_url,omitempty"`
	VehiclePhoto2URL   *string            `json:"vehicle_photo_2_url,omitempty"`
	VehiclePhoto3URL   *string            `json:"vehicle_photo_3_url,omitempty"`
	VehiclePhoto4URL   *string            `json:"vehicle_photo_4_url,omitempty"`
	Notes              *string            `json:"notes,omitempty"`
	LastContactedAt    *string            `json:"last_contacted_at,omitempty"`
	ActivatedAt        *string            `json:"activated_at,omitempty"`
	SubmittedAt        *string            `json:"submitted_at,omitempty"`
	ReviewedAt         *string            `json:"reviewed_at,omitempty"`
	ReviewReason       *string            `json:"review_reason,omitempty"`
	CreatedAt          string             `json:"created_at,omitempty"`
	UpdatedAt          string             `json:"updated_at,omitempty"`
	Completion         RiderCompletionDTO `json:"completion"`
}

// RiderSelfDriver son los datos del conductor que él mismo puede ver en su panel.
type RiderSelfDriver struct {
	UserID          string  `json:"user_id"`
	Name            string  `json:"name"`
	SurName         string  `json:"sur_name,omitempty"`
	ProfilePhotoURL *string `json:"profile_photo_url,omitempty"`
}

// RiderSelfVehicle es el vehículo del conductor. Por ahora hay uno solo por rider;
// si aún no tiene perfil se serializa como objeto vacío.
type RiderSelfVehicle struct {
	Type  *string `json:"type,omitempty"`
	Plate *string `json:"plate,omitempty"`
	Brand *string `json:"brand,omitempty"`
	Model *string `json:"model,omitempty"`
	Color *string `json:"color,omitempty"`
}

// RiderSelfProfileResponse es lo mínimo que necesita el panel del conductor activo.
// No incluye contacto, notas ni fotos de documentos.
type RiderSelfProfileResponse struct {
	Driver  RiderSelfDriver  `json:"driver"`
	Vehicle RiderSelfVehicle `json:"vehicle"`
}

type CandidateResponse struct {
	UserID       string                `json:"user_id"`
	Name         string                `json:"name"`
	SurName      string                `json:"sur_name,omitempty"`
	Phone        string                `json:"phone,omitempty"`
	Email        string                `json:"email,omitempty"`
	HasRiderRole bool                  `json:"has_rider_role"`
	Profile      *RiderProfileResponse `json:"profile,omitempty"`
}

// UpsertRiderProfileRequest: campos omitidos no se tocan.
// Las fotos vienen como data-URL/base64; el API las sube a Cloudinary.
type UpsertRiderProfileRequest struct {
	VehicleType     *string `json:"vehicle_type"`
	VehiclePlate    *string `json:"vehicle_plate"`
	VehicleBrand    *string `json:"vehicle_brand"`
	VehicleModel    *string `json:"vehicle_model"`
	VehicleColor    *string `json:"vehicle_color"`
	Notes           *string `json:"notes"`
	ProfilePhoto    *string `json:"profile_photo"`
	IDPhoto         *string `json:"id_photo"`
	CarnetPhoto     *string `json:"carnet_photo"`
	LicensePhoto    *string `json:"license_photo"`
	CirculationCard *string `json:"circulation_card"`
	VehiclePhoto1   *string `json:"vehicle_photo_1"`
	VehiclePhoto2   *string `json:"vehicle_photo_2"`
	VehiclePhoto3   *string `json:"vehicle_photo_3"`
	VehiclePhoto4   *string `json:"vehicle_photo_4"`
}

// RiderApplicationRequest es lo que el propio solicitante puede guardar: campos omitidos no se tocan.
// Las fotos vienen como data-URL/base64; el API las sube a Cloudinary.
type RiderApplicationRequest struct {
	VehicleType     *string `json:"vehicle_type" validate:"omitempty,oneof=moto bici carro" example:"moto" enums:"moto,bici,carro"`
	VehiclePlate    *string `json:"vehicle_plate" validate:"omitempty,max=15" example:"AB123CD"`
	VehicleBrand    *string `json:"vehicle_brand" validate:"omitempty,max=50" example:"Yamaha"`
	VehicleModel    *string `json:"vehicle_model" validate:"omitempty,max=50" example:"FZ"`
	VehicleColor    *string `json:"vehicle_color" validate:"omitempty,max=30" example:"Negro"`
	ProfilePhoto    *string `json:"profile_photo" validate:"omitempty,max=7000000"`
	IDPhoto         *string `json:"id_photo" validate:"omitempty,max=7000000"`
	CarnetPhoto     *string `json:"carnet_photo" validate:"omitempty,max=7000000"`
	LicensePhoto    *string `json:"license_photo" validate:"omitempty,max=7000000"`
	CirculationCard *string `json:"circulation_card" validate:"omitempty,max=7000000"`
	VehiclePhoto1   *string `json:"vehicle_photo_1" validate:"omitempty,max=7000000"`
	VehiclePhoto2   *string `json:"vehicle_photo_2" validate:"omitempty,max=7000000"`
	VehiclePhoto3   *string `json:"vehicle_photo_3" validate:"omitempty,max=7000000"`
	VehiclePhoto4   *string `json:"vehicle_photo_4" validate:"omitempty,max=7000000"`
}

func (r *RiderApplicationRequest) Validate() error {
	return applicationValidate.StructCtx(context.Background(), r)
}

func (r RiderApplicationRequest) ToUpsert() UpsertRiderProfileRequest {
	return UpsertRiderProfileRequest{
		VehicleType:     r.VehicleType,
		VehiclePlate:    r.VehiclePlate,
		VehicleBrand:    r.VehicleBrand,
		VehicleModel:    r.VehicleModel,
		VehicleColor:    r.VehicleColor,
		ProfilePhoto:    r.ProfilePhoto,
		IDPhoto:         r.IDPhoto,
		CarnetPhoto:     r.CarnetPhoto,
		LicensePhoto:    r.LicensePhoto,
		CirculationCard: r.CirculationCard,
		VehiclePhoto1:   r.VehiclePhoto1,
		VehiclePhoto2:   r.VehiclePhoto2,
		VehiclePhoto3:   r.VehiclePhoto3,
		VehiclePhoto4:   r.VehiclePhoto4,
	}
}

// RiderApplicationResponse es la solicitud vista por el propio solicitante: sin notas internas ni datos de contacto.
type RiderApplicationResponse struct {
	Status             string             `json:"status" example:"draft" enums:"draft,pending_review,changes_requested,rejected,active,suspended"`
	CanEdit            bool               `json:"can_edit"`
	ProfilePhotoURL    *string            `json:"profile_photo_url,omitempty"`
	VehicleType        *string            `json:"vehicle_type,omitempty"`
	VehiclePlate       *string            `json:"vehicle_plate,omitempty"`
	VehicleBrand       *string            `json:"vehicle_brand,omitempty"`
	VehicleModel       *string            `json:"vehicle_model,omitempty"`
	VehicleColor       *string            `json:"vehicle_color,omitempty"`
	IDPhotoURL         *string            `json:"id_photo_url,omitempty"`
	CarnetPhotoURL     *string            `json:"carnet_photo_url,omitempty"`
	LicensePhotoURL    *string            `json:"license_photo_url,omitempty"`
	CirculationCardURL *string            `json:"circulation_card_url,omitempty"`
	VehiclePhoto1URL   *string            `json:"vehicle_photo_1_url,omitempty"`
	VehiclePhoto2URL   *string            `json:"vehicle_photo_2_url,omitempty"`
	VehiclePhoto3URL   *string            `json:"vehicle_photo_3_url,omitempty"`
	VehiclePhoto4URL   *string            `json:"vehicle_photo_4_url,omitempty"`
	SubmittedAt        *string            `json:"submitted_at,omitempty"`
	ReviewReason       *string            `json:"review_reason,omitempty"`
	Completion         RiderCompletionDTO `json:"completion"`
}

type TouchContactRequest struct {
	Notes *string `json:"notes"`
}

// ActivateErrorResponse también se usa al enviar una solicitud incompleta.
type ActivateErrorResponse struct {
	Message string   `json:"message"`
	Missing []string `json:"missing"`
}

func CompletionFromModel(c models.RiderCompletion) RiderCompletionDTO {
	missing := c.Missing
	if missing == nil {
		missing = []string{}
	}
	return RiderCompletionDTO{
		IsComplete:    c.IsComplete,
		Missing:       missing,
		FilledCount:   c.FilledCount,
		RequiredCount: c.RequiredCount,
	}
}
