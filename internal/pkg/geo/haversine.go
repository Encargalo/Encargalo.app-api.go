package geo

import "math"

const earthRadiusKm = 6371.0

// HaversineKm calcula la distancia en línea recta (km) entre dos coordenadas.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// BoundingBoxDeltas da el delta de latitud/longitud (en grados) que cubre
// radiusKm alrededor de lat — un pre-filtro barato en SQL antes de la
// distancia exacta por Haversine (sin PostGIS, no evalúa trigonometría fila
// por fila en la base).
func BoundingBoxDeltas(lat, radiusKm float64) (latDelta, lngDelta float64) {
	const kmPerDegreeLat = earthRadiusKm * math.Pi / 180
	latDelta = radiusKm / kmPerDegreeLat
	lngDelta = radiusKm / (kmPerDegreeLat * math.Cos(lat*math.Pi/180))
	return latDelta, lngDelta
}
