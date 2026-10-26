package domain_test

import (
	"testing"

	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanTransitionTo_AllowedPaths(t *testing.T) {
	cases := []struct {
		name string
		from financeModels.WithdrawalStatus
		to   financeModels.WithdrawalStatus
	}{
		{"reclamar una pendiente", financeModels.WithdrawalPending, financeModels.WithdrawalProcessing},
		{"rechazar una pendiente", financeModels.WithdrawalPending, financeModels.WithdrawalRejected},
		{"pagar una reclamada", financeModels.WithdrawalProcessing, financeModels.WithdrawalCompleted},
		{"rechazar una reclamada", financeModels.WithdrawalProcessing, financeModels.WithdrawalRejected},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := financeModels.WithdrawalRequest{Status: c.from}
			assert.True(t, req.CanTransitionTo(c.to))
		})
	}
}

func TestCanTransitionTo_ForbiddenPaths(t *testing.T) {
	cases := []struct {
		name string
		from financeModels.WithdrawalStatus
		to   financeModels.WithdrawalStatus
	}{
		// Sin reclamo no hay monto congelado que transferir.
		{"pagar sin reclamar", financeModels.WithdrawalPending, financeModels.WithdrawalCompleted},
		{"reabrir una pagada", financeModels.WithdrawalCompleted, financeModels.WithdrawalProcessing},
		{"reabrir una rechazada", financeModels.WithdrawalRejected, financeModels.WithdrawalProcessing},
		{"pagar una rechazada", financeModels.WithdrawalRejected, financeModels.WithdrawalCompleted},
		{"rechazar una pagada", financeModels.WithdrawalCompleted, financeModels.WithdrawalRejected},
		{"volver a pendiente", financeModels.WithdrawalProcessing, financeModels.WithdrawalPending},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := financeModels.WithdrawalRequest{Status: c.from}
			assert.False(t, req.CanTransitionTo(c.to))
		})
	}
}

func TestIsActive(t *testing.T) {
	assert.True(t, financeModels.WithdrawalRequest{Status: financeModels.WithdrawalPending}.IsActive())
	assert.True(t, financeModels.WithdrawalRequest{Status: financeModels.WithdrawalProcessing}.IsActive())
	assert.False(t, financeModels.WithdrawalRequest{Status: financeModels.WithdrawalCompleted}.IsActive())
	assert.False(t, financeModels.WithdrawalRequest{Status: financeModels.WithdrawalRejected}.IsActive())
}

func TestPayableFrom(t *testing.T) {
	// Sin deuda posterior se paga lo retenido.
	assert.Equal(t, 800.0, financeModels.PayableFrom(800, 0))

	// Ganancias bancarias posteriores no aumentan el retiro: quedan para el
	// siguiente.
	assert.Equal(t, 800.0, financeModels.PayableFrom(800, 150))

	// Un pedido en efectivo deja el disponible negativo y esa deuda se salda
	// contra el propio retiro: 800 retenidos menos 728 que el rider cobró.
	assert.Equal(t, 72.0, financeModels.PayableFrom(800, -728))

	// Si el efectivo se comió el retiro completo no hay nada que transferir.
	assert.Equal(t, -100.0, financeModels.PayableFrom(800, -900))
	assert.Equal(t, 0.0, financeModels.PayableFrom(800, -800))
}

func TestWithdrawalListRow_PayablePrefersFrozenAmount(t *testing.T) {
	frozen := 72.0
	// Una vez reclamada manda el congelado, aunque el saldo se haya movido
	// después: la transferencia ya salió por ese valor.
	claimed := financeModels.WithdrawalListRow{
		WithdrawalRequest: financeModels.WithdrawalRequest{Amount: 800, PayableAmount: &frozen},
		AvailableBalance:  -5000,
	}
	require.Equal(t, 72.0, claimed.Payable())

	// Sin reclamar se recalcula contra el saldo disponible actual.
	pending := financeModels.WithdrawalListRow{
		WithdrawalRequest: financeModels.WithdrawalRequest{Amount: 800},
		AvailableBalance:  -728,
	}
	require.Equal(t, 72.0, pending.Payable())
}
