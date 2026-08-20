package app

import (
	"context"
	"encoding/json"
	"maps"
	"regexp"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/tracking/domain/dtos"
	"Encargalo.app-api.go/internal/tracking/domain/models"
	"Encargalo.app-api.go/internal/tracking/domain/ports"
	"github.com/google/uuid"
)

const maxParamsBytes = 4096

var typePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type trackingApp struct {
	repo ports.TrackingRepo
}

func New(repo ports.TrackingRepo) ports.TrackingApp {
	return &trackingApp{repo: repo}
}

func (a *trackingApp) Track(ctx context.Context, userID, sessionID uuid.UUID, event dtos.TrackEvent) error {
	eventType := strings.TrimSpace(event.Type)
	if !typePattern.MatchString(eventType) {
		return errcustom.ErrInvalidTrackingType
	}

	params := cloneParams(event.Params)
	params["user_id"] = userID.String()
	params["session_id"] = sessionID.String()

	raw, err := json.Marshal(params)
	if err != nil {
		return errcustom.Unexpected(err)
	}
	if len(raw) > maxParamsBytes {
		return errcustom.ErrTrackingParamsTooLarge
	}

	return a.repo.Insert(ctx, models.Event{
		ID:        event.ID,
		Type:      eventType,
		Params:    params,
		CreatedAt: time.Now().UTC(),
	})
}

func cloneParams(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+2)
	maps.Copy(out, in)
	return out
}
