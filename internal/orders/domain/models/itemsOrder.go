package models

import (
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ItemsOrder struct {
	bun.BaseModel `bun:"table:orders.order_items"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	OrderID     uuid.UUID `bun:"order_id"`
	ItemID      uuid.UUID `bun:"item_id"`
	Amount      int       `bun:"amount"`
	UnitPrice   int       `bun:"unit_price"`
	TotalPrice  int       `bun:"total_price"`
	Observation string    `bun:"observation"`

	Additions []AdditionsOrder `bun:"rel:has-many,join:id=order_item_id"`
}

func (io *ItemsOrder) BuildDtoToModel(dto dtos.CreateItemsOrder, orderID uuid.UUID) {
	io.ID = uuid.New()
	io.OrderID = orderID
	io.ItemID = dto.ItemID
	io.Amount = dto.Amount
	io.Observation = dto.Observation
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
