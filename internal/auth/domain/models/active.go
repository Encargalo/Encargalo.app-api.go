package models

import (
	"time"

	"github.com/google/uuid"
)

type ActiveSession struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	UserType  Role       `json:"user_type"`
	ShopID    *uuid.UUID `json:"shop_id,omitempty"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

func (a *ActiveSession) BuildActiveSessionModel(userID uuid.UUID, userType, ipUser, userAgent string, shopID *uuid.UUID) {
	a.ID = uuid.New()
	a.UserID = userID
	a.UserType = Role(userType)
	a.ShopID = shopID
	a.IPAddress = ipUser
	a.UserAgent = userAgent
	a.CreatedAt = time.Now()
	a.ExpiresAt = time.Now().AddDate(1, 0, 0)
}
