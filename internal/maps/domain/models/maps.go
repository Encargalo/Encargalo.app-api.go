package models

// ReverseGeocodeResult is the resolved address for a lat/lng pair.
type ReverseGeocodeResult struct {
	FormattedAddress string `json:"formatted_address"`
	ShortAddress     string `json:"short_address"`
}

// PlacePrediction is one autocomplete suggestion.
type PlacePrediction struct {
	PlaceID       string `json:"place_id"`
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`
}

// PlaceDetails is the resolved address/coordinates for a place_id.
type PlaceDetails struct {
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}
