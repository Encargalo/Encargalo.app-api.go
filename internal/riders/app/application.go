package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

// GetOwnApplication, a diferencia de Save/Submit, no bloquea con ensureNotRider a cualquier
// cuenta con rol Rider: un solicitante aprobado también tiene ese rol, y necesita ver su propio
// status real (active) en vez de un 409 indistinguible del caso legacy. Solo se trata como
// "legacy sin expediente" cuando no hay fila en riders.profiles.
func (a *riderProfileApp) GetOwnApplication(ctx context.Context, userID uuid.UUID) (dtos.RiderApplicationResponse, error) {
	profile, err := a.repo.GetByUserID(ctx, userID)
	if errors.Is(err, errcustom.ErrNotFound) {
		if err := a.ensureNotRider(ctx, userID); err != nil {
			return dtos.RiderApplicationResponse{}, err
		}
		return toApplicationResponse(&models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}), nil
	}
	if err != nil {
		return dtos.RiderApplicationResponse{}, err
	}
	return toApplicationResponse(profile), nil
}

func (a *riderProfileApp) SaveOwnApplication(ctx context.Context, userID uuid.UUID, req dtos.RiderApplicationRequest) (dtos.RiderApplicationResponse, error) {
	if err := a.ensureNotRider(ctx, userID); err != nil {
		return dtos.RiderApplicationResponse{}, err
	}

	profile, err := a.ownApplication(ctx, userID)
	if err != nil {
		return dtos.RiderApplicationResponse{}, err
	}
	// Se valida antes de subir fotos para no dejar assets huérfanos en Cloudinary.
	if !profile.ApplicantCanEdit() {
		return dtos.RiderApplicationResponse{}, errcustom.ErrRiderApplicationLocked
	}

	photos, err := a.applyChanges(ctx, profile, req.ToUpsert())
	if err != nil {
		return dtos.RiderApplicationResponse{}, err
	}
	if err := a.repo.SaveApplication(ctx, profile); err != nil {
		a.deleteImages(ctx, userID, photos.uploaded)
		return dtos.RiderApplicationResponse{}, err
	}
	a.deleteImages(ctx, userID, photos.replaced)
	a.publishRiderUpdated(ctx, userID)
	return toApplicationResponse(profile), nil
}

func (a *riderProfileApp) SubmitOwnApplication(ctx context.Context, userID uuid.UUID) (dtos.RiderApplicationResponse, error) {
	if err := a.ensureNotRider(ctx, userID); err != nil {
		return dtos.RiderApplicationResponse{}, err
	}

	profile, err := a.ownApplication(ctx, userID)
	if err != nil {
		return dtos.RiderApplicationResponse{}, err
	}
	if !profile.ApplicantCanEdit() {
		return dtos.RiderApplicationResponse{}, errcustom.ErrRiderApplicationLocked
	}
	if !profile.IsComplete() {
		return toApplicationResponse(profile), errcustom.ErrRiderProfileIncomplete
	}

	now := time.Now()
	if err := a.repo.SubmitApplication(ctx, userID, now); err != nil {
		return dtos.RiderApplicationResponse{}, err
	}
	profile.Status = models.RiderProfilePendingReview
	profile.SubmittedAt = &now
	a.publishRiderUpdated(ctx, userID)
	return toApplicationResponse(profile), nil
}

// ensureNotRider frena a los riders legacy sin expediente: crearles uno en draft los bloquearía en IsActiveRider.
func (a *riderProfileApp) ensureNotRider(ctx context.Context, userID uuid.UUID) error {
	hasRole, err := a.repo.HasRiderRole(ctx, userID)
	if err != nil {
		return err
	}
	if hasRole {
		return errcustom.ErrAlreadyRider
	}
	return nil
}

func (a *riderProfileApp) RequestApplicationChanges(ctx context.Context, userID, staffID uuid.UUID, reason string) (dtos.RiderProfileResponse, error) {
	return a.review(ctx, userID, staffID, models.RiderProfileChangesRequested, reason)
}

func (a *riderProfileApp) RejectApplication(ctx context.Context, userID, staffID uuid.UUID, reason string) (dtos.RiderProfileResponse, error) {
	return a.review(ctx, userID, staffID, models.RiderProfileRejected, reason)
}

// review deja por escrito la decisión del staff sobre una solicitud enviada. El motivo lo lee el
// conductor en su app, así que se exige en los dos casos: con changes_requested es lo único que le
// dice qué corregir antes de reenviar.
func (a *riderProfileApp) review(
	ctx context.Context,
	userID, staffID uuid.UUID,
	status models.RiderProfileStatus,
	reason string,
) (dtos.RiderProfileResponse, error) {
	if !models.IsReviewDecision(status) {
		return dtos.RiderProfileResponse{}, errcustom.ErrRiderReviewNotPending
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return dtos.RiderProfileResponse{}, errcustom.ErrReviewReasonRequired
	}

	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	// Se comprueba aquí para distinguir "no existe" de "ya no está en revisión"; el repo lo
	// vuelve a condicionar en el UPDATE por si alguien decidió en paralelo.
	if !profile.AwaitsReview() {
		return dtos.RiderProfileResponse{}, errcustom.ErrRiderReviewNotPending
	}

	if err := a.repo.ReviewApplication(ctx, userID, staffID, status, reason, time.Now()); err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	a.publishRiderUpdated(ctx, userID)
	return a.GetProfile(ctx, userID)
}

func (a *riderProfileApp) ownApplication(ctx context.Context, userID uuid.UUID) (*models.RiderProfile, error) {
	profile, err := a.repo.GetByUserID(ctx, userID)
	if errors.Is(err, errcustom.ErrNotFound) {
		return &models.RiderProfile{UserID: userID, Status: models.RiderProfileDraft}, nil
	}
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func toApplicationResponse(p *models.RiderProfile) dtos.RiderApplicationResponse {
	out := dtos.RiderApplicationResponse{
		Status:             string(p.Status),
		CanEdit:            p.ApplicantCanEdit(),
		ProfilePhotoURL:    p.ProfilePhotoURL,
		VehicleType:        p.VehicleType,
		VehiclePlate:       p.VehiclePlate,
		VehicleBrand:       p.VehicleBrand,
		VehicleModel:       p.VehicleModel,
		VehicleColor:       p.VehicleColor,
		IDPhotoURL:         p.IDPhotoURL,
		CarnetPhotoURL:     p.CarnetPhotoURL,
		LicensePhotoURL:    p.LicensePhotoURL,
		CirculationCardURL: p.CirculationCardURL,
		VehiclePhoto1URL:   p.VehiclePhoto1URL,
		VehiclePhoto2URL:   p.VehiclePhoto2URL,
		VehiclePhoto3URL:   p.VehiclePhoto3URL,
		VehiclePhoto4URL:   p.VehiclePhoto4URL,
		ReviewReason:       p.ReviewReason,
		Completion:         dtos.CompletionFromModel(p.Completion()),
	}
	if p.SubmittedAt != nil {
		s := p.SubmittedAt.Format(time.RFC3339)
		out.SubmittedAt = &s
	}
	return out
}
