package models

import (
	"Encargalo.app-api.go/internal/pkg/id"
	"encoding/json"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"github.com/google/uuid"
)

const (
	// QuotePaymentTTL: ventana para transferir / usar balance.
	QuotePaymentTTL = 30 * time.Minute
	// QuoteProofGrace: extra tras expires_at si el create trae comprobante.
	QuoteProofGrace = 15 * time.Minute
)

// OrderQuote is the server-frozen checkout snapshot stored in Redis.
type OrderQuote struct {
	QuoteID        uuid.UUID               `json:"quote_id"`
	CustomerID     uuid.UUID               `json:"customer_id"`
	ShopID         uuid.UUID               `json:"shop_id"`
	Address        dtos.Address            `json:"address"`
	Items          []dtos.CreateItemsOrder `json:"items"`
	TotalUSD       float64                 `json:"total_usd"`
	TotalBs        float64                 `json:"total_bs"`
	DeliveryFeeUSD float64                 `json:"delivery_fee_usd"`
	DeliveryFeeBs  float64                 `json:"delivery_fee_bs"`
	// DeliverySubsidyUSD es el envío gratis del primer pedido; DeliveryFeeUSD ya lo trae restado.
	DeliverySubsidyUSD float64             `json:"delivery_subsidy_usd,omitempty"`
	BcvRate            float64             `json:"bcv_rate"`
	IsLaunchPromo      bool                `json:"is_launch_promo"`
	PromoSubsidyAmount float64             `json:"promo_subsidy_amount"`
	IsZeroCommission   bool                `json:"is_zero_commission"`
	Lines              []QuoteLineSnapshot `json:"lines"`
	CartFingerprint    string              `json:"cart_fingerprint"`
	CreatedAt          time.Time           `json:"created_at"`
	ExpiresAt          time.Time           `json:"expires_at"`
	ProofGraceUntil    time.Time           `json:"proof_grace_until"`
}

type QuoteLineSnapshot struct {
	ItemID         uuid.UUID               `json:"item_id"`
	Name           string                  `json:"name"`
	Image          string                  `json:"image"`
	Amount         int                     `json:"amount"`
	UnitPrice      float64                 `json:"unit_price"`
	TotalPrice     float64                 `json:"total_price"`
	Observation    string                  `json:"observation"`
	Flavors        json.RawMessage         `json:"flavors"`
	FlavorsDetails []ItemFlavor            `json:"flavors_details,omitempty"`
	Additions      []QuoteAdditionSnapshot `json:"additions"`
}

type QuoteAdditionSnapshot struct {
	AdditionID uuid.UUID `json:"addition_id"`
	Name       string    `json:"name"`
	UnitPrice  float64   `json:"unit_price"`
	TotalPrice float64   `json:"total_price"`
}

// AcceptableForCreate: vigente, o en gracia si hay comprobante.
func (q *OrderQuote) AcceptableForCreate(now time.Time, hasProof bool) bool {
	if now.Before(q.ExpiresAt) || now.Equal(q.ExpiresAt) {
		return true
	}
	return hasProof && (now.Before(q.ProofGraceUntil) || now.Equal(q.ProofGraceUntil))
}

func (q *OrderQuote) RedisTTL(now time.Time) time.Duration {
	until := q.ProofGraceUntil
	if until.IsZero() {
		until = q.ExpiresAt
	}
	ttl := until.Sub(now)
	if ttl < time.Minute {
		return time.Minute
	}
	return ttl
}

func SnapshotLinesFromOrder(order *Order) []QuoteLineSnapshot {
	out := make([]QuoteLineSnapshot, 0, len(order.ItemsOrders))
	for i := range order.ItemsOrders {
		line := order.ItemsOrders[i]
		adds := make([]QuoteAdditionSnapshot, 0, len(line.Additions))
		for j := range line.Additions {
			adds = append(adds, QuoteAdditionSnapshot{
				AdditionID: line.Additions[j].AdditionID,
				Name:       line.Additions[j].Name,
				UnitPrice:  line.Additions[j].UnitPrice,
				TotalPrice: line.Additions[j].TotalPrice,
			})
		}
		var flavors json.RawMessage
		if len(line.Flavors) > 0 {
			flavors = append(json.RawMessage(nil), line.Flavors...)
		}
		out = append(out, QuoteLineSnapshot{
			ItemID:         line.ItemID,
			Name:           line.Name,
			Image:          line.Image,
			Amount:         line.Amount,
			UnitPrice:      line.UnitPrice,
			TotalPrice:     line.TotalPrice,
			Observation:    line.Observation,
			Flavors:        flavors,
			FlavorsDetails: append([]ItemFlavor(nil), line.FlavorsDetails...),
			Additions:      adds,
		})
	}
	return out
}

func ApplyQuoteSnapshot(order *Order, quote *OrderQuote) {
	order.QuoteID = &quote.QuoteID
	order.ShopID = quote.ShopID
	order.Address = quote.Address.Address
	order.Latitude = quote.Address.Latitude
	order.Longitude = quote.Address.Longitude
	order.DeliveryFee = quote.DeliveryFeeUSD
	order.DeliverySubsidyUSD = quote.DeliverySubsidyUSD
	order.TotalPrice = quote.TotalUSD
	order.IsLaunchPromo = quote.IsLaunchPromo
	order.PromoSubsidyAmount = quote.PromoSubsidyAmount
	order.IsZeroCommission = quote.IsZeroCommission

	order.ItemsOrders = make([]ItemsOrder, len(quote.Lines))
	for i := range quote.Lines {
		line := quote.Lines[i]
		itemID := id.New()
		order.ItemsOrders[i] = ItemsOrder{
			ID:             itemID,
			OrderID:        order.ID,
			ItemID:         line.ItemID,
			Name:           line.Name,
			Image:          line.Image,
			Amount:         line.Amount,
			UnitPrice:      line.UnitPrice,
			TotalPrice:     line.TotalPrice,
			Observation:    line.Observation,
			Flavors:        line.Flavors,
			FlavorsDetails: append([]ItemFlavor(nil), line.FlavorsDetails...),
		}
		if len(line.Additions) > 0 {
			order.ItemsOrders[i].Additions = make([]AdditionsOrder, len(line.Additions))
			for j := range line.Additions {
				order.ItemsOrders[i].Additions[j] = AdditionsOrder{
					ID:          id.New(),
					OrderItemID: itemID,
					AdditionID:  line.Additions[j].AdditionID,
					Name:        line.Additions[j].Name,
					Amount:      line.Amount,
					UnitPrice:   line.Additions[j].UnitPrice,
					TotalPrice:  line.Additions[j].TotalPrice,
				}
			}
		}
	}
}
