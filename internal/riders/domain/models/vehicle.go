package models

import shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"

// VehicleCanCarryShopType: la bici solo lleva restaurante — Turbo y supermercado son volumen y peso que no le entran.
// Un vehículo sin cargar o un tipo de shop desconocido no excluyen: hay riders legacy sin el campo.
func VehicleCanCarryShopType(vehicleType, shopType string) bool {
	if vehicleType != VehicleTypeBike {
		return true
	}
	return shopType != shopsmodels.ShopTypeTurbo && shopType != shopsmodels.ShopTypeSupermarket
}
