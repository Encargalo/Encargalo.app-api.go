package shops

import "github.com/google/uuid"

type SearchShopRow struct {
	ID              uuid.UUID `bun:"id"`
	Name            string    `bun:"name"`
	Tag             string    `bun:"tag"`
	Address         string    `bun:"address"`
	Latitude        *float64  `bun:"latitude"`
	Longitude       *float64  `bun:"longitude"`
	HomePhone       string    `bun:"home_phone"`
	LogoImage       string    `bun:"logo_image"`
	Banner          string    `bun:"banner"`
	Opened          bool      `bun:"opened"`
	Type            string    `bun:"type"`
	Score           float64   `bun:"score"`
	Coverage        float32   `bun:"coverage"`
	MinimumDelivery int       `bun:"minimum_delivery"`
	MaximumDelivery int       `bun:"maximum_delivery"`
	MatchedByName   bool      `bun:"matched_by_name"`
}
