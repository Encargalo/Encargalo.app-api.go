package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type riderProfileApp struct {
	repo     ports.RiderProfileRepo
	uploader uploadimage.UploadImage
}

func NewRiderProfileApp(repo ports.RiderProfileRepo, uploader uploadimage.UploadImage) ports.RiderProfileApp {
	return &riderProfileApp{repo: repo, uploader: uploader}
}

func (a *riderProfileApp) FindCandidate(ctx context.Context, phone string) (dtos.CandidateResponse, error) {
	userID, name, sur, ph, email, err := a.repo.FindAccountByPhone(ctx, phone)
	if err != nil {
		return dtos.CandidateResponse{}, err
	}
	hasRole, err := a.repo.HasRiderRole(ctx, userID)
	if err != nil {
		return dtos.CandidateResponse{}, err
	}
	out := dtos.CandidateResponse{
		UserID:       userID.String(),
		Name:         name,
		SurName:      sur,
		Phone:        ph,
		Email:        email,
		HasRiderRole: hasRole,
	}
	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return out, nil
		}
		return dtos.CandidateResponse{}, err
	}
	resp := a.toResponse(profile, name, sur, ph, email, hasRole)
	out.Profile = &resp
	return out, nil
}

func (a *riderProfileApp) GetProfile(ctx context.Context, userID uuid.UUID) (dtos.RiderProfileResponse, error) {
	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	name, sur, ph, email, err := a.repo.GetAccountBasics(ctx, userID)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	hasRole, err := a.repo.HasRiderRole(ctx, userID)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	return a.toResponse(profile, name, sur, ph, email, hasRole), nil
}

func (a *riderProfileApp) GetOwnProfile(ctx context.Context, userID uuid.UUID) (dtos.RiderSelfProfileResponse, error) {
	name, sur, _, _, err := a.repo.GetAccountBasics(ctx, userID)
	if err != nil {
		return dtos.RiderSelfProfileResponse{}, err
	}
	out := dtos.RiderSelfProfileResponse{
		Driver: dtos.RiderSelfDriver{UserID: userID.String(), Name: name, SurName: sur},
	}

	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return out, nil
		}
		return dtos.RiderSelfProfileResponse{}, err
	}
	out.Driver.ProfilePhotoURL = profile.ProfilePhotoURL
	out.Vehicle = dtos.RiderSelfVehicle{
		Type:  profile.VehicleType,
		Plate: profile.VehiclePlate,
		Brand: profile.VehicleBrand,
		Model: profile.VehicleModel,
		Color: profile.VehicleColor,
	}
	return out, nil
}

func (a *riderProfileApp) ListProfiles(ctx context.Context, status models.RiderProfileStatus) ([]dtos.RiderProfileResponse, error) {
	if status == "" {
		status = models.RiderProfilePendingReview
	}
	list, err := a.repo.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]dtos.RiderProfileResponse, 0, len(list))
	for i := range list {
		p := list[i]
		name, sur, ph, email, err := a.repo.GetAccountBasics(ctx, p.UserID)
		if err != nil {
			continue
		}
		hasRole, _ := a.repo.HasRiderRole(ctx, p.UserID)
		out = append(out, a.toResponse(&p, name, sur, ph, email, hasRole))
	}
	return out, nil
}

func (a *riderProfileApp) UpsertProfile(ctx context.Context, userID uuid.UUID, req dtos.UpsertRiderProfileRequest) (dtos.RiderProfileResponse, error) {
	if _, _, _, _, err := a.repo.GetAccountBasics(ctx, userID); err != nil {
		return dtos.RiderProfileResponse{}, err
	}

	existing, err := a.repo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, errcustom.ErrNotFound) {
		return dtos.RiderProfileResponse{}, err
	}

	profile := &models.RiderProfile{
		UserID: userID,
		Status: models.RiderProfileDraft,
	}
	if existing != nil {
		profile = existing
		// No bajar de active/suspended a draft por un upsert parcial de fotos.
		if profile.Status == "" {
			profile.Status = models.RiderProfileDraft
		}
	}

	photos, err := a.applyChanges(ctx, profile, req)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}

	// Para upsert parcial vía COALESCE en SQL: enviamos solo campos tocados.
	// Si el perfil ya existía, reescribimos todo el struct (valores actuales + cambios).
	if err := a.repo.Upsert(ctx, profile); err != nil {
		a.deleteImages(ctx, profile.UserID, photos.uploaded)
		return dtos.RiderProfileResponse{}, err
	}
	a.deleteImages(ctx, profile.UserID, photos.replaced)
	return a.GetProfile(ctx, userID)
}

