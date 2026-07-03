package models

type TrendingRow struct {
	ShopID        string  `bun:"shop_id"`
	ShopName      string  `bun:"shop_name"`
	ShopTag       string  `bun:"shop_tag"`
	ShopLogo      string  `bun:"shop_logo"`
	ShopAddress   string  `bun:"shop_address"`
	ShopScore     float64 `bun:"shop_score"`
	ShopType      string  `bun:"shop_type"`
	ShopLatitude  float64 `bun:"shop_latitude"`
	ShopLongitude float64 `bun:"shop_longitude"`
	ItemID        string  `bun:"item_id"`
	ItemName      string  `bun:"item_name"`
	ItemPrice     int     `bun:"item_price"`
	ItemImage     string  `bun:"item_image"`
	ItemDesc      string  `bun:"item_desc"`
	ItemScore     float32 `bun:"item_score"`
	ItemSold      int     `bun:"item_sold"`
}
