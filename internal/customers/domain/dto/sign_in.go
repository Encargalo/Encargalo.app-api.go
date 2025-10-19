package dto

import "context"

type SignIn struct {
	PhoneNumber string `json:"phone_number" validate:"required,e164" example:"+573001112233"`
	Password    string `json:"password" validate:"required" example:"claveSegura123"`
}

func (s *SignIn) Validate() error {

	_ = conform.Struct(context.Background(), s)
	return validate.Struct(s)

}
