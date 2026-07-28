package shops

const (
	ShopTypeRestaurant  = "Restaurant"
	ShopTypeSupermarket = "Supermarket"
	ShopTypeTurbo       = "Turbo"
)

func IsValidShopType(value string) bool {
	return value == ShopTypeRestaurant || value == ShopTypeSupermarket || value == ShopTypeTurbo
}
