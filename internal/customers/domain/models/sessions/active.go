package sessions

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ActiveSession struct {
	bun.BaseModel `bun:"table:sessions.active"`

	ID         uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	UserID     uuid.UUID  `bun:"user_id,type:uuid,notnull"`
	UserType   string     `bun:"user_type,notnull"`
	IPAddress  string     `bun:"ip_address,notnull"`
	UserAgent  string     `bun:"user_agent,notnull"`
	CreatedAt  time.Time  `bun:"created_at,default:now()"`
	ExpiresAt  time.Time  `bun:"expires_at,notnull"`
	LastAccess *time.Time `bun:"last_access,nullzero"`
}

func (a *ActiveSession) BuildActiveSessionModel(userID uuid.UUID, userType, ipUser, userAgent string) {
	a.UserID = userID
	a.UserType = userType
	a.IPAddress = ipUser
	a.UserAgent = userAgent
	a.ExpiresAt = time.Now().AddDate(0, 1, 0)

}
