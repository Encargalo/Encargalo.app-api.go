package models

import (
	"encoding/json"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/products/domain/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Order struct {
	bun.BaseModel `bun:"table:orders.orders"`

	ID                 uuid.UUID       `bun:"id,pk,type:uuid" json:"id"`
	ShopID             uuid.UUID       `bun:"shop_id" json:"shop_id"`
	CustomerID         uuid.UUID       `bun:"customer_id" json:"customer_id"`
	BatchID            uuid.UUID       `bun:"batch_id,nullzero,type:uuid" json:"batch_id,omitempty"`
	Number             int             `bun:"number" json:"number"`
	Address            string          `bun:"address" json:"address"`
	Latitude           float64         `bun:"latitude" json:"latitude"`
	Longitude          float64         `bun:"longitude" json:"longitude"`
	H3Cell             string          `bun:"h3_cell" json:"h3_cell,omitempty"`
	MethodPayment      string          `bun:"method_payment" json:"method_payment"`
	Status             string          `bun:"status" json:"status"`
	DeliveryFee        float64         `bun:"delivery_fee" json:"delivery_fee"`
	TotalPrice         float64         `bun:"total_price" json:"total_price"`
	DeliveryFeeBs      float64         `bun:"-" json:"delivery_fee_bs"`
	TotalPriceBs       float64         `bun:"-" json:"total_price_bs"`
	Details            json.RawMessage `bun:"details,type:jsonb" json:"details"`
	PickupCode         *string         `bun:"pickup_code" json:"pickup_code,omitempty"`
	DeliveryCode       *string         `bun:"delivery_code" json:"delivery_code,omitempty"`
	PickupCodeUsedAt   *time.Time      `bun:"pickup_code_used_at" json:"-"`
	DeliveryCodeUsedAt *time.Time      `bun:"delivery_code_used_at" json:"-"`
	RiderID            *uuid.UUID      `bun:"rider_id,nullzero,type:uuid" json:"rider_id,omitempty"`
	RiderAcceptedAt    *time.Time      `bun:"rider_accepted_at" json:"rider_accepted_at,omitempty"`
	CreatedAt          time.Time       `bun:"created_at,default:current_timestamp" json:"created_at"`
	// PendingAt: cuando la orden entró a Pending (reloj del auto-rechazo por no aceptación).
	PendingAt *time.Time `bun:"pending_at" json:"pending_at,omitempty"`
	// PendingAlertMinute: último minuto de aviso Discord emitido (4 u 8).
	PendingAlertMinute int `bun:"pending_alert_minute" json:"pending_alert_minute,omitempty"`

	// IsLaunchPromo: 50% de descuento al cliente aplicado (día 1 de vida del negocio).
	IsLaunchPromo bool `bun:"is_launch_promo" json:"is_launch_promo"`
	// IsZeroCommission: comisión 0% aplicada al liquidar (días 1-8 de vida del negocio).
	IsZeroCommission bool `bun:"is_zero_commission" json:"-"`
	// PromoSubsidyAmount es lo que puso la plataforma para cubrir el 50% del día 1 (0 el resto de los días).
	PromoSubsidyAmount float64 `bun:"promo_subsidy_amount" json:"promo_subsidy_amount"`

	ItemsOrders []ItemsOrder `bun:"rel:has-many,join:id=order_id" json:"-"`
}

func (o *Order) BuildDtoToModel(dto dtos.CreateOrder, batchID uuid.UUID) {
	o.ID = dto.ID
	o.ShopID = dto.ShopID
	o.CustomerID = dto.CustomerID
	o.BatchID = batchID
	o.Address = dto.Address.Address
	o.Latitude = dto.Address.Latitude
	o.Longitude = dto.Address.Longitude
	o.MethodPayment = dto.MethodPayment
	o.Status = string(StatusPending)
	o.ItemsOrders = make([]ItemsOrder, len(dto.CreateItemsOrder))

	for i := range dto.CreateItemsOrder {
		o.ItemsOrders[i].BuildDtoToModel(dto.CreateItemsOrder[i], dto.ID)

	}

}

func (o *Order) GetItemsID() []uuid.UUID {

	ids := make([]uuid.UUID, len(o.ItemsOrders))
	for i := range o.ItemsOrders {
		ids[i] = o.ItemsOrders[i].ItemID
	}

	return ids

}

func (o *Order) GetFlavorsID() []uuid.UUID {

	idSet := make(map[uuid.UUID]struct{})
	for i := range o.ItemsOrders {
		var chosen map[string]int
		if err := json.Unmarshal(o.ItemsOrders[i].Flavors, &chosen); err != nil {
			continue
		}
		for idStr := range chosen {
			if id, err := uuid.Parse(idStr); err == nil {
				idSet[id] = struct{}{}
			}
		}
	}

	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	return ids

}

func (o *Order) SetFlavorsName(flavors []Flavor) {
	for i := range o.ItemsOrders {
		o.ItemsOrders[i].SetFlavorsDetails(flavors)
	}
}

func (o *Order) SetItemName(items models.Items) {

	nameMap := make(map[uuid.UUID]string, len(items))
	for _, item := range items {
		nameMap[item.ID] = item.Name
	}

	for i := range o.ItemsOrders {
		if name, ok := nameMap[o.ItemsOrders[i].ItemID]; ok {
			o.ItemsOrders[i].Name = name
		}
	}
}

func (o *Order) SetItemImage(items models.Items) {

	imageMap := make(map[uuid.UUID]string, len(items))
	for _, item := range items {
		imageMap[item.ID] = item.Image
	}

	for i := range o.ItemsOrders {
		if image, ok := imageMap[o.ItemsOrders[i].ItemID]; ok {
			o.ItemsOrders[i].Image = image
		}
	}
}

func (o *Order) SetPrices(items models.Items) {

	priceMap := make(map[uuid.UUID]float64, len(items))
	for _, item := range items {
		// ShopPrice ya refleja el descuento propio del negocio (si tiene uno
		// activo), así el subtotal de la orden nace con ese descuento
		// aplicado y todo lo demás (promo día 1, comisión 0%) opera sobre él.
		priceMap[item.ID] = item.ShopPrice()
	}

	var total float64
	for i := range o.ItemsOrders {
		if price, ok := priceMap[o.ItemsOrders[i].ItemID]; ok {
			o.ItemsOrders[i].UnitPrice = price
			o.ItemsOrders[i].TotalPrice = money.RoundUSD(price * float64(o.ItemsOrders[i].Amount))
			total += o.ItemsOrders[i].TotalPrice
		}
	}

	o.TotalPrice = money.RoundUSD(total) + o.DeliveryFee
}

// ApplyLaunchPromo, si active, subsidia el 50% del subtotal (items +
// adiciones, sin delivery_fee) del pedido: el cliente paga la mitad y la
// plataforma cubre la otra mitad. Debe llamarse después de SetPrices y
// SetAdditionalsPrices, cuando TotalPrice ya incluye items + adiciones + delivery_fee.
func (o *Order) ApplyLaunchPromo(active bool) {
	if !active {
		return
	}
	subtotal := o.TotalPrice - o.DeliveryFee
	o.PromoSubsidyAmount = subtotal * 0.5
	o.IsLaunchPromo = true
	o.TotalPrice -= o.PromoSubsidyAmount
}

// ApplyZeroCommission marca el pedido para que, al liquidar, el negocio
// reciba el 100% del subtotal en vez del 90% habitual. No cambia el precio
// que paga el cliente, solo la comisión que retiene la plataforma.
func (o *Order) ApplyZeroCommission(active bool) {
	o.IsZeroCommission = active
}

func (o *Order) SetAdditionalsName(additions []Addition) {

	nameMap := make(map[uuid.UUID]string, len(additions))
	for _, addition := range additions {
		nameMap[addition.ID] = addition.Name
	}

	for i := range o.ItemsOrders {
		for j := range o.ItemsOrders[i].Additions {
			if name, ok := nameMap[o.ItemsOrders[i].Additions[j].AdditionID]; ok {
				o.ItemsOrders[i].Additions[j].Name = name
			}
		}
	}

}

func (o *Order) ApplyBsPrice(rate float64) {
	o.DeliveryFeeBs = money.RoundBs(o.DeliveryFee * rate)
	o.TotalPriceBs = money.RoundBs(o.TotalPrice * rate)
}

func (o *Order) SetAdditionalsPrices(addittions []Addition) {

	priceMap := make(map[uuid.UUID]float64, len(addittions))
	for _, addition := range addittions {
		priceMap[addition.ID] = addition.Price
	}

	var totalAdd float64
	for i := range o.ItemsOrders {

		var subTotalAdd float64

		for j := range o.ItemsOrders[i].Additions {

			if price, ok := priceMap[o.ItemsOrders[i].Additions[j].AdditionID]; ok {

				o.ItemsOrders[i].Additions[j].UnitPrice = price

				o.ItemsOrders[i].Additions[j].TotalPrice = money.RoundUSD(price * float64(o.ItemsOrders[i].Amount))

				subTotalAdd += o.ItemsOrders[i].Additions[j].TotalPrice

			}
		}

		o.ItemsOrders[i].TotalPrice += subTotalAdd
		totalAdd += subTotalAdd

	}
	o.TotalPrice += money.RoundUSD(totalAdd)
}
