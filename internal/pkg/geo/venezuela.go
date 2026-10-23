package geo

// Caja envolvente de Venezuela (continental + insular) con un margen que no recorta costa ni frontera.
const (
	venezuelaMinLat = 0.5
	venezuelaMaxLat = 12.3
	venezuelaMinLng = -73.5
	venezuelaMaxLng = -59.7
)

// InVenezuela indica si la coordenada cae dentro de la caja envolvente del país.
// Es una caja, no la frontera real: puede dar true en un punto fronterizo de
// Colombia, Brasil o Guyana, pero nunca descarta un punto venezolano.
func InVenezuela(lat, lng float64) bool {
	return lat >= venezuelaMinLat && lat <= venezuelaMaxLat &&
		lng >= venezuelaMinLng && lng <= venezuelaMaxLng
}
