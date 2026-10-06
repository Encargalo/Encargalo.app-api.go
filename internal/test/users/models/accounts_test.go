package models_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestAccountToDomainDTO_CreatedAtInUTC(t *testing.T) {
	caracas := time.FixedZone("VET", -4*60*60)
	acc := models.Account{
		Name:      "Carlos",
		CreatedAt: time.Date(2026, time.March, 15, 10, 30, 0, 0, caracas),
	}

	got := acc.ToDomainDTO()

	assert.True(t, got.CreatedAt.Equal(acc.CreatedAt))
	assert.Equal(t, time.UTC, got.CreatedAt.Location())
}

func TestAccountToDomainDTO_BirthdayDate(t *testing.T) {
	birthday := time.Date(1990, time.May, 20, 0, 0, 0, 0, time.UTC)

	withBirthday := models.Account{Name: "Carlos", BirthdayDate: &birthday}
	withoutBirthday := models.Account{Name: "Carlos"}

	assert.Equal(t, "1990-05-20", withBirthday.ToDomainDTO().BirthdayDate)
	assert.Equal(t, "", withoutBirthday.ToDomainDTO().BirthdayDate)
}
