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
