package models

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type RiderProfileStatus string

const (
	RiderProfileDraft            RiderProfileStatus = "draft"
	RiderProfilePendingReview    RiderProfileStatus = "pending_review"
	RiderProfileChangesRequested RiderProfileStatus = "changes_requested"
	RiderProfileRejected         RiderProfileStatus = "rejected"
	RiderProfileActive           RiderProfileStatus = "active"
	RiderProfileSuspended        RiderProfileStatus = "suspended"

	VehicleTypeMoto = "moto"
	VehicleTypeBike = "bici"
	VehicleTypeCar  = "carro"
)

type RiderProfile struct {
	bun.BaseModel `bun:"table:riders.profiles"`

	UserID             uuid.UUID          `bun:"user_id,pk,type:uuid"`
	Status             RiderProfileStatus `bun:"status,notnull"`
	ProfilePhotoURL    *string            `bun:"profile_photo_url,nullzero"`
	VehicleType        *string            `bun:"vehicle_type,nullzero"`
	VehiclePlate       *string            `bun:"vehicle_plate,nullzero"`
	VehicleBrand       *string            `bun:"vehicle_brand,nullzero"`
	VehicleModel       *string            `bun:"vehicle_model,nullzero"`
	VehicleColor       *string            `bun:"vehicle_color,nullzero"`
	IDPhotoURL         *string            `bun:"id_photo_url,nullzero"`
	CarnetPhotoURL     *string            `bun:"carnet_photo_url,nullzero"`
	LicensePhotoURL    *string            `bun:"license_photo_url,nullzero"`
	CirculationCardURL *string            `bun:"circulation_card_url,nullzero"`
	VehiclePhoto1URL   *string            `bun:"vehicle_photo_1_url,nullzero"`
	VehiclePhoto2URL   *string            `bun:"vehicle_photo_2_url,nullzero"`
	VehiclePhoto3URL   *string            `bun:"vehicle_photo_3_url,nullzero"`
	VehiclePhoto4URL   *string            `bun:"vehicle_photo_4_url,nullzero"`
	Notes              *string            `bun:"notes,nullzero"`
	LastContactedAt    *time.Time         `bun:"last_contacted_at,nullzero"`
	ActivatedAt        *time.Time         `bun:"activated_at,nullzero"`
	ActivatedBy        *uuid.UUID         `bun:"activated_by,nullzero,type:uuid"`
	SubmittedAt        *time.Time         `bun:"submitted_at,nullzero"`
	ReviewedAt         *time.Time         `bun:"reviewed_at,nullzero"`
	ReviewedBy         *uuid.UUID         `bun:"reviewed_by,nullzero,type:uuid"`
	ReviewReason       *string            `bun:"review_reason,nullzero"`
	CreatedAt          time.Time          `bun:"created_at,nullzero"`
	UpdatedAt          time.Time          `bun:"updated_at,nullzero"`
}

// ApplicantEditableStatuses son los estados en que el propio solicitante puede editar o enviar su solicitud.
var ApplicantEditableStatuses = []RiderProfileStatus{RiderProfileDraft, RiderProfileChangesRequested}

func ValidRiderProfileStatus(status RiderProfileStatus) bool {
	switch status {
	case RiderProfileDraft, RiderProfilePendingReview, RiderProfileChangesRequested,
		RiderProfileRejected, RiderProfileActive, RiderProfileSuspended:
		return true
	}
	return false
}

func (p *RiderProfile) ApplicantCanEdit() bool {
	return slices.Contains(ApplicantEditableStatuses, p.Status)
}

type RiderCompletion struct {
	IsComplete    bool     `json:"is_complete"`
	Missing       []string `json:"missing"`
	FilledCount   int      `json:"filled_count"`
	RequiredCount int      `json:"required_count"`
}

func strSet(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}

// Completion calcula el checklist mínimo para activar.
func (p *RiderProfile) Completion() RiderCompletion {
	checks := []struct {
		key string
		ok  bool
	}{
		{"profile_photo", strSet(p.ProfilePhotoURL)},
		{"id_photo", strSet(p.IDPhotoURL)},
		{"carnet_photo", strSet(p.CarnetPhotoURL)},
		{"license_photo", strSet(p.LicensePhotoURL)},
		{"circulation_card", strSet(p.CirculationCardURL)},
		{"vehicle_photo_1", strSet(p.VehiclePhoto1URL)},
		{"vehicle_photo_2", strSet(p.VehiclePhoto2URL)},
		{"vehicle_photo_3", strSet(p.VehiclePhoto3URL)},
		{"vehicle_photo_4", strSet(p.VehiclePhoto4URL)},
		{"vehicle_type", strSet(p.VehicleType)},
	}

	missing := make([]string, 0)
	filled := 0
	for _, c := range checks {
		if c.ok {
			filled++
		} else {
			missing = append(missing, c.key)
		}
	}

	// Placa requerida salvo bici.
	requiresPlate := !strSet(p.VehicleType) || !strings.EqualFold(strings.TrimSpace(*p.VehicleType), VehicleTypeBike)
	if requiresPlate {
		if strSet(p.VehiclePlate) {
			filled++
		} else {
			missing = append(missing, "vehicle_plate")
		}
	}

	required := len(checks)
	if requiresPlate {
		required++
	}

	return RiderCompletion{
		IsComplete:    len(missing) == 0,
		Missing:       missing,
		FilledCount:   filled,
		RequiredCount: required,
	}
}

func (p *RiderProfile) IsComplete() bool {
	return p.Completion().IsComplete
}

// Columnas SQL whitelisted para ClearPhoto (evitar inyección vía field).
var RiderPhotoColumns = map[string]string{
	"profile_photo":    "profile_photo_url",
	"id_photo":         "id_photo_url",
	"carnet_photo":     "carnet_photo_url",
	"license_photo":    "license_photo_url",
	"circulation_card": "circulation_card_url",
	"vehicle_photo_1":  "vehicle_photo_1_url",
	"vehicle_photo_2":  "vehicle_photo_2_url",
	"vehicle_photo_3":  "vehicle_photo_3_url",
	"vehicle_photo_4":  "vehicle_photo_4_url",
}

func ValidRiderPhotoField(field string) bool {
	_, ok := RiderPhotoColumns[field]
	return ok
}

// PhotoURLForField devuelve la URL actual del campo foto, o nil.
func (p *RiderProfile) PhotoURLForField(field string) *string {
	if p == nil {
		return nil
	}
	switch field {
	case "profile_photo":
		return p.ProfilePhotoURL
	case "id_photo":
		return p.IDPhotoURL
	case "carnet_photo":
		return p.CarnetPhotoURL
	case "license_photo":
		return p.LicensePhotoURL
	case "circulation_card":
		return p.CirculationCardURL
	case "vehicle_photo_1":
		return p.VehiclePhoto1URL
	case "vehicle_photo_2":
		return p.VehiclePhoto2URL
	case "vehicle_photo_3":
		return p.VehiclePhoto3URL
	case "vehicle_photo_4":
		return p.VehiclePhoto4URL
	default:
		return nil
	}
}
