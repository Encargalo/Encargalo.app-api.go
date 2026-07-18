package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"github.com/google/uuid"
)

// CartFingerprint is a stable hash of shop + address + items so CreateQuote
// can reuse an active quote when the customer re-enters the same checkout.
func CartFingerprint(shopID uuid.UUID, address dtos.Address, items []dtos.CreateItemsOrder) string {
	type addRef struct {
		AdditionID string `json:"addition_id"`
	}
	type itemRef struct {
		ItemID      string         `json:"item_id"`
		Amount      int            `json:"amount"`
		Observation string         `json:"observation"`
		Flavors     map[string]int `json:"flavors"`
		Additions   []addRef       `json:"additions"`
	}
	payload := struct {
		ShopID       string    `json:"shop_id"`
		Address      string    `json:"address"`
		Latitude     float64   `json:"latitude"`
		Longitude    float64   `json:"longitude"`
		Instructions string    `json:"instructions"`
		Items        []itemRef `json:"items"`
	}{
		ShopID:       shopID.String(),
		Address:      address.Address,
		Latitude:     address.Latitude,
		Longitude:    address.Longitude,
		Instructions: address.Instructions,
		Items:        make([]itemRef, 0, len(items)),
	}

	sorted := append([]dtos.CreateItemsOrder(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ItemID.String() < sorted[j].ItemID.String()
	})

	for _, it := range sorted {
		flavors := it.Flavors
		if flavors == nil {
			flavors = map[string]int{}
		}
		adds := make([]addRef, 0, len(it.Additions))
		for _, a := range it.Additions {
			adds = append(adds, addRef{AdditionID: a.AdditionID.String()})
		}
		sort.Slice(adds, func(i, j int) bool { return adds[i].AdditionID < adds[j].AdditionID })
		payload.Items = append(payload.Items, itemRef{
			ItemID:      it.ItemID.String(),
			Amount:      it.Amount,
			Observation: it.Observation,
			Flavors:     flavors,
			Additions:   adds,
		})
	}

	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
