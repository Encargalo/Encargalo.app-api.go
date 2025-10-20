package dtos

import (
	"context"

	"github.com/go-playground/mold/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validate = validator.New()
	conform  = modifiers.New()
)

type SearchShopsByID struct {
	ID  uuid.UUID `query:"id"`
	Tag string    `query:"tag"`
}

func (s *SearchShopsByID) Validate() error {
	_ = conform.Struct(context.Background(), s)
	return validate.Struct(s)
}
