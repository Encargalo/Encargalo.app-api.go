package models

import (
	"Encargalo.app-api.go/internal/products/domain/dtos"
	"github.com/google/uuid"
)

// FlavorEntry representa un sabor embebido en JSONB (categoría o item).
type FlavorEntry struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Available   *bool     `json:"available,omitempty"`
}

func (f FlavorEntry) IsAvailable() bool {
	if f.Available == nil {
		return true
	}
	return *f.Available
}

type FlavorEntries []FlavorEntry

func (f FlavorEntries) ToDomainDTO() dtos.FlavorsResponse {
	resp := make(dtos.FlavorsResponse, len(f))
	for i, entry := range f {
		resp[i] = dtos.FlavorResponse{
			ID:        entry.ID,
			Name:      entry.Name,
			Available: entry.IsAvailable(),
		}
	}
	return resp
}

func (f FlavorEntries) ToCustomerDTO() dtos.FlavorsResponse {
	resp := make(dtos.FlavorsResponse, 0, len(f))
	for _, entry := range f {
		if !entry.IsAvailable() {
			continue
		}
		resp = append(resp, dtos.FlavorResponse{
			ID:        entry.ID,
			Name:      entry.Name,
			Available: true,
		})
	}
	return resp
}

func (f FlavorEntries) CountAvailable() int {
	count := 0
	for _, entry := range f {
		if entry.IsAvailable() {
			count++
		}
	}
	return count
}

func (f FlavorEntries) FindByID(id uuid.UUID) (FlavorEntry, bool) {
	for _, entry := range f {
		if entry.ID == id {
			return entry, true
		}
	}
	return FlavorEntry{}, false
}

func (f FlavorEntries) ContainsID(id uuid.UUID) bool {
	_, ok := f.FindByID(id)
	return ok
}

func (f FlavorEntries) FilterByIDs(ids []uuid.UUID) FlavorEntries {
	if len(ids) == 0 {
		return nil
	}
	wanted := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	selected := make(FlavorEntries, 0, len(ids))
	for _, entry := range f {
		if _, ok := wanted[entry.ID]; ok {
			selected = append(selected, entry)
		}
	}
	return selected
}

func (f FlavorEntries) IDs() []uuid.UUID {
	ids := make([]uuid.UUID, len(f))
	for i, entry := range f {
		ids[i] = entry.ID
	}
	return ids
}

func (f FlavorEntries) IndexByID() map[uuid.UUID]FlavorEntry {
	index := make(map[uuid.UUID]FlavorEntry, len(f))
	for _, entry := range f {
		index[entry.ID] = entry
	}
	return index
}

func (f FlavorEntries) HasName(name string, excludeID uuid.UUID) bool {
	for _, entry := range f {
		if entry.ID == excludeID {
			continue
		}
		if entry.Name == name {
			return true
		}
	}
	return false
}

func (f FlavorEntries) Upsert(entry FlavorEntry) FlavorEntries {
	for i, current := range f {
		if current.ID == entry.ID {
			f[i] = entry
			return f
		}
	}
	return append(f, entry)
}

func (f FlavorEntries) RemoveByID(id uuid.UUID) FlavorEntries {
	filtered := make(FlavorEntries, 0, len(f))
	for _, entry := range f {
		if entry.ID != id {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
