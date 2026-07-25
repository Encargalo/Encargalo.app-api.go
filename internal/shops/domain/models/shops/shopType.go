package shops

const (
	ShopTypeRestaurant  = "Restaurant"
	ShopTypeSupermarket = "Supermarket"
)

func IsValidShopType(value string) bool {
	return value == ShopTypeRestaurant || value == ShopTypeSupermarket
}
