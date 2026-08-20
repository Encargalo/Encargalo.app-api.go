package googlemaps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAddressComponents_FlattensGoogleTypes(t *testing.T) {
	raw := []googleAddressComponent{
		{LongName: "123", ShortName: "123", Types: []string{"street_number"}},
		{LongName: "Avenida Francisco de Miranda", ShortName: "Av. Fco. de Miranda", Types: []string{"route"}},
		{LongName: "Los Palos Grandes", ShortName: "Los Palos Grandes", Types: []string{"neighborhood", "political"}},
		{LongName: "Chacao", ShortName: "Chacao", Types: []string{"locality", "political"}},
		{LongName: "Municipio Chacao", ShortName: "Chacao", Types: []string{"administrative_area_level_2", "political"}},
		{LongName: "Miranda", ShortName: "Miranda", Types: []string{"administrative_area_level_1", "political"}},
		{LongName: "Venezuela", ShortName: "VE", Types: []string{"country", "political"}},
		{LongName: "1060", ShortName: "1060", Types: []string{"postal_code"}},
	}

	components, parts := parseAddressComponents(raw)

	require.Len(t, components, 8)
	require.Equal(t, "123", parts.StreetNumber)
	require.Equal(t, "Avenida Francisco de Miranda", parts.Route)
	require.Equal(t, "Los Palos Grandes", parts.Neighborhood)
	require.Equal(t, "Chacao", parts.Locality)
	require.Equal(t, "Municipio Chacao", parts.AdministrativeAreaLevel2)
	require.Equal(t, "Miranda", parts.AdministrativeAreaLevel1)
	require.Equal(t, "Venezuela", parts.Country)
	require.Equal(t, "VE", parts.CountryCode)
	require.Equal(t, "1060", parts.PostalCode)
}

func TestParseAddressComponents_Empty(t *testing.T) {
	components, parts := parseAddressComponents(nil)

	require.Empty(t, components)
	require.Equal(t, AddressParts{}, parts)
}
