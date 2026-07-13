package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
)

// MaxAdditionsSelection es el máximo fijo de adiciones distintas por item.
const MaxAdditionsSelection = 5

// MacroSlugAdiciones identifica la macro categoría cuyos items son el catálogo de adiciones.
const MacroSlugAdiciones = "adiciones"

type AdditionEntry struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Price     int       `json:"price"`
	Available *bool     `json:"available,omitempty"`
}

func (a AdditionEntry) IsAvailable() bool {
	if a.Available == nil {
		return true
	}
	return *a.Available
}

type AdditionEntries []AdditionEntry

func (a AdditionEntries) ToDomainDTO() dtos.AdditionsResponse {
	resp := make(dtos.AdditionsResponse, len(a))
	for i, entry := range a {
		resp[i] = dtos.AdditionResponse{
			ID:        entry.ID,
			Name:      entry.Name,
			Ref:       float64(entry.Price),
			Available: entry.IsAvailable(),
		}
	}
	return resp
}

func (a AdditionEntries) ToCustomerDTO() dtos.AdditionsResponse {
	resp := make(dtos.AdditionsResponse, 0, len(a))
	for _, entry := range a {
		if !entry.IsAvailable() {
			continue
		}
		resp = append(resp, dtos.AdditionResponse{
			ID:        entry.ID,
			Name:      entry.Name,
			Ref:       float64(entry.Price),
			Available: true,
		})
	}
	return resp
}

func (a AdditionEntries) FindByID(id uuid.UUID) (AdditionEntry, bool) {
	for _, entry := range a {
		if entry.ID == id {
			return entry, true
		}
	}
	return AdditionEntry{}, false
}

func (a AdditionEntries) Upsert(entry AdditionEntry) AdditionEntries {
	for i, current := range a {
		if current.ID == entry.ID {
			a[i] = entry
			return a
		}
	}
	return append(a, entry)
}

func (a AdditionEntries) RemoveByID(id uuid.UUID) AdditionEntries {
	filtered := make(AdditionEntries, 0, len(a))
	for _, entry := range a {
		if entry.ID != id {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func AdditionEntryFromItem(item Item) AdditionEntry {
	available := EffectiveItemCanOrder(item.AvailabilityStatus, item.Details)
	return AdditionEntry{
		ID:        item.ID,
		Name:      item.Name,
		Price:     item.Price,
		Available: BoolPtr(available),
	}
}
