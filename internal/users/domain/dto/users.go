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
	BirthdayDate string `json:"birthday_date" validate:"omitempty,datetime=2006-01-02" example:"1990-05-20"`
	Password     string `json:"password" validate:"required,min=8" example:"claveSegura123"`
}

func (c *RegisterCustomer) Validate() error {

	_ = conform.Struct(context.Background(), c)

	if err := validate.Struct(c); err != nil {
		return err
	}

	if c.BirthdayDate == "" {
		return nil
	}

	birthDate, err := time.Parse("2006-01-02", c.BirthdayDate)
	if err != nil {
		return err
	}
	hoy := time.Now()
	edad := hoy.Year() - birthDate.Year()
	if hoy.YearDay() < birthDate.YearDay() {
		edad--
	}

	if edad < 18 {
		return errors.New("the age requirement is not met")
	}

	return nil
}

// RegisterCustomerFromGoogle registra un customer a partir de un login de
// Google: no incluye phone ni password porque Google ya verificó la
// propiedad del email; el teléfono se completa más adelante desde el perfil.
type RegisterCustomerFromGoogle struct {
	Name    string `example:"Carlos"`
	SurName string `example:"Ramírez"`
	Email   string `validate:"required,email" example:"carlos.ramirez@example.com"`
}

// UpdateCustomer es una actualización parcial: solo los campos presentes en
// el body se modifican, el resto de la cuenta queda intacto.
type UpdateCustomer struct {
	Name    *string `json:"name" validate:"omitempty,min=3" example:"Carlos"`
	SurName *string `json:"sur_name" validate:"omitempty,min=3" example:"Ramírez"`
	Phone   *string `json:"phone" validate:"omitempty,e164" example:"+573001112233"`
	Email   *string `json:"email" validate:"omitempty,email" example:"carlos.ramirez@example.com"`
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
	ID           uuid.UUID  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name         string     `json:"name" example:"Carlos"`
	SurName      string     `json:"sur_name" example:"Ramírez"`
	Phone        string     `json:"phone" example:"+573001112233"`
	Email        string     `json:"email" example:"carlos.ramirez@example.com"`
	BirthdayDate string     `json:"birthday_date" example:"1990-05-20"`
	Role         string     `json:"role" example:"customer"`
	ShopID       *uuid.UUID `json:"shop_id,omitempty" example:"456e7890-e89b-12d3-a456-426614174001"`
	Shop         *ShopInfo  `json:"shop,omitempty"`
	// ScheduledDeletionAt viene presente solo si el usuario solicitó eliminar
	// su cuenta: es la fecha en la que, de no volver a iniciar sesión antes,
	// el worker de finalización la marcará deleted_at.
	ScheduledDeletionAt *time.Time `json:"scheduled_deletion_at,omitempty" example:"2026-09-06T00:00:00Z"`
}

type ShopInfo struct {
	ID     uuid.UUID `json:"id" example:"456e7890-e89b-12d3-a456-426614174001"`
	Name   string    `json:"name" example:"Mi Tienda"`
	Opened bool      `json:"opened" example:"true"`
}
