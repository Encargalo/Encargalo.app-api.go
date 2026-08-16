package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/bizclock"
	"Encargalo.app-api.go/internal/users/domain/ports"
)

// accountDeletionWorkerRunHour es la hora (0-23, hora de negocio
// America/Caracas) en la que corre el barrido diario.
const accountDeletionWorkerRunHour = 0

// AccountDeletionWorker finaliza (soft delete), una vez al día a
// medianoche, todas las cuentas cuya solicitud de eliminación venció el
// período de gracia de 30 días sin que el usuario volviera a iniciar sesión.
type AccountDeletionWorker struct {
	users ports.UsersApp
}

func NewAccountDeletionWorker(users ports.UsersApp) *AccountDeletionWorker {
	return &AccountDeletionWorker{users: users}
}

func (w *AccountDeletionWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *AccountDeletionWorker) loop(ctx context.Context) {
	for {
		timer := time.NewTimer(durationUntilNextRun())
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			w.tick(ctx)
		}
	}
}

// durationUntilNextRun calcula cuánto falta para la próxima medianoche en
// hora de negocio (America/Caracas). Venezuela no tiene horario de verano,
// así que "próxima medianoche" es siempre exacto, sin drift por DST.
func durationUntilNextRun() time.Duration {
	now := bizclock.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), accountDeletionWorkerRunHour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}

func (w *AccountDeletionWorker) tick(ctx context.Context) {
	n, err := w.users.FinalizePendingDeletions(ctx)
	if err != nil {
		slog.Error("AccountDeletionWorker tick failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("AccountDeletionWorker: cuentas finalizadas por vencimiento de 30 días", "count", n)
	}
}
