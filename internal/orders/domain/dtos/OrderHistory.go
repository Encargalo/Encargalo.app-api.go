package dtos

import (
	"time"

	"github.com/google/uuid"
)

// OrderHistoryItem: los primeros 9 campos son el contrato original; el resto
// es aditivo. Los datos del cliente y el teléfono del local no se exponen.
type OrderHistoryItem struct {
	ID            uuid.UUID `json:"id"`
	Number        int       `json:"number"`
	Status        string    `json:"status"`
	TotalPrice    float64   `json:"total_price"`
	DeliveryFee   float64   `json:"delivery_fee"`
	MethodPayment string    `json:"method_payment"`
	Address       string    `json:"address"`
	ShopID        uuid.UUID `json:"shop_id"`
	CreatedAt     time.Time `json:"created_at"`

	ShopName            string  `json:"shop_name"`
	ShopLogo            string  `json:"shop_logo"`
	ShopLatitude        float64 `json:"shop_latitude"`
	ShopLongitude       float64 `json:"shop_longitude"`
	AddressInstructions string  `json:"address_instructions"`
	IsLaunchPromo       bool    `json:"is_launch_promo"`
	// TotalPriceBs y DeliveryFeeBs son null en pedidos sin tasa BCV congelada.
	TotalPriceBs  *float64 `json:"total_price_bs"`
	DeliveryFeeBs *float64 `json:"delivery_fee_bs"`
	// ProductsCount = suma de amount de items + suma de amount de adiciones.
	ProductsCount int                   `json:"products_count"`
	Items         []OrderHistoryProduct `json:"items"`
	// StatusHistory es null en pedidos anteriores al registro de transiciones.
	StatusHistory []OrderStatusChange  `json:"status_history"`
	Payment       *OrderHistoryPayment `json:"payment"`
}

type OrderHistoryProduct struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Amount       int                    `json:"amount"`
	UnitPrice    float64                `json:"unit_price"`
	TotalPrice   float64                `json:"total_price"`
	UnitPriceBs  float64                `json:"unit_price_bs"`
	TotalPriceBs float64                `json:"total_price_bs"`
	Observation  string                 `json:"observation"`
	Flavors      []OrderHistoryFlavor   `json:"flavors"`
	Additions    []OrderHistoryAddition `json:"additions"`
}

type OrderHistoryFlavor struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Amount int       `json:"amount"`
}

type OrderHistoryAddition struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Amount       int       `json:"amount"`
	UnitPrice    float64   `json:"unit_price"`
	TotalPrice   float64   `json:"total_price"`
	UnitPriceBs  float64   `json:"unit_price_bs"`
	TotalPriceBs float64   `json:"total_price_bs"`
}

type OrderStatusChange struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
}

type OrderHistoryPayment struct {
	Status            string   `json:"status"`
	ExpectedAmountBs  float64  `json:"expected_amount_bs"`
	ObservedAmountBs  *float64 `json:"observed_amount_bs"`
	DifferenceBs      *float64 `json:"difference_bs"`
	RejectionReason   *string  `json:"rejection_reason"`
	CreditedBs        *float64 `json:"credited_bs"`
	BalanceAmountBs   float64  `json:"balance_amount_bs"`
	PagoMovilAmountBs float64  `json:"pago_movil_amount_bs"`
	CashAmountBs      float64  `json:"cash_amount_bs"`
	BcvRate           float64  `json:"bcv_rate"`
}

type OrderHistoryResponse []OrderHistoryItem