// photoChanges son las URLs que applyChanges subió y las que quedan reemplazadas al guardar.
type photoChanges struct {
	uploaded []string
	replaced []string
}

// applyChanges copia al perfil los campos enviados y sube a Cloudinary las fotos nuevas.
func (a *riderProfileApp) applyChanges(ctx context.Context, profile *models.RiderProfile, req dtos.UpsertRiderProfileRequest) (photoChanges, error) {
	if req.VehicleType != nil {
		v := strings.TrimSpace(*req.VehicleType)
		profile.VehicleType = &v
	}
	if req.VehiclePlate != nil {
		v := strings.TrimSpace(*req.VehiclePlate)
		profile.VehiclePlate = &v
	}
	if req.VehicleBrand != nil {
		v := strings.TrimSpace(*req.VehicleBrand)
		profile.VehicleBrand = &v
	}
	if req.VehicleModel != nil {
		v := strings.TrimSpace(*req.VehicleModel)
		profile.VehicleModel = &v
	}
	if req.VehicleColor != nil {
		v := strings.TrimSpace(*req.VehicleColor)
		profile.VehicleColor = &v
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		profile.Notes = &v
	}

	type uploadJob struct {
		raw string
		dst **string
	}
	jobs := make([]uploadJob, 0, 9)
	addJob := func(raw *string, dst **string) {
		if raw == nil || strings.TrimSpace(*raw) == "" {
			return
		}
		jobs = append(jobs, uploadJob{raw: *raw, dst: dst})
	}
	addJob(req.ProfilePhoto, &profile.ProfilePhotoURL)
	addJob(req.IDPhoto, &profile.IDPhotoURL)
	addJob(req.CarnetPhoto, &profile.CarnetPhotoURL)
	addJob(req.LicensePhoto, &profile.LicensePhotoURL)
	addJob(req.CirculationCard, &profile.CirculationCardURL)
	addJob(req.VehiclePhoto1, &profile.VehiclePhoto1URL)
	addJob(req.VehiclePhoto2, &profile.VehiclePhoto2URL)
	addJob(req.VehiclePhoto3, &profile.VehiclePhoto3URL)
	addJob(req.VehiclePhoto4, &profile.VehiclePhoto4URL)

	var changes photoChanges
	if len(jobs) == 0 {
		return changes, nil
	}

	urls := make([]string, len(jobs))
	g, gctx := errgroup.WithContext(ctx)
	for i := range jobs {
		g.Go(func() error {
			url, err := a.upload(gctx, jobs[i].raw)
			if err != nil {
				return err
			}
			urls[i] = url
			return nil
		})
	}
	err := g.Wait()
	for i := range jobs {
		if urls[i] != "" {
			changes.uploaded = append(changes.uploaded, urls[i])
		}
	}
	if err != nil {
		a.deleteImages(ctx, profile.UserID, changes.uploaded)
		return photoChanges{}, err
	}
	for i := range jobs {
		if strSet(*jobs[i].dst) {
			changes.replaced = append(changes.replaced, **jobs[i].dst)
		}
		*jobs[i].dst = &urls[i]
	}
	return changes, nil
}

// deleteImages es best-effort: un asset que no se pudo borrar no debe fallar la operación.
func (a *riderProfileApp) deleteImages(ctx context.Context, userID uuid.UUID, urls []string) {
	if a.uploader == nil {
		return
	}
	for _, url := range urls {
		if err := a.uploader.DeleteImage(ctx, url); err != nil {
			slog.Warn("rider deleteImages cloudinary", "userID", userID, "url", url, "error", err)
		}
	}
}

