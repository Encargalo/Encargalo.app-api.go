package models

import (
	"time"

	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Items []Item

type Item struct {
	bun.BaseModel `bun:"table:products.items" swaggerignore:"true"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ShopID      uuid.UUID `bun:"shop_id" json:"shop_id"`
	CategoryID  uuid.UUID `bun:"category_id" json:"category_id"`
	Name        string    `bun:"name" json:"name"`
	Price       int       `bun:"price" json:"price"`
	Image       string    `bun:"image" json:"image"`
	Description string    `bun:"description" json:"description"`
	IsAvailable bool      `bun:"is_available"`
	Score       float32   `bun:"score" json:"score"`
	HasFlavors  bool      `bun:"has_flavors" json:"has_flavors"`
	CreatedAt   time.Time `bun:"created_at,default:now()" json:"-"`
	UpdatedAt   time.Time `bun:"updated_at,default:now()" json:"-"`
	DeletedAt   time.Time `bun:"deleted_at" json:"-"`

	ItemRule ItemsRules `bun:"rel:has-many,join:id=item_id" json:"rules,omitempty"`
}

func (i *Item) ToDomainDTO() dtos.ItemResponse {
	return dtos.ItemResponse{
		ID:          i.ID,
		ShopID:      i.ShopID,
		CategoryID:  i.CategoryID,
		Name:        i.Name,
		Price:       i.Price,
		Image:       i.Image,
		Description: i.Description,
		Score:       i.Score,
		HasFlavors:  i.HasFlavors,
		Rules:       i.ItemRule.ToDomainDTO(),
	}
}

func (i *Items) ToDomainDTO() dtos.ItemsResponse {

	var items dtos.ItemsResponse

	for _, v := range *i {
		items.Add(v.ToDomainDTO())
	}

	return items

}
