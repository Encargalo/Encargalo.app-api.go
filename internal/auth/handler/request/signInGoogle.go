package request

import "context"

type SignInGoogleRequest struct {
	IDToken string `json:"id_token" validate:"required" example:"eyJhbGciOiJSUzI1NiIsIn..."`
}

func (s *SignInGoogleRequest) Validate() error {

	_ = conform.Struct(context.Background(), s)
	return validate.Struct(s)

}
