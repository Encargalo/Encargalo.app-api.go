package models

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"time"

	"Encargalo.app-api.go/internal/pkg/id"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ActivateAccount struct {
	bun.BaseModel `bun:"table:users.activate_account"`

	ID             uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CustomerID     uuid.UUID  `bun:"customer_id,type:uuid"`
	ActivationCode string     `bun:"activation_code,notnull"`
	CreatedAt      time.Time  `bun:"created_at,default:now()"`
	UpdatedAt      time.Time  `bun:"updated_at,default:now()"`
	DeletedAt      *time.Time `bun:"deleted_at,nullzero"`
}

func (a *ActivateAccount) BuildActivateAccount(customerID uuid.UUID) error {
	code, err := generateCode()
	if err != nil {
		return err
	}
	a.ID = id.New()
	a.CustomerID = customerID
	a.ActivationCode = code
	return nil
}

// generateCode devuelve un código de 6 dígitos (100000-999999) con crypto/rand.
func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900_000))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n.Int64()+100_000, 10), nil
}
