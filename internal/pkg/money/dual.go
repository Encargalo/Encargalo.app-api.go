package money

const (
	CurrencyVES = "VES"
	CurrencyUSD = "USD"
)

// NormalizeCurrency devuelve defaultCurrency si currency está vacío.
func NormalizeCurrency(currency, defaultCurrency string) string {
	if currency == "" {
		return defaultCurrency
	}
	return currency
}

// BalanceDual convierte un saldo en su moneda nativa a Bs y USD usando la tasa
// BCV vigente (informativa para el total actual del balance).
func BalanceDual(amount float64, currency string, spotRate float64) (balanceBs, balanceUSD float64) {
	switch NormalizeCurrency(currency, CurrencyVES) {
	case CurrencyUSD:
		balanceUSD = RoundUSD(amount)
		if spotRate > 0 {
			balanceBs = RoundBs(amount * spotRate)
		}
	default:
		balanceBs = RoundBs(amount)
		if spotRate > 0 {
			balanceUSD = RoundUSD(amount / spotRate)
		}
	}
	return balanceBs, balanceUSD
}

// TransactionDual convierte un movimiento a Bs y USD. Si el movimiento trae
// bcvRate congelado, se usa para la conversión cruzada; si no, spotRate.
func TransactionDual(amount float64, currency string, bcvRate *float64, spotRate float64) (amountBs, amountUSD float64) {
	rate := spotRate
	if bcvRate != nil && *bcvRate > 0 {
		rate = *bcvRate
	}

	switch NormalizeCurrency(currency, CurrencyVES) {
	case CurrencyUSD:
		amountUSD = RoundUSD(amount)
		if rate > 0 {
			amountBs = RoundBs(amount * rate)
		}
	default:
		amountBs = RoundBs(amount)
		if rate > 0 {
			amountUSD = RoundUSD(amount / rate)
		}
	}
	return amountBs, amountUSD
}
