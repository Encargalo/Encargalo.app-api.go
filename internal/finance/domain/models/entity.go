package models

const (
	EntityTypeRider = "rider"
	EntityTypeShop  = "shop"
)

// NativeCurrency devuelve la moneda del ledger para un entity_type.
func NativeCurrency(entityType string) string {
	if entityType == EntityTypeShop {
		return CurrencyUSD
	}
	return CurrencyVES
}
