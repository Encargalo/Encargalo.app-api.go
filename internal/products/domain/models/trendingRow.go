package models

import "time"

type TrendingRow struct {
	ShopID                string     `bun:"shop_id"`
	ShopName              string     `bun:"shop_name"`
	ShopTag               string     `bun:"shop_tag"`
	ShopLogo              string     `bun:"shop_logo"`
	ShopAddress           string     `bun:"shop_address"`
	ShopScore             float64    `bun:"shop_score"`
	ShopType              string     `bun:"shop_type"`
	ShopLatitude          float64    `bun:"shop_latitude"`
	ShopLongitude         float64    `bun:"shop_longitude"`
	ShopLaunchPromoEndsAt *time.Time `bun:"shop_launch_promo_ends_at"`
	ItemID                string     `bun:"item_id"`
	ItemCategoryID        string     `bun:"item_category_id"`
	ItemName              string     `bun:"item_name"`
	ItemPrice             int        `bun:"item_price"`
	ItemImage             string     `bun:"item_image"`
	ItemDesc              string     `bun:"item_desc"`
	ItemScore             float32    `bun:"item_score"`
	ItemSold              int        `bun:"item_sold"`
}

// HasActiveLaunchPromo indica si, para esta fila, el negocio sigue dentro de
// su ventana de descuento día 1.
func (r TrendingRow) HasActiveLaunchPromo() bool {
	return r.ShopLaunchPromoEndsAt != nil && time.Now().Before(*r.ShopLaunchPromoEndsAt)
}
