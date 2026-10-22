package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"Encargalo.app-api.go/internal/notifications/domain/models"
	"Encargalo.app-api.go/internal/shared/config"
)

type Destination interface {
	Name() string
	Send(ctx context.Context, msgs []models.NotificationMessage) error
}

var (
	_ Destination = (*Fanout)(nil)
	_ Destination = (*Discord)(nil)
	_ Destination = (*Slack)(nil)
)

type Fanout struct {
	dests []Destination
}

func NewFanout(cfg config.NotificationsConfig) Destination {
	dests := make([]Destination, 0, 1)
	if cfg.Slack.Configured() {
		dests = append(dests, NewSlack(cfg))
	}
	return &Fanout{dests: dests}
}

func (f *Fanout) Name() string {
	return "fanout"
}

func (f *Fanout) Send(ctx context.Context, msgs []models.NotificationMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	if len(f.dests) == 0 {
		return errors.New("no notification destinations enabled")
	}

	var errs []error
	for _, dest := range f.dests {
		if err := dest.Send(ctx, msgs); err != nil {
			slog.Error("error sending notification", "destination", dest.Name(), "error", err)
			errs = append(errs, fmt.Errorf("%s: %w", dest.Name(), err))
		}
	}
	return errors.Join(errs...)
}
