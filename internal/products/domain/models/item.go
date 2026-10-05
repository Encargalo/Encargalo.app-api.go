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

	ID                 uuid.UUID          `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ShopID             uuid.UUID          `bun:"shop_id" json:"shop_id"`
	CategoryID         uuid.UUID          `bun:"category_id" json:"category_id"`
	Name               string             `bun:"name" json:"name"`
	Price              float64            `bun:"price" json:"price"`
	Image              string             `bun:"image" json:"image"`
	ImagePublicID      string             `bun:"image_public_id" json:"-"`
	ImageHash          string             `bun:"image_hash" json:"-"`
	Description        string             `bun:"description" json:"description"`
	IsAvailable        bool               `bun:"is_available"`
	AvailabilityStatus AvailabilityStatus `bun:"availability_status,notnull,default:'active'"`
	Score              float32            `bun:"score" json:"score"`
	HasFlavors         bool               `bun:"has_flavors" json:"has_flavors"`
	Details            ItemDetails        `bun:"details,type:jsonb,notnull" json:"details"`
	CreatedAt          time.Time          `bun:"created_at,default:now()" json:"-"`
	UpdatedAt          time.Time          `bun:"updated_at,default:now()" json:"-"`
	DeletedAt          *time.Time         `bun:"deleted_at" json:"-"`

	Category *Category     `bun:"rel:belongs-to,join:category_id=id" json:"-"`
	Discount *ItemDiscount `bun:"rel:has-one,join:id=item_id" json:"-"`
}

// ShopPrice devuelve el precio que realmente cobra el negocio, tras aplicar
// su propio descuento (si tiene uno activo). Es el precio base sobre el que
// luego se compone la promo de lanzamiento de la plataforma (día 1).
func (i *Item) ShopPrice() float64 {
	if i.Discount == nil || !i.Discount.IsCurrentlyActive() {
		return i.Price
	}
	return i.Discount.Apply(i.Price)
}

func (i *Item) BuildToModel(item dtos.RegisterItem, imageURL, imgaePublicID, imageHash string) {
	i.ShopID = item.ShopID
	i.CategoryID = item.CategoryID
	i.Name = item.Name
	i.Price = item.Price
	i.Image = imageURL
	i.ImagePublicID = imgaePublicID
	i.ImageHash = imageHash
	i.Description = item.Description
	i.AvailabilityStatus = AvailabilityActive
	i.IsAvailable = true
	i.HasFlavors = item.HasFlavors
}

func (i *Item) ToDomainDTO(categoryDetails CategoryDetails) dtos.ItemResponse {
	shopPrice := i.ShopPrice()
	effective := EffectiveItemAvailability(i.AvailabilityStatus, i.Details)

	resp := dtos.ItemResponse{
		ID:                 i.ID,
		ShopID:             i.ShopID,
		CategoryID:         i.CategoryID,
		Name:               i.Name,
		Ref:                i.Price,
		EffectiveRef:       float64(shopPrice),
		HasShopDiscount:    i.Discount != nil && i.Discount.IsCurrentlyActive(),
		Image:              i.Image,
		Description:        i.Description,
		Score:              i.Score,
		HasFlavors:         i.HasFlavors,
		AvailabilityStatus: string(i.AvailabilityStatus),
		EffectiveStatus:    string(effective),
		CanOrder:           effective.CanOrder(),
	}

	if i.HasFlavors {
		resp.FlavorSelection = i.Details.ToFlavorSelection(categoryDetails)
	}

	if i.Details.HasAdditions() {
		resp.HasAdditions = true
		resp.AdditionSelection = i.Details.ToAdditionSelection()
	}

	if i.HasFlavors || i.Details.HasAdditions() {
		resp.Details = i.Details.ToDomainDTO()
	}

	if i.Category != nil {
		resp.Schedules = i.Category.Schedules.ToDomainDTO()
	}

	if i.Discount != nil && i.Discount.DeletedAt == nil {
		discountDTO := i.Discount.ToDomainDTO()
		resp.ShopDiscount = &discountDTO
	}

	return resp
}

func (i *Item) ToCustomerDTO(categoryDetails CategoryDetails) dtos.ItemResponse {
	resp := i.ToDomainDTO(categoryDetails)
	resp.Details = i.Details.ToCustomerDTO()
	return resp
}

func (i *Items) ToCustomerDTO(categoryDetails CategoryDetails) dtos.ItemsResponse {
	var items dtos.ItemsResponse
	for _, v := range *i {
		items.Add(v.ToCustomerDTO(categoryDetails))
	}
	return items
}

func (i *Items) ToDomainDTO(categoryDetails CategoryDetails) dtos.ItemsResponse {

	var items dtos.ItemsResponse

	for _, v := range *i {
		items.Add(v.ToDomainDTO(categoryDetails))
	}

	return items

}

func (i *Items) ToDomainDTOWithCategory() dtos.ItemsResponse {
	var items dtos.ItemsResponse

	for _, v := range *i {
		categoryDetails := DefaultCategoryDetails()
		if v.Category != nil {
			categoryDetails = v.Category.Details
		}
		items.Add(v.ToDomainDTO(categoryDetails))
	}

	return items
}
