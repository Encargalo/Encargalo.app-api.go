package models

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ActivateAccount struct {
	bun.BaseModel `bun:"table:customers.activate_account"`

	ID             uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CustomerID     uuid.UUID  `bun:"customer_id,type:uuid"`
	ActivationCode string     `bun:"activation_code,notnull"`
	CreatedAt      time.Time  `bun:"created_at,default:now()"`
	UpdatedAt      time.Time  `bun:"updated_at,default:now()"`
	DeletedAt      *time.Time `bun:"deleted_at,nullzero"`
}

func (a *ActivateAccount) BuildActivateAccount(customerID uuid.UUID) {
	a.CustomerID = customerID
	a.ActivationCode = strconv.Itoa(generateCode())
}

func generateCode() int {
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	return rand.Intn(900000) + 100000
}
