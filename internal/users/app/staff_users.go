package app

import (
	"context"
	"log/slog"
	"strings"

	"Encargalo.app-api.go/internal/pkg/cursor"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

const (
	staffUsersDefaultLimit = 20
	staffUsersMaxLimit     = 100
	staffUsersMinQueryLen  = 2
)

type staffUsersApp struct {
	repo ports.StaffUsersRepo
}

func NewStaffUsersApp(repo ports.StaffUsersRepo) ports.StaffUsersApp {
	return &staffUsersApp{repo: repo}
}

func (a *staffUsersApp) ListUsers(ctx context.Context, query dto.ListStaffUsersQuery) (dto.StaffUserListResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = staffUsersDefaultLimit
	}
	if limit > staffUsersMaxLimit {
		limit = staffUsersMaxLimit
	}

	status := strings.TrimSpace(strings.ToLower(query.Status))
	if status == "" {
		status = dto.StaffUserStatusAll
	}
	switch status {
	case dto.StaffUserStatusAll, dto.StaffUserStatusActive, dto.StaffUserStatusBlocked:
	default:
		return dto.StaffUserListResponse{}, errcustom.ErrInvalidFilter
	}

	search := strings.TrimSpace(query.Query)
	if search != "" && len(search) < staffUsersMinQueryLen {
		return dto.StaffUserListResponse{}, errcustom.ErrInvalidSearchQuery
	}

	var after *cursor.Key
	if raw := strings.TrimSpace(query.Cursor); raw != "" {
		k, err := cursor.Decode(raw)
		if err != nil {
			return dto.StaffUserListResponse{}, errcustom.ErrInvalidCursor
		}
		after = &k
	}

	rows, err := a.repo.ListManagedCustomers(ctx, search, status, limit+1, after)
	if err != nil {
		return dto.StaffUserListResponse{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	userIDs := make([]uuid.UUID, len(rows))
	for i := range rows {
		userIDs[i] = rows[i].ID
	}
	rolesByUser, err := a.repo.GetRolesForUserIDs(ctx, userIDs)
	if err != nil {
		return dto.StaffUserListResponse{}, err
	}

	items := make([]dto.StaffUserListItem, len(rows))
	for i := range rows {
		items[i] = accountToStaffListItem(&rows[i], rolesByUser[rows[i].ID])
	}

	resp := dto.StaffUserListResponse{
		Items:   items,
		HasMore: hasMore,
	}
	if hasMore {
		last := rows[len(rows)-1]
		encoded := cursor.Encode(cursor.Key{CreatedAt: last.CreatedAt, ID: last.ID})
		resp.NextCursor = &encoded
	}
	return resp, nil
}

func (a *staffUsersApp) FindCandidate(ctx context.Context, phone string) (dto.StaffUserCandidateResponse, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return dto.StaffUserCandidateResponse{}, errcustom.ErrInvalidSearchQuery
	}

	acc, err := a.repo.FindManagedCustomerByPhone(ctx, phone)
	if err != nil {
		return dto.StaffUserCandidateResponse{}, err
	}

	roles, err := a.rolesForUser(ctx, acc.ID)
	if err != nil {
		return dto.StaffUserCandidateResponse{}, err
	}

	return accountToCandidate(acc, roles), nil
}

func (a *staffUsersApp) rolesForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rolesByUser, err := a.repo.GetRolesForUserIDs(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}
	return rolesByUser[userID], nil
}

func (a *staffUsersApp) GetUser(ctx context.Context, userID uuid.UUID) (dto.StaffUserDetailResponse, error) {
	acc, err := a.repo.GetManagedCustomerByID(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}
	roles, err := a.rolesForUser(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}
	return accountToStaffDetail(acc, roles), nil
}

func (a *staffUsersApp) UpdateUser(ctx context.Context, staffID, userID uuid.UUID, body dto.StaffUpdateUserRequest) (dto.StaffUserDetailResponse, error) {
	if body.Name == nil && body.SurName == nil && body.Email == nil {
		return dto.StaffUserDetailResponse{}, errcustom.ErrInvalidSearchQuery
	}

	current, err := a.repo.GetManagedCustomerByID(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}

	if body.Email != nil {
		email := strings.TrimSpace(*body.Email)
		if email != "" {
			if dup, err := a.repo.SearchCustomerByEmailAndNotIDEquals(ctx, userID, email); err != nil {
				if err != errcustom.ErrNotFound {
					return dto.StaffUserDetailResponse{}, err
				}
			} else if dup != nil {
				return dto.StaffUserDetailResponse{}, errcustom.ErrEmailAlreadyExist
			}
		}
	}

	acc := &models.Account{
		Name:    current.Name,
		SurName: current.SurName,
		Email:   current.Email,
	}
	if body.Name != nil {
		acc.Name = *body.Name
	}
	if body.SurName != nil {
		acc.SurName = body.SurName
	}
	if body.Email != nil {
		email := strings.TrimSpace(*body.Email)
		acc.Email = &email
	}

	if err := a.repo.UpdateManagedCustomer(ctx, userID, acc); err != nil {
		return dto.StaffUserDetailResponse{}, err
	}

	slog.Info("staff actualizó usuario", "staffID", staffID, "userID", userID)
	return a.GetUser(ctx, userID)
}

