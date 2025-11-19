package entity

import "github.com/google/uuid"

type ShopDistance struct {
	ShopID   uuid.UUID
	Distance float64
	Duration int64
}
