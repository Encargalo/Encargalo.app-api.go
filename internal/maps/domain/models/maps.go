package models

// AddressComponent is one Google address_components entry, passed through so
// the client can assemble the address however it needs.
type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

// AddressParts is the same Google data flattened by type. Names match Google's
// type identifiers: locality = city, administrative_area_level_1 = state,
// administrative_area_level_2 = municipality/province.
type AddressParts struct {
	StreetNumber             string `json:"street_number,omitempty"`
	Route                    string `json:"route,omitempty"`
	Neighborhood             string `json:"neighborhood,omitempty"`
	Sublocality              string `json:"sublocality,omitempty"`
	Locality                 string `json:"locality,omitempty"`
	AdministrativeAreaLevel3 string `json:"administrative_area_level_3,omitempty"`
	AdministrativeAreaLevel2 string `json:"administrative_area_level_2,omitempty"`
	AdministrativeAreaLevel1 string `json:"administrative_area_level_1,omitempty"`
	Country                  string `json:"country,omitempty"`
	CountryCode              string `json:"country_code,omitempty"`
	PostalCode               string `json:"postal_code,omitempty"`
	Premise                  string `json:"premise,omitempty"`
	Subpremise               string `json:"subpremise,omitempty"`
}

// ReverseGeocodeResult is the resolved address for a lat/lng pair. Top-level
// fields mirror results[0]; results carries every candidate Google returned.
type ReverseGeocodeResult struct {
	FormattedAddress  string             `json:"formatted_address"`
	ShortAddress      string             `json:"short_address"`
	Parts             AddressParts       `json:"parts"`
	AddressComponents []AddressComponent `json:"address_components"`
	Results           []GeocodeCandidate `json:"results"`
	// NearestPlace es el lugar con nombre más cercano a la coordenada (ej. una
	// estación o urbanización), buscado aparte porque la Geocoding API no
	// conoce POIs: solo direcciones. Nulo si no hay ninguno dentro del radio.
	NearestPlace *NearestPlace `json:"nearest_place,omitempty"`
}

// NearestPlace es el lugar con nombre más cercano a un punto, usado por el
// cliente para anteponer un nombre reconocible (ej. "Estación Charallave Sur")
// cuando el reverse geocode por sí solo solo trae calle/urbanización.
type NearestPlace struct {
	Name           string `json:"name"`
	Category       string `json:"category"`
	DistanceMeters int    `json:"distance_meters"`
}

// GeocodeCandidate is one entry of Google's reverse geocoding results.
type GeocodeCandidate struct {
	FormattedAddress  string             `json:"formatted_address"`
	PlaceID           string             `json:"place_id"`
	Types             []string           `json:"types"`
	Lat               float64            `json:"lat"`
	Lng               float64            `json:"lng"`
	LocationType      string             `json:"location_type"`
	Parts             AddressParts       `json:"parts"`
	AddressComponents []AddressComponent `json:"address_components"`
}

// PlacePrediction is one autocomplete suggestion.
type PlacePrediction struct {
	PlaceID       string `json:"place_id"`
	MainText      string `json:"main_text"`
	SecondaryText string `json:"secondary_text"`
}

// PlaceDetails is the resolved address/coordinates for a place_id.
type PlaceDetails struct {
	Address           string             `json:"address"`
	Lat               float64            `json:"lat"`
	Lng               float64            `json:"lng"`
	Parts             AddressParts       `json:"parts"`
	AddressComponents []AddressComponent `json:"address_components"`
}

// Route is the driving route between two points; polyline uses Google's
// encoded polyline format.
type Route struct {
	Polyline       string `json:"polyline"`
	DistanceMeters int64  `json:"distance_meters"`
	DurationMin    int64  `json:"duration_min"`
}

type PopularPlace struct {
	PlaceID    string  `json:"place_id"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Address    string  `json:"address"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	DistanceKm float64 `json:"distance_km"`
}
