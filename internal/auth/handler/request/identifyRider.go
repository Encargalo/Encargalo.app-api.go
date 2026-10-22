package request

import "context"

type IdentifyRiderPhoneRequest struct {
	Phone string `json:"phone_number" validate:"required,e164" example:"+573001112233"`
}

func (s *IdentifyRiderPhoneRequest) Validate() error {
	_ = conform.Struct(context.Background(), s)
	return validate.Struct(s)
}

type IdentifyRiderPhoneResponse struct {
	Next string `json:"next" example:"sign_in"`
}
