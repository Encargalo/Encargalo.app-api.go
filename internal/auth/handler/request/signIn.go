package request

import (
	"context"

	"github.com/go-playground/mold/modifiers"
	"github.com/go-playground/validator/v10"
)

var (
	validate = validator.New()
	conform  = modifiers.New()
)

type SignInRequest struct {
	Phone    string `json:"phone" validate:"required,e164" example:"+573001112233"`
	Password string `json:"password" validate:"required" example:"claveSegura123"`
}

func (s *SignInRequest) Validate() error {

	_ = conform.Struct(context.Background(), s)
	return validate.Struct(s)

}
