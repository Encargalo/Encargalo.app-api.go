package dto

import (
	"context"
	"errors"
	"time"

	"github.com/go-playground/mold/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validate = validator.New()
	conform  = modifiers.New()
)

type RegisterCustomer struct {
	Name         string `json:"name" validate:"required,min=3" example:"Carlos"`
	SurName      string `json:"sur_name" validate:"required,min=3" example:"Ramírez"`
	Phone        string `json:"phone" validate:"required,e164" example:"+573001112233"`
	Email        string `json:"email" example:"carlos.ramirez@example.com"`
	BirthdayDate string `json:"birthday_date" validate:"required,datetime=2006-01-02" example:"1990-05-20"`
	Password     string `json:"password" validate:"required,min=8" example:"claveSegura123"`
}

func (c *RegisterCustomer) Validate() error {

	_ = conform.Struct(context.Background(), c)

	if err := validate.Struct(c); err != nil {
		return err
	}

	birthDate, err := time.Parse("2006-01-02", c.BirthdayDate)
	hoy := time.Now()
	edad := hoy.Year() - birthDate.Year()
	if hoy.YearDay() < birthDate.YearDay() {
		edad--
	}

	if edad < 18 {
		return errors.New("the age requirement is not met")
	}

	return err
}

type UpdateCustomer struct {
	Name    string `json:"name" validate:"required,min=3" example:"Carlos"`
	SurName string `json:"sur_name" validate:"required,min=3" example:"Ramírez"`
	Phone   string `json:"phone" validate:"required,e164" example:"+573001112233"`
	Email   string `json:"email" example:"carlos.ramirez@example.com"`
}

func (c *UpdateCustomer) Validate() error {

	_ = conform.Struct(context.Background(), c)

	return validate.Struct(c)
}

type UpdatePassword struct {
	Password string `json:"password" validate:"required,min=8" example:"claveSegura123"`
}

func (c *UpdatePassword) Validate() error {

	_ = conform.Struct(context.Background(), c)

	return validate.Struct(c)
}

type CustomerResponse struct {
	ID           uuid.UUID `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name         string    `json:"name" example:"Carlos"`
	SurName      string    `json:"sur_name" example:"Ramírez"`
	Phone        string    `json:"phone" example:"+573001112233"`
	Email        string    `json:"email" example:"carlos.ramirez@example.com"`
	BirthdayDate string    `json:"birthday_date" example:"1990-05-20"`
}
