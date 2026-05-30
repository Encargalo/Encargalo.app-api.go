package models

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Account struct {
	bun.BaseModel `bun:"table:users.accounts"`

	ID               uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	Name             string     `bun:"name,notnull"`
	SurName          *string    `bun:"sur_name"`
	Phone            string     `bun:"phone,notnull"`
	Email            *string    `bun:"email"`
	Password         string     `bun:"password,notnull"`
	BirthdayDate     *time.Time `bun:"birthday_date,nullzero"`
	ActivationStatus string     `bun:"activation_status,default:'in progress'"`
	Enabled          bool       `bun:"enabled,default:true"`
	CreatedAt        time.Time  `bun:"created_at,default:now()"`
	UpdatedAt        time.Time  `bun:"updated_at,default:now()"`
	DeletedAt        *time.Time `bun:"deleted_at,soft_delete,nullzero"`

	// Populated via JOIN — zero values for customers
	ShopID uuid.UUID     `bun:"-"`
	Role   string        `bun:"-"`
	Shop   *dto.ShopInfo `bun:"-"`
}

func (c *Account) BuildCustomerRegisterModel(customer dto.RegisterCustomer) {
	c.Name = customer.Name
	c.SurName = &customer.SurName
	c.Phone = customer.Phone
	c.Email = &customer.Email
	c.Password = customer.Password
}

func (c *Account) BuildCustomerUpdateModel(customer dto.UpdateCustomer) {
	c.Name = customer.Name
	c.SurName = &customer.SurName
	c.Phone = customer.Phone
	c.Email = &customer.Email
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

	role := c.Role
	if role == "" {
		role = string(models.Customer)
	}

	response := dto.CustomerResponse{
		ID:      c.ID,
		Name:    c.Name,
		SurName: surName,
		Phone:   c.Phone,
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
