package ports

import (
	"context"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
)

type RiderProfileRepo interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.RiderProfile, error)
	Upsert(ctx context.Context, profile *models.RiderProfile) error
	// ClearPhoto pone NULL en la columna del campo foto (whitelist en models.RiderPhotoColumns).
	ClearPhoto(ctx context.Context, userID uuid.UUID, field string) error
	ListByStatus(ctx context.Context, status models.RiderProfileStatus) ([]models.RiderProfile, error)
	Activate(ctx context.Context, userID, staffID uuid.UUID, activatedAt time.Time) error
	Suspend(ctx context.Context, userID uuid.UUID) error
	TouchContact(ctx context.Context, userID uuid.UUID, contactedAt time.Time, notes *string) error
	// SaveApplication guarda la solicitud solo si no existe o está en un estado editable por el solicitante;
	// si no, devuelve ErrRiderApplicationLocked.
	SaveApplication(ctx context.Context, profile *models.RiderProfile) error
	// SubmitApplication pasa la solicitud a pending_review; ErrRiderApplicationLocked si ya no es editable.
	SubmitApplication(ctx context.Context, userID uuid.UUID, submittedAt time.Time) error
	HasRiderRole(ctx context.Context, userID uuid.UUID) (bool, error)
	EnsureRiderRole(ctx context.Context, userID uuid.UUID) error
	DisableRiderRole(ctx context.Context, userID uuid.UUID) error
	FindAccountByPhone(ctx context.Context, phone string) (userID uuid.UUID, name, surName, phoneOut, email string, err error)
	GetAccountBasics(ctx context.Context, userID uuid.UUID) (name, surName, phone, email string, err error)
}

type RiderProfileApp interface {
	FindCandidate(ctx context.Context, phone string) (dtos.CandidateResponse, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (dtos.RiderProfileResponse, error)
	// GetOwnProfile devuelve al propio conductor sus datos y su vehículo, sin campos internos.
	// Si aún no tiene perfil, devuelve solo los datos de la cuenta con el vehículo vacío.
	GetOwnProfile(ctx context.Context, userID uuid.UUID) (dtos.RiderSelfProfileResponse, error)
	ListProfiles(ctx context.Context, status models.RiderProfileStatus) ([]dtos.RiderProfileResponse, error)
	UpsertProfile(ctx context.Context, userID uuid.UUID, req dtos.UpsertRiderProfileRequest) (dtos.RiderProfileResponse, error)
	// DeletePhoto limpia el campo en DB y elimina el asset en Cloudinary.
	DeletePhoto(ctx context.Context, userID uuid.UUID, field string) (dtos.RiderProfileResponse, error)
	Activate(ctx context.Context, userID, staffID uuid.UUID) (dtos.RiderProfileResponse, error)
	Suspend(ctx context.Context, userID uuid.UUID) (dtos.RiderProfileResponse, error)
	TouchContact(ctx context.Context, userID uuid.UUID, notes *string) (dtos.RiderProfileResponse, error)
	// GetOwnApplication devuelve la solicitud del propio solicitante; si no existe, una vacía en draft.
	GetOwnApplication(ctx context.Context, userID uuid.UUID) (dtos.RiderApplicationResponse, error)
	// SaveOwnApplication guarda la solicitud sin enviarla. No asigna el rol Rider ni cambia el estado.
	SaveOwnApplication(ctx context.Context, userID uuid.UUID, req dtos.RiderApplicationRequest) (dtos.RiderApplicationResponse, error)
	// SubmitOwnApplication envía a revisión una solicitud completa. No asigna el rol Rider.
	// Con ErrRiderProfileIncomplete también devuelve la solicitud, para leer qué falta en Completion.
	SubmitOwnApplication(ctx context.Context, userID uuid.UUID) (dtos.RiderApplicationResponse, error)
	// IsActiveRider used by auth SignInRider gate.
	IsActiveRider(ctx context.Context, userID uuid.UUID) (bool, error)
}
