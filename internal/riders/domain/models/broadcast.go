package models

import (
	"encoding/json"
	"time"

	ordersModels "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/pkg/money"
	"github.com/google/uuid"
)

// CustomerLocation is the customer data exposed to riders. Before a rider
// accepts an order (new_order broadcasts to every available rider), only the
// location fields are set — Name/LastName/Phone stay empty. Once a rider has
// accepted the order (order_accepted, order_update), those messages go only
// to that rider, so Name/LastName/Phone are included too.
type CustomerLocation struct {
	Address      string  `json:"address"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Instructions string  `json:"instructions,omitempty"`
	Name         string  `json:"name,omitempty"`
	LastName     string  `json:"last_name,omitempty"`
	Phone        string  `json:"phone,omitempty"`
}

// RiderOrderFlavor is a chosen flavor with no pricing, just enough to prep
// the order (e.g. "Mitad Pepperoni").
type RiderOrderFlavor struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Amount int       `json:"amount"`
}

// RiderOrderAddition is a chosen addition with no pricing.
type RiderOrderAddition struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Amount int       `json:"amount"`
}

// RiderOrderItem is a product line with no pricing and no observation —
// riders only need what to hand over, not what it cost or why.
type RiderOrderItem struct {
	ID        uuid.UUID            `json:"id"`
	Name      string               `json:"name"`
	Image     string               `json:"image"`
	Amount    int                  `json:"amount"`
	Flavors   []RiderOrderFlavor   `json:"flavors,omitempty"`
	Additions []RiderOrderAddition `json:"additions,omitempty"`
}

// RiderOrder is the order view sent to riders: it drops fields riders have no
// use for (delivery address/coordinates live in CustomerLocation instead; the
// customer never sees a total, and Details carries customer PII/pricing).
// DeliveryCode is never included — it's shown by the customer to the rider
// in person, so the rider must not receive it through this channel at all.
type RiderOrder struct {
	ID            uuid.UUID `json:"id"`
	ShopID        uuid.UUID `json:"shop_id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	BatchID       uuid.UUID `json:"batch_id,omitempty"`
	Number        int       `json:"number"`
	MethodPayment string    `json:"method_payment"`
	Status        string    `json:"status"`
	DeliveryFee   float64   `json:"delivery_fee"`
	DeliveryFeeBs float64   `json:"delivery_fee_bs"`
	// CollectUSD es el total que el rider cobra en efectivo. Su pago ya está
	// en delivery_fee; el resto nos lo debe.
	CollectUSD      *float64         `json:"collect_usd,omitempty"`
	CollectBs       *float64         `json:"collect_bs,omitempty"`
	Items           []RiderOrderItem `json:"items,omitempty"`
	PickupCode      *string          `json:"pickup_code,omitempty"`
	RiderID         *uuid.UUID       `json:"rider_id,omitempty"`
	RiderAcceptedAt *time.Time       `json:"rider_accepted_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
}

// NewRiderOrder builds the rider-facing order view, applying the rider's
// delivery-fee share. assigned must be false for new_order (broadcast to
// every available rider, who hasn't committed to the order yet) and true
// once a rider has accepted it (order_accepted, order_update) — only then
// are Items and the pickup code included.
func NewRiderOrder(order ordersModels.Order, assigned bool) RiderOrder {
	riderOrder := RiderOrder{
		ID:              order.ID,
		ShopID:          order.ShopID,
		CustomerID:      order.CustomerID,
		BatchID:         order.BatchID,
		Number:          order.Number,
		MethodPayment:   order.MethodPayment,
		Status:          order.Status,
		DeliveryFee:     order.DeliveryFee * RiderShare,
		DeliveryFeeBs:   money.RoundBs(order.DeliveryFeeBs * RiderShare),
		RiderID:         order.RiderID,
		RiderAcceptedAt: order.RiderAcceptedAt,
		CreatedAt:       order.CreatedAt,
	}
	if order.MethodPayment == string(ordersModels.MethodEfectivo) {
		collect := order.TotalPrice
		collectBs := order.TotalPriceBs
		riderOrder.CollectUSD = &collect
		riderOrder.CollectBs = &collectBs
	}

	if assigned {
		riderOrder.Items = riderItemsFromDetails(order.Details)
		riderOrder.PickupCode = order.PickupCode
	}

	return riderOrder
}

// riderItemsFromDetails reads the product lines out of the order's Details
// JSONB. Unmarshaling straight into RiderOrderItem drops price/observation
// fields present in Details automatically, since they have no matching field.
func riderItemsFromDetails(details json.RawMessage) []RiderOrderItem {
	var parsed struct {
		Items []RiderOrderItem `json:"items"`
	}
	if err := json.Unmarshal(details, &parsed); err != nil {
		return nil
	}
	return parsed.Items
}
