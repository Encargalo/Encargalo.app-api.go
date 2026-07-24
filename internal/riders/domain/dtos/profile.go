package dtos

import "Encargalo.app-api.go/internal/riders/domain/models"

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
	CreatedAt          string             `json:"created_at,omitempty"`
	UpdatedAt          string             `json:"updated_at,omitempty"`
	Completion         RiderCompletionDTO `json:"completion"`
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

type TouchContactRequest struct {
	Notes *string `json:"notes"`
}

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
