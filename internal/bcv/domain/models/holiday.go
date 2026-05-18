package models

import (
	"time"

	"github.com/uptrace/bun"
)

type BankingHoliday struct {
	bun.BaseModel `bun:"table:bcv.banking_holidays"`

	Date time.Time `bun:"date,pk,type:date"`
}
