package googlemaps

// googleAddressComponent is the wire shape of a single Google address_components
// entry. Kept unexported so callers consume the mapped AddressComponent.
type googleAddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

// AddressComponent is one Google address_components entry, passed through so
// the client can assemble the address however it needs.
type AddressComponent struct {
	LongName  string
	ShortName string
	Types     []string
}

// AddressParts is the same Google data flattened by type. Names match Google's
// type identifiers (locality = city, administrative_area_level_1 = state, etc.).
type AddressParts struct {
	StreetNumber             string
	Route                    string
	Neighborhood             string
	Sublocality              string
	Locality                 string
	AdministrativeAreaLevel3 string
	AdministrativeAreaLevel2 string
	AdministrativeAreaLevel1 string
	Country                  string
	CountryCode              string
	PostalCode               string
	Premise                  string
	Subpremise               string
}

func parseAddressComponents(raw []googleAddressComponent) ([]AddressComponent, AddressParts) {
	components := make([]AddressComponent, 0, len(raw))
	var parts AddressParts

	for _, c := range raw {
		components = append(components, AddressComponent(c))
		applyComponentToParts(&parts, c)
	}

	return components, parts
}

func applyComponentToParts(parts *AddressParts, c googleAddressComponent) {
	for _, t := range c.Types {
		switch t {
		case "street_number":
			parts.StreetNumber = c.LongName
		case "route":
			parts.Route = c.LongName
		case "neighborhood":
			parts.Neighborhood = c.LongName
		case "sublocality", "sublocality_level_1":
			if parts.Sublocality == "" {
				parts.Sublocality = c.LongName
			}
		case "locality":
			parts.Locality = c.LongName
		case "administrative_area_level_3":
			parts.AdministrativeAreaLevel3 = c.LongName
		case "administrative_area_level_2":
			parts.AdministrativeAreaLevel2 = c.LongName
		case "administrative_area_level_1":
			parts.AdministrativeAreaLevel1 = c.LongName
		case "country":
			parts.Country = c.LongName
			parts.CountryCode = c.ShortName
		case "postal_code":
			parts.PostalCode = c.LongName
		case "premise":
			parts.Premise = c.LongName
		case "subpremise":
			parts.Subpremise = c.LongName
		}
	}
}
