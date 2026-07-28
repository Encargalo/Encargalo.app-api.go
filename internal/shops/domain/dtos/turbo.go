package dtos

// TurboEntryResponse es la respuesta de GET /turbo/entry: el nodo Turbo
// asignado por geolocalización, o available=false si no hay cobertura.
type TurboEntryResponse struct {
	Available  bool          `json:"available"`
	Shop       *ShopResponse `json:"shop,omitempty"`
	EtaMinutes *int          `json:"eta_minutes,omitempty"`
	CoverageKm float64       `json:"coverage_km,omitempty"`
}
