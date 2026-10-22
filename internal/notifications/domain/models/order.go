package models

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Order struct {
	bun.BaseModel `bun:"table:orders.orders"`
	ID            uuid.UUID       `bun:"id,pk"`
	Number        int             `bun:"number" json:"Numero De Orden"`
	ShopID        uuid.UUID       `bun:"shop_id" json:"ID De La Tienda"`
	CustomerID    uuid.UUID       `bun:"customer_id" json:"ID Del Cliente"`
	Total         float64         `bun:"total_price" json:"Total"`
	DeliveryFee   float64         `bun:"delivery_fee" json:"DeliveryFee"`
	Details       json.RawMessage `bun:"details,type:jsonb"`

	Shop     Shop     `bun:"rel:belongs-to,join:shop_id=id" json:"Tienda"`
	Customer Customer `bun:"rel:belongs-to,join:customer_id=id" json:"Cliente"`
}

// SubtotalWithoutDelivery is the order cost excluding delivery fee.
func (o Order) SubtotalWithoutDelivery() float64 {
	sub := o.Total - o.DeliveryFee
	if sub < 0 {
		return 0
	}
	return sub
}

// BcvRate is the frozen checkout rate stored in details.payment.bcv_rate.
func (o Order) BcvRate() float64 {
	if len(o.Details) == 0 {
		return 0
	}
	var details struct {
		Payment *struct {
			BcvRate float64 `json:"bcv_rate"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(o.Details, &details); err != nil || details.Payment == nil {
		return 0
	}
	return details.Payment.BcvRate
}

type Shop struct {
	bun.BaseModel `bun:"table:business.shops"`
	ID            uuid.UUID `bun:"id,pk" json:"ID"`
	OwnerID       uuid.UUID `bun:"owner_id" json:"OwnerID"`
	Name          string    `bun:"name" json:"Nombre De La Tienda"`
	Address       string    `bun:"address" json:"Direccion"`

	Owner Customer `bun:"rel:belongs-to,join:owner_id=id" json:"Owner"`
}

type Customer struct {
	bun.BaseModel `bun:"table:users.accounts"`
	ID            uuid.UUID `bun:"id,pk" json:"ID"`
	FirstName     string    `bun:"name" json:"Nombre"`
	LastName      string    `bun:"sur_name" json:"Apellido"`
	Phone         *string   `bun:"phone" json:"Telefono"`
}

func (c Customer) FullName() string {
	if c.LastName == "" {
		return c.FirstName
	}
	return c.FirstName + " " + c.LastName
}

func (c Customer) PhoneOrDash() string {
	if c.Phone == nil || *c.Phone == "" {
		return "—"
	}
	return *c.Phone
}
