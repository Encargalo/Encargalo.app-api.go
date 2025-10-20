package models

import (
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AdditionsOrder struct {
	bun.BaseModel `bun:"table:orders.additions_order"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	AdditionID  uuid.UUID `bun:"addition_id,type:uuid,notnull" json:"addition_id"`
	OrderItemID uuid.UUID `bun:"order_item_id,type:uuid,notnull" json:"order_item_id"`
	Amount      int       `bun:"amount,notnull" json:"amount"`
	UnitPrice   int       `bun:"unit_price,notnull" json:"unit_price"`
	TotalPrice  int       `bun:"total_price,notnull" json:"total_price"`
}

func (ao *AdditionsOrder) BuildDtoToModel(dto dtos.CreateAdditionsOrders, orderItemID uuid.UUID, amount int) {
	ao.AdditionID = dto.AdditionID
	ao.OrderItemID = orderItemID
	ao.Amount = amount
}
