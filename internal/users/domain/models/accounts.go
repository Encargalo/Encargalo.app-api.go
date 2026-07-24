package models

import (
	"time"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// defaultRole es el rol asumido cuando Account no viene de un JOIN con
// user_roles (p. ej. justo tras un registro). Coincide con auth.models.Customer,
// pero se declara localmente para no acoplar el dominio de users al de auth.
const defaultRole = "Customer"

type Account struct {
	bun.BaseModel `bun:"table:users.accounts"`

	ID               uuid.UUID          `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	Name             string             `bun:"name,notnull"`
	SurName          *string            `bun:"sur_name"`
	Phone            *string            `bun:"phone"`
	Email            *string            `bun:"email"`
	Password         string             `bun:"password,notnull"`
	BirthdayDate     *time.Time         `bun:"birthday_date,nullzero"`
	ActivationStatus string             `bun:"activation_status,default:'in progress'"`
	Enabled          bool               `bun:"enabled,default:true"`
	CreatedAt        time.Time          `bun:"created_at,default:now()"`
	UpdatedAt        time.Time          `bun:"updated_at,default:now()"`
	DeletedAt        *time.Time         `bun:"deleted_at,soft_delete,nullzero"`
	PagoMovil        *CustomerPagoMovil `bun:"pago_movil,type:jsonb" json:"pago_movil,omitempty"`

	// Populated via JOIN — zero values for customers
	ShopID uuid.UUID     `bun:"-"`
	Role   string        `bun:"-"`
	Shop   *dto.ShopInfo `bun:"-"`
}

func (c *Account) BuildCustomerRegisterModel(customer dto.RegisterCustomer) {
	c.Name = customer.Name
	c.SurName = &customer.SurName
	c.Phone = &customer.Phone
	c.Email = &customer.Email
	c.Password = customer.Password
}

// BuildCustomerRegisterFromGoogleModel construye una cuenta a partir de un
// login de Google: el teléfono queda sin asignar hasta que el usuario lo
// complete desde su perfil, y no hay password porque la identidad ya fue
// verificada por Google.
func (c *Account) BuildCustomerRegisterFromGoogleModel(customer dto.RegisterCustomerFromGoogle) {
	c.Name = customer.Name
	c.SurName = &customer.SurName
	c.Email = &customer.Email
}

// BuildCustomerUpdateModel es una actualización parcial: solo copia los
// campos que vinieron en el request (no nil), dejando el resto en su valor
// cero para que el repo los omita del UPDATE.
func (c *Account) BuildCustomerUpdateModel(customer dto.UpdateCustomer) {
	if customer.Name != nil {
		c.Name = *customer.Name
	}
	if customer.SurName != nil {
		c.SurName = customer.SurName
	}
	if customer.Phone != nil {
		c.Phone = customer.Phone
	}
	if customer.Email != nil {
		c.Email = customer.Email
	}
	c.UpdatedAt = time.Now()
}

func (c *Account) BuildCustomerUpdatePasswordModel(customer dto.UpdatePassword) {
	c.Password = customer.Password
}

func (c *Account) ToDomainDTO() dto.CustomerResponse {
	surName := ""
	if c.SurName != nil {
		surName = *c.SurName
	}

	email := ""
	if c.Email != nil {
		email = *c.Email
	}

	phone := ""
	if c.Phone != nil {
		phone = *c.Phone
	}

	role := c.Role
	if role == "" {
		role = defaultRole
	}

	response := dto.CustomerResponse{
		ID:      c.ID,
		Name:    c.Name,
		SurName: surName,
		Phone:   phone,
		Email:   email,
		Role:    role,
	}

	if c.ShopID != (uuid.UUID{}) {
		shopID := c.ShopID
		response.ShopID = &shopID
		response.Shop = c.Shop
	}

	return response
}