func (a *riderProfileApp) DeletePhoto(ctx context.Context, userID uuid.UUID, field string) (dtos.RiderProfileResponse, error) {
	field = strings.TrimSpace(field)
	if !models.ValidRiderPhotoField(field) {
		return dtos.RiderProfileResponse{}, errcustom.ErrNotFound
	}

	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}

	urlPtr := profile.PhotoURLForField(field)
	if strSet(urlPtr) {
		if a.uploader != nil {
			if delErr := a.uploader.DeleteImage(ctx, *urlPtr); delErr != nil {
				// Best-effort: Cloudinary puede fallar (ya borrado, parse); igual limpiamos DB.
				slog.Warn("rider DeletePhoto cloudinary", "userID", userID, "field", field, "error", delErr)
			}
		}
		if err := a.repo.ClearPhoto(ctx, userID, field); err != nil {
			return dtos.RiderProfileResponse{}, err
		}
	}

	return a.GetProfile(ctx, userID)
}

func strSet(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}

func (a *riderProfileApp) Activate(ctx context.Context, userID, staffID uuid.UUID) (dtos.RiderProfileResponse, error) {
	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	comp := profile.Completion()
	if !comp.IsComplete {
		return dtos.RiderProfileResponse{}, errcustom.ErrRiderProfileIncomplete
	}
	now := time.Now()
	if err := a.repo.EnsureRiderRole(ctx, userID); err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	if err := a.repo.Activate(ctx, userID, staffID, now); err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	return a.GetProfile(ctx, userID)
}

func (a *riderProfileApp) Suspend(ctx context.Context, userID uuid.UUID) (dtos.RiderProfileResponse, error) {
	if err := a.repo.Suspend(ctx, userID); err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	_ = a.repo.DisableRiderRole(ctx, userID)
	return a.GetProfile(ctx, userID)
}

func (a *riderProfileApp) TouchContact(ctx context.Context, userID uuid.UUID, notes *string) (dtos.RiderProfileResponse, error) {
	var n *string
	if notes != nil {
		v := strings.TrimSpace(*notes)
		n = &v
	}
	if err := a.repo.TouchContact(ctx, userID, time.Now(), n); err != nil {
		return dtos.RiderProfileResponse{}, err
	}
	return a.GetProfile(ctx, userID)
}

func (a *riderProfileApp) IsActiveRider(ctx context.Context, userID uuid.UUID) (bool, error) {
	profile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			// Legacy: riders creados antes del expediente KYC.
			return true, nil
		}
		return false, err
	}
	return profile.Status == models.RiderProfileActive, nil
}

func (a *riderProfileApp) upload(ctx context.Context, imageB64 string) (string, error) {
	if a.uploader == nil {
		return "", errcustom.ErrUploadingImage
	}
	res, err := a.uploader.UploadImage(ctx, imageB64)
	if err != nil || res == nil || res.SecureURL == "" {
		return "", errcustom.ErrUploadingImage
	}
	return res.SecureURL, nil
}

func (a *riderProfileApp) toResponse(
	p *models.RiderProfile,
	name, sur, phone, email string,
	hasRole bool,
) dtos.RiderProfileResponse {
	comp := p.Completion()
	out := dtos.RiderProfileResponse{
		UserID:             p.UserID.String(),
		Status:             string(p.Status),
		Name:               name,
		SurName:            sur,
		Phone:              phone,
		Email:              email,
		HasRiderRole:       hasRole,
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
		Notes:              p.Notes,
		Completion:         dtos.CompletionFromModel(comp),
	}
	if !p.CreatedAt.IsZero() {
		out.CreatedAt = p.CreatedAt.Format(time.RFC3339)
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = p.UpdatedAt.Format(time.RFC3339)
	}
	if p.LastContactedAt != nil {
		s := p.LastContactedAt.Format(time.RFC3339)
		out.LastContactedAt = &s
	}
	if p.ActivatedAt != nil {
		s := p.ActivatedAt.Format(time.RFC3339)
		out.ActivatedAt = &s
	}
	if p.SubmittedAt != nil {
		s := p.SubmittedAt.Format(time.RFC3339)
		out.SubmittedAt = &s
	}
	if p.ReviewedAt != nil {
		s := p.ReviewedAt.Format(time.RFC3339)
		out.ReviewedAt = &s
	}
	out.ReviewReason = p.ReviewReason
	return out
}
