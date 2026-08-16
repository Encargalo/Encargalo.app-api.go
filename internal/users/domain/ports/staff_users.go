package ports

import (
	"context"

	"Encargalo.app-api.go/internal/pkg/cursor"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
)

type StaffUsersApp interface {
	ListUsers(ctx context.Context, query dto.ListStaffUsersQuery) (dto.StaffUserListResponse, error)
	FindCandidate(ctx context.Context, phone string) (dto.StaffUserCandidateResponse, error)
	GetUser(ctx context.Context, userID uuid.UUID) (dto.StaffUserDetailResponse, error)
	UpdateUser(ctx context.Context, staffID, userID uuid.UUID, body dto.StaffUpdateUserRequest) (dto.StaffUserDetailResponse, error)
	BlockUser(ctx context.Context, staffID, userID uuid.UUID) (dto.StaffUserDetailResponse, error)
	UnblockUser(ctx context.Context, staffID, userID uuid.UUID) (dto.StaffUserDetailResponse, error)
}

type StaffUsersRepo interface {
	ListManagedCustomers(ctx context.Context, query string, status string, limit int, after *cursor.Key) ([]models.Account, error)
	FindManagedCustomerByPhone(ctx context.Context, phone string) (*models.Account, error)
	GetManagedCustomerByID(ctx context.Context, userID uuid.UUID) (*models.Account, error)
	SearchCustomerByEmailAndNotIDEquals(ctx context.Context, userID uuid.UUID, email string) (*models.Account, error)
	UpdateManagedCustomer(ctx context.Context, userID uuid.UUID, account *models.Account) error
	SetAccountEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
	GetRolesForUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]string, error)
}
