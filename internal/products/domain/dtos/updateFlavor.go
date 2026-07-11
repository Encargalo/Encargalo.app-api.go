package dtos

import "context"

type UpdateFlavor struct {
	Name        string  `json:"name" validate:"required" example:"Picante"`
	Description *string `json:"description" example:"Con salsa picante casera"`
}

func (u *UpdateFlavor) Validate() error {
	_ = conform.Struct(context.Background(), u)
	return validate.Struct(u)
}
