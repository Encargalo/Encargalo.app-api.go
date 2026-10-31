package models

// SupermarketCategoryCandidate es una (macro, categoría) distinta disponible
// hoy para el home de supermercado, antes de aplicar la rotación diaria.
type SupermarketCategoryCandidate struct {
	MacroID      string `bun:"macro_id"`
	MacroName    string `bun:"macro_name"`
	MacroSlug    string `bun:"macro_slug"`
	CategoryName string `bun:"category_name"`
}

// SupermarketHomeRow es una fila cruda (shop + categoría + producto) del home
// de supermercado, ya acotada por shop_rank <= N en SQL para que ningún
// negocio domine una repisa.
type SupermarketHomeRow struct {
	ShopID        string  `bun:"shop_id"`
	ShopName      string  `bun:"shop_name"`
	ShopSlug      string  `bun:"shop_slug"`
	ShopLogo      string  `bun:"shop_logo"`
	ShopLatitude  float64 `bun:"shop_latitude"`
	ShopLongitude float64 `bun:"shop_longitude"`
	CategoryID    string  `bun:"category_id"`
	CategoryName  string  `bun:"category_name"`
	MacroID       string  `bun:"macro_id"`
	MacroName     string  `bun:"macro_name"`
	MacroSlug     string  `bun:"macro_slug"`
	ItemID        string  `bun:"item_id"`
	ItemName      string  `bun:"item_name"`
	ItemPrice     float64 `bun:"item_price"`
	ItemImage     string  `bun:"item_image"`
	ItemScore     float32 `bun:"item_score"`
	ItemSold      int     `bun:"item_sold"`
}
