package dto

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	StaffUserStatusAll     = "all"
	StaffUserStatusActive  = "active"
	StaffUserStatusBlocked = "blocked"
)

type StaffUserListItem struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	SurName             string     `json:"sur_name,omitempty"`
	Phone               string     `json:"phone,omitempty"`
	Email               string     `json:"email,omitempty"`
	Enabled             bool       `json:"enabled"`
	Roles               []string   `json:"roles"`
	CreatedAt           time.Time  `json:"created_at"`
	ScheduledDeletionAt *time.Time `json:"scheduled_deletion_at,omitempty"`
}

type StaffUserListResponse struct {
	Items      []StaffUserListItem `json:"items"`
	NextCursor *string             `json:"next_cursor"`
	HasMore    bool                `json:"has_more"`
}

type StaffUserDetailResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	SurName             string     `json:"sur_name,omitempty"`
	Phone               string     `json:"phone,omitempty"`
	Email               string     `json:"email,omitempty"`
	Enabled             bool       `json:"enabled"`
	Roles               []string   `json:"roles"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ScheduledDeletionAt *time.Time `json:"scheduled_deletion_at,omitempty"`
}

type StaffUserCandidateResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Name    string    `json:"name"`
	SurName string    `json:"sur_name,omitempty"`
	Phone   string    `json:"phone,omitempty"`
	Email   string    `json:"email,omitempty"`
	Enabled bool      `json:"enabled"`
	Roles   []string  `json:"roles"`
}

type StaffUpdateUserRequest struct {
	Name    *string `json:"name" validate:"omitempty,min=3"`
	SurName *string `json:"sur_name" validate:"omitempty,min=3"`
	Email   *string `json:"email" validate:"omitempty,email"`
}

func (r *StaffUpdateUserRequest) Validate() error {
	_ = conform.Struct(context.Background(), r)
	return validate.Struct(r)
}

type ListStaffUsersQuery struct {
	Cursor string
	Limit  int
	Query  string
	Status string
}
