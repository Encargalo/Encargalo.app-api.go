package app

import (
	"context"
	"errors"
	"time"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

func (a *riderProfileApp) GetOwnApplication(ctx context.Context, userID uuid.UUID) (dtos.RiderApplicationResponse, error) {
	if err := a.ensureNotRider(ctx, userID); err != nil {
		return dtos.RiderApplicationResponse{}, err
	}

	profile, err := a.ownApplication(ctx, userID)
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
