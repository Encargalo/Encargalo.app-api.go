package dtos

import (
	"errors"

	"github.com/google/uuid"
)

type TrackEvent struct {
	ID     uuid.UUID      `json:"id"`
	Type   string         `json:"type"`
	Params map[string]any `json:"params"`
}

func (e TrackEvent) Validate() error {
	if e.ID == uuid.Nil {
		return errors.New("id es requerido")
	}
	if e.Type == "" {
		return errors.New("type es requerido")
	}
	return nil
}
