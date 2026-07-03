package models

import (
	"encoding/json"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ItemsOrder struct {
	bun.BaseModel `bun:"table:orders.order_items"`

	ID          uuid.UUID       `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	OrderID     uuid.UUID       `bun:"order_id"`
	ItemID      uuid.UUID       `bun:"item_id"`
	Name        string          `bun:"name"`
	Image       string          `bun:"image"`
	Amount      int             `bun:"amount"`
	UnitPrice   int             `bun:"unit_price"`
	TotalPrice  int             `bun:"total_price"`
	Observation string          `bun:"observation"`
	Flavors     json.RawMessage `bun:"flavors,type:jsonb"`

	Additions []AdditionsOrder `bun:"rel:has-many,join:id=order_item_id"`

	// FlavorsDetails is resolved in-memory from Flavors (id -> amount) once
	// the referenced flavors are looked up; it feeds Details.BuildDetails
	// and is never persisted directly.
	FlavorsDetails []ItemFlavor `bun:"-"`
}

// ItemFlavor is a chosen flavor with its resolved name and amount.
type ItemFlavor struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Amount int       `json:"amount"`
}

// SetFlavorsDetails resolves this item's chosen flavor IDs (stored as
// {flavorID: amount} in Flavors) into named entries using the given catalog.
func (io *ItemsOrder) SetFlavorsDetails(flavors []Flavor) {
	nameMap := make(map[uuid.UUID]string, len(flavors))
	for _, f := range flavors {
		nameMap[f.ID] = f.Name
	}

	var chosen map[string]int
	if err := json.Unmarshal(io.Flavors, &chosen); err != nil {
		return
	}

	io.FlavorsDetails = make([]ItemFlavor, 0, len(chosen))
	for idStr, amount := range chosen {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		io.FlavorsDetails = append(io.FlavorsDetails, ItemFlavor{
			ID:     id,
			Name:   nameMap[id],
			Amount: amount,
		})
	}
}

func (io *ItemsOrder) BuildDtoToModel(dto dtos.CreateItemsOrder, orderID uuid.UUID) {
	io.ID = uuid.New()
	io.OrderID = orderID
	io.ItemID = dto.ItemID
	io.Amount = dto.Amount
	io.Observation = dto.Observation
	io.Flavors = convertertToJSONFlavors(dto.Flavors)
	Additions := make([]AdditionsOrder, len(dto.Additions))

	for i := range dto.Additions {
		Additions[i].BuildDtoToModel(dto.Additions[i], io.ID, io.Amount)
	}

	io.Additions = Additions
}

func (io *ItemsOrder) GetAdditionsID() []uuid.UUID {

	ids := make([]uuid.UUID, len(io.Additions))
	for i := range io.Additions {
		ids[i] = io.Additions[i].AdditionID
	}

	return ids

}

func convertertToJSONFlavors(flavors map[string]int) json.RawMessage {
	data, _ := json.Marshal(flavors)
	return data
}