func (a *staffUsersApp) BlockUser(ctx context.Context, staffID, userID uuid.UUID) (dto.StaffUserDetailResponse, error) {
	acc, err := a.repo.GetManagedCustomerByID(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}
	if !acc.Enabled {
		return a.accountDetailWithRoles(ctx, acc)
	}

	if err := a.repo.SetAccountEnabled(ctx, userID, false); err != nil {
		return dto.StaffUserDetailResponse{}, err
	}

	slog.Info("staff bloqueó usuario", "staffID", staffID, "userID", userID)
	return a.GetUser(ctx, userID)
}

func (a *staffUsersApp) UnblockUser(ctx context.Context, staffID, userID uuid.UUID) (dto.StaffUserDetailResponse, error) {
	acc, err := a.repo.GetManagedCustomerByID(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}
	if acc.Enabled {
		return a.accountDetailWithRoles(ctx, acc)
	}

	if err := a.repo.SetAccountEnabled(ctx, userID, true); err != nil {
		return dto.StaffUserDetailResponse{}, err
	}

	slog.Info("staff desbloqueó usuario", "staffID", staffID, "userID", userID)
	return a.GetUser(ctx, userID)
}

func (a *staffUsersApp) GrantStaff(ctx context.Context, staffID, userID uuid.UUID) (dto.StaffUserDetailResponse, error) {
	if _, err := a.repo.GetManagedCustomerByID(ctx, userID); err != nil {
		return dto.StaffUserDetailResponse{}, err
	}

	hasStaff, err := a.repo.HasStaffRole(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}
	if hasStaff {
		return a.GetUser(ctx, userID)
	}

	if err := a.repo.EnsureStaffRole(ctx, userID, staffID); err != nil {
		return dto.StaffUserDetailResponse{}, err
	}

	slog.Info("staff otorgó rol Staff", "staffID", staffID, "userID", userID)
	return a.GetUser(ctx, userID)
}

func (a *staffUsersApp) RevokeStaff(ctx context.Context, staffID, userID uuid.UUID) (dto.StaffUserDetailResponse, bool, error) {
	if staffID == userID {
		return dto.StaffUserDetailResponse{}, false, errcustom.ErrCannotRevokeOwnStaff
	}

	if _, err := a.repo.GetManagedCustomerByID(ctx, userID); err != nil {
		return dto.StaffUserDetailResponse{}, false, err
	}

	hasStaff, err := a.repo.HasStaffRole(ctx, userID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, false, err
	}
	if !hasStaff {
		resp, err := a.GetUser(ctx, userID)
		return resp, false, err
	}

	count, err := a.repo.CountEnabledStaff(ctx)
	if err != nil {
		return dto.StaffUserDetailResponse{}, false, err
	}
	if count <= 1 {
		return dto.StaffUserDetailResponse{}, false, errcustom.ErrLastStaffMember
	}

	if err := a.repo.DisableStaffRole(ctx, userID); err != nil {
		return dto.StaffUserDetailResponse{}, false, err
	}

	slog.Info("staff revocó rol Staff", "staffID", staffID, "userID", userID)
	resp, err := a.GetUser(ctx, userID)
	return resp, true, err
}

func (a *staffUsersApp) accountDetailWithRoles(ctx context.Context, acc *models.Account) (dto.StaffUserDetailResponse, error) {
	roles, err := a.rolesForUser(ctx, acc.ID)
	if err != nil {
		return dto.StaffUserDetailResponse{}, err
	}
	return accountToStaffDetail(acc, roles), nil
}

func accountToCandidate(acc *models.Account, roles []string) dto.StaffUserCandidateResponse {
	out := dto.StaffUserCandidateResponse{
		UserID:  acc.ID,
		Name:    acc.Name,
		Enabled: acc.Enabled,
		Roles:   roles,
	}
	if acc.SurName != nil {
		out.SurName = *acc.SurName
	}
	if acc.Phone != nil {
		out.Phone = *acc.Phone
	}
	if acc.Email != nil {
		out.Email = *acc.Email
	}
	return out
}

func accountToStaffListItem(acc *models.Account, roles []string) dto.StaffUserListItem {
	item := dto.StaffUserListItem{
		ID:        acc.ID,
		Name:      acc.Name,
		Enabled:   acc.Enabled,
		Roles:     roles,
		CreatedAt: acc.CreatedAt,
	}
	if acc.SurName != nil {
		item.SurName = *acc.SurName
	}
	if acc.Phone != nil {
		item.Phone = *acc.Phone
	}
	if acc.Email != nil {
		item.Email = *acc.Email
	}
	if acc.DeletionRequestedAt != nil {
		scheduled := acc.DeletionRequestedAt.Add(models.AccountDeletionGracePeriod)
		item.ScheduledDeletionAt = &scheduled
	}
	return item
}

func accountToStaffDetail(acc *models.Account, roles []string) dto.StaffUserDetailResponse {
	detail := dto.StaffUserDetailResponse{
		ID:        acc.ID,
		Name:      acc.Name,
		Enabled:   acc.Enabled,
		Roles:     roles,
		CreatedAt: acc.CreatedAt,
		UpdatedAt: acc.UpdatedAt,
	}
	if acc.SurName != nil {
		detail.SurName = *acc.SurName
	}
	if acc.Phone != nil {
		detail.Phone = *acc.Phone
	}
	if acc.Email != nil {
		detail.Email = *acc.Email
	}
	if acc.DeletionRequestedAt != nil {
		scheduled := acc.DeletionRequestedAt.Add(models.AccountDeletionGracePeriod)
		detail.ScheduledDeletionAt = &scheduled
	}
	return detail
}
