package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	userModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type riderProfileRepo struct {
	db *bun.DB
}

func NewRiderProfileRepo(db *bun.DB) ports.RiderProfileRepo {
	return &riderProfileRepo{db: db}
}

func (r *riderProfileRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.RiderProfile, error) {
	var p models.RiderProfile
	err := r.db.NewSelect().Model(&p).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("rider GetByUserID failed", "userID", userID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return &p, nil
}

func (r *riderProfileRepo) ClearPhoto(ctx context.Context, userID uuid.UUID, field string) error {
	col, ok := models.RiderPhotoColumns[field]
	if !ok {
		return errcustom.ErrNotFound
	}
	now := time.Now()
	// col solo desde whitelist estática.
	q := r.db.NewUpdate().
		Model((*models.RiderProfile)(nil)).
		Set(col+" = NULL").
		Set("updated_at = ?", now).
		Where("user_id = ?", userID)
	res, err := q.Exec(ctx)
	if err != nil {
		slog.Error("rider ClearPhoto failed", "userID", userID, "field", field, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrNotFound
	}
	return nil
}

func (r *riderProfileRepo) Upsert(ctx context.Context, profile *models.RiderProfile) error {
	if profile.Status == "" {
		profile.Status = models.RiderProfileDraft
	}

	_, err := r.upsertQuery(profile).
		Set("status = EXCLUDED.status").
		Set("notes = COALESCE(EXCLUDED.notes, p.notes)").
		Exec(ctx)
	if err != nil {
		slog.Error("rider Upsert failed", "userID", profile.UserID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *riderProfileRepo) SaveApplication(ctx context.Context, profile *models.RiderProfile) error {
	if profile.Status == "" {
		profile.Status = models.RiderProfileDraft
	}

	// El WHERE del ON CONFLICT evita pisar una solicitud que el staff movió mientras se subían las fotos.
	res, err := r.upsertQuery(profile).
		Where("p.status IN (?)", bun.In(models.ApplicantEditableStatuses)).
		Exec(ctx)
	if err != nil {
		slog.Error("rider SaveApplication failed", "userID", profile.UserID, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrRiderApplicationLocked
	}
	return nil
}

// upsertQuery arma el upsert parcial de los datos del expediente; status y notes los decide cada caller.
func (r *riderProfileRepo) upsertQuery(profile *models.RiderProfile) *bun.InsertQuery {
	now := time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	// Bun inserta en "riders"."profiles" sin alias usable en ON CONFLICT.
	// Sin "AS p", Postgres no resuelve ni "profiles.col" ni refs schema-qualified.
	return r.db.NewInsert().
		Model(profile).
		ModelTableExpr("riders.profiles AS p").
		On("CONFLICT (user_id) DO UPDATE").
		Set("profile_photo_url = COALESCE(EXCLUDED.profile_photo_url, p.profile_photo_url)").
		Set("vehicle_type = COALESCE(EXCLUDED.vehicle_type, p.vehicle_type)").
		Set("vehicle_plate = COALESCE(EXCLUDED.vehicle_plate, p.vehicle_plate)").
		Set("vehicle_brand = COALESCE(EXCLUDED.vehicle_brand, p.vehicle_brand)").
		Set("vehicle_model = COALESCE(EXCLUDED.vehicle_model, p.vehicle_model)").
		Set("vehicle_color = COALESCE(EXCLUDED.vehicle_color, p.vehicle_color)").
		Set("id_photo_url = COALESCE(EXCLUDED.id_photo_url, p.id_photo_url)").
		Set("carnet_photo_url = COALESCE(EXCLUDED.carnet_photo_url, p.carnet_photo_url)").
		Set("license_photo_url = COALESCE(EXCLUDED.license_photo_url, p.license_photo_url)").
		Set("circulation_card_url = COALESCE(EXCLUDED.circulation_card_url, p.circulation_card_url)").
		Set("vehicle_photo_1_url = COALESCE(EXCLUDED.vehicle_photo_1_url, p.vehicle_photo_1_url)").
		Set("vehicle_photo_2_url = COALESCE(EXCLUDED.vehicle_photo_2_url, p.vehicle_photo_2_url)").
		Set("vehicle_photo_3_url = COALESCE(EXCLUDED.vehicle_photo_3_url, p.vehicle_photo_3_url)").
		Set("vehicle_photo_4_url = COALESCE(EXCLUDED.vehicle_photo_4_url, p.vehicle_photo_4_url)").
		Set("updated_at = EXCLUDED.updated_at")
}

func (r *riderProfileRepo) SubmitApplication(ctx context.Context, userID uuid.UUID, submittedAt time.Time) error {
	res, err := r.db.NewUpdate().
		Model((*models.RiderProfile)(nil)).
		Set("status = ?", models.RiderProfilePendingReview).
		Set("submitted_at = ?", submittedAt).
		Set("review_reason = NULL").
		Set("updated_at = ?", submittedAt).
		Where("user_id = ?", userID).
		Where("status IN (?)", bun.In(models.ApplicantEditableStatuses)).
		Exec(ctx)
	if err != nil {
		slog.Error("rider SubmitApplication failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrRiderApplicationLocked
	}
	return nil
}

// ReviewApplication solo mueve solicitudes en pending_review: si el staff decide dos veces,
// o el conductor alcanzó a reenviar, la segunda decisión no pisa el estado.
func (r *riderProfileRepo) ReviewApplication(
	ctx context.Context,
	userID, staffID uuid.UUID,
	status models.RiderProfileStatus,
	reason string,
	reviewedAt time.Time,
) error {
	res, err := r.db.NewUpdate().
		Model((*models.RiderProfile)(nil)).
		Set("status = ?", status).
		Set("review_reason = ?", reason).
		Set("reviewed_at = ?", reviewedAt).
		Set("reviewed_by = ?", staffID).
		Set("updated_at = ?", reviewedAt).
		Where("user_id = ?", userID).
		Where("status = ?", models.RiderProfilePendingReview).
		Exec(ctx)
	if err != nil {
		slog.Error("rider ReviewApplication failed", "userID", userID, "status", status, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrRiderReviewNotPending
	}
	return nil
}

func (r *riderProfileRepo) ListByStatus(ctx context.Context, status models.RiderProfileStatus) ([]models.RiderProfile, error) {
	var list []models.RiderProfile
	q := r.db.NewSelect().Model(&list).OrderExpr("updated_at ASC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Scan(ctx); err != nil {
		slog.Error("rider ListByStatus failed", "status", status, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return list, nil
}

func (r *riderProfileRepo) Activate(ctx context.Context, userID, staffID uuid.UUID, activatedAt time.Time) error {
	res, err := r.db.NewUpdate().
		Model((*models.RiderProfile)(nil)).
		Set("status = ?", models.RiderProfileActive).
		Set("activated_at = ?", activatedAt).
		Set("activated_by = ?", staffID).
		Set("updated_at = ?", activatedAt).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		slog.Error("rider Activate failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrNotFound
	}
	return nil
}

func (r *riderProfileRepo) Suspend(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	res, err := r.db.NewUpdate().
		Model((*models.RiderProfile)(nil)).
		Set("status = ?", models.RiderProfileSuspended).
		Set("updated_at = ?", now).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		slog.Error("rider Suspend failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrNotFound
	}
	return nil
}

func (r *riderProfileRepo) TouchContact(ctx context.Context, userID uuid.UUID, contactedAt time.Time, notes *string) error {
	q := r.db.NewUpdate().
		Model((*models.RiderProfile)(nil)).
		Set("last_contacted_at = ?", contactedAt).
		Set("updated_at = ?", contactedAt).
		Where("user_id = ?", userID)
	if notes != nil {
		q = q.Set("notes = ?", *notes)
	}
	res, err := q.Exec(ctx)
	if err != nil {
		slog.Error("rider TouchContact failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errcustom.ErrNotFound
	}
	return nil
}

func (r *riderProfileRepo) HasRiderRole(ctx context.Context, userID uuid.UUID) (bool, error) {
	exists, err := r.db.NewSelect().
		TableExpr("users.user_roles ur").
		Join("JOIN users.roles ro ON ro.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Where("ro.name = ?", "Rider").
		Where("ur.enabled = TRUE").
		Exists(ctx)
	if err != nil {
		slog.Error("rider HasRiderRole failed", "userID", userID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return exists, nil
}

func (r *riderProfileRepo) EnsureRiderRole(ctx context.Context, userID uuid.UUID) error {
	res, err := r.db.NewRaw(`
		UPDATE users.user_roles ur
		SET enabled = TRUE
		FROM users.roles ro
		WHERE ur.role_id = ro.id AND ur.user_id = ? AND ro.name = 'Rider'
	`, userID).Exec(ctx)
	if err != nil {
		slog.Error("rider EnsureRiderRole update failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		return nil
	}
	_, err = r.db.NewRaw(`
		INSERT INTO users.user_roles (id, user_id, role_id, enabled)
		SELECT ?, ?, id, TRUE FROM users.roles WHERE name = 'Rider'
	`, id.New(), userID).Exec(ctx)
	if err != nil {
		slog.Error("rider EnsureRiderRole insert failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *riderProfileRepo) DisableRiderRole(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.NewRaw(`
		UPDATE users.user_roles ur
		SET enabled = FALSE
		FROM users.roles ro
		WHERE ur.role_id = ro.id AND ur.user_id = ? AND ro.name = 'Rider'
	`, userID).Exec(ctx)
	if err != nil {
		slog.Error("rider DisableRiderRole failed", "userID", userID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *riderProfileRepo) FindAccountByPhone(ctx context.Context, phone string) (uuid.UUID, string, string, string, string, error) {
	phone = strings.TrimSpace(phone)
	var acc userModels.Account
	err := r.db.NewSelect().
		Model(&acc).
		Where("phone = ?", phone).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, "", "", "", "", errcustom.ErrNotFound
		}
		slog.Error("rider FindAccountByPhone failed", "phone", phone, "error", err)
		return uuid.Nil, "", "", "", "", errcustom.Unexpected(err)
	}
	sur, ph, em := "", "", ""
	if acc.SurName != nil {
		sur = *acc.SurName
	}
	if acc.Phone != nil {
		ph = *acc.Phone
	}
	if acc.Email != nil {
		em = *acc.Email
	}
	return acc.ID, acc.Name, sur, ph, em, nil
}

func (r *riderProfileRepo) GetAccountBasics(ctx context.Context, userID uuid.UUID) (string, string, string, string, error) {
	var acc userModels.Account
	err := r.db.NewSelect().
		Model(&acc).
		Where("id = ?", userID).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", "", errcustom.ErrNotFound
		}
		slog.Error("rider GetAccountBasics failed", "userID", userID, "error", err)
		return "", "", "", "", errcustom.Unexpected(err)
	}
	sur, ph, em := "", "", ""
	if acc.SurName != nil {
		sur = *acc.SurName
	}
	if acc.Phone != nil {
		ph = *acc.Phone
	}
	if acc.Email != nil {
		em = *acc.Email
	}
	return acc.Name, sur, ph, em, nil
}

func (r *riderProfileRepo) UpsertDeviceToken(ctx context.Context, token models.RiderDeviceToken) error {
	token.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().
		Model(&token).
		On("CONFLICT (rider_id) DO UPDATE").
		Set("fcm_token = EXCLUDED.fcm_token").
		Set("platform = EXCLUDED.platform").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		slog.Error("rider UpsertDeviceToken failed", "riderID", token.RiderID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *riderProfileRepo) GetDeviceTokensByRiderIDs(ctx context.Context, riderIDs []uuid.UUID) ([]models.RiderDeviceToken, error) {
	if len(riderIDs) == 0 {
		return nil, nil
	}
	var tokens []models.RiderDeviceToken
	err := r.db.NewSelect().Model(&tokens).Where("rider_id IN (?)", bun.In(riderIDs)).Scan(ctx)
	if err != nil {
		slog.Error("rider GetDeviceTokensByRiderIDs failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return tokens, nil
}

func (r *riderProfileRepo) DeleteDeviceToken(ctx context.Context, riderID uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*models.RiderDeviceToken)(nil)).Where("rider_id = ?", riderID).Exec(ctx)
	if err != nil {
		slog.Error("rider DeleteDeviceToken failed", "riderID", riderID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}
