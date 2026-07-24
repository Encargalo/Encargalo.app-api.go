package models

import "github.com/google/uuid"

// Flavor is the read model orders uses to resolve flavor names chosen on an item.
type Flavor struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Available *bool     `json:"available,omitempty"`
}

func (f Flavor) IsAvailable() bool {
	if f.Available == nil {
		return true
	}
	return *f.Available
}
