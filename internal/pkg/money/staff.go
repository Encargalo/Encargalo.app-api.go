package money

import (
	"Encargalo.app-api.go/internal/shared/errcustom"
)

// NativeDeltaFromStaffBs convierte un monto ingresado por staff en bolívares al
// delta del ledger en la moneda nativa de la entidad. isDebit=true resta saldo.
func NativeDeltaFromStaffBs(amountBs float64, nativeCurrency string, bcvRate float64, isDebit bool) (delta float64, frozenRate *float64) {
	rate := bcvRate
	frozenRate = &rate

	var native float64
	switch NormalizeCurrency(nativeCurrency, CurrencyVES) {
	case CurrencyUSD:
		native = RoundUSD(amountBs / bcvRate)
	default:
		native = RoundBs(amountBs)
	}
	if isDebit {
		native = -native
	}
	return native, frozenRate
}

// EnsureDebitAllowed valida que un débito no deje el balance nativo negativo.
func EnsureDebitAllowed(balance, delta float64, nativeCurrency string) error {
	if delta >= 0 {
		return nil
	}
	remaining := balance + delta
	switch NormalizeCurrency(nativeCurrency, CurrencyVES) {
	case CurrencyUSD:
		if RoundUSD(remaining) < 0 {
			return errcustom.ErrInsufficientBalance
		}
	default:
		if RoundBs(remaining) < 0 {
			return errcustom.ErrInsufficientBalance
		}
	}
	return nil
}
