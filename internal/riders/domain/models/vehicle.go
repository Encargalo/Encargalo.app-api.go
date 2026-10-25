package models

import (
	"strings"

	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
)

// NormalizeVehicleType canoniza el valor crudo del perfil: el upsert de staff no
// valida el campo, así que "Bici" y " bici " son la misma bici.
func NormalizeVehicleType(vehicleType string) string {
	return strings.ToLower(strings.TrimSpace(vehicleType))
}

func IsBike(vehicleType string) bool {
	return NormalizeVehicleType(vehicleType) == VehicleTypeBike
}

// VehicleCanCarryShopType: la bici solo lleva restaurante — Turbo y supermercado son volumen y peso que no le entran.
// Un vehículo sin cargar o un tipo de shop desconocido no excluyen: hay riders legacy sin el campo.
func VehicleCanCarryShopType(vehicleType, shopType string) bool {
	if !IsBike(vehicleType) {
		return true
	}
	return shopType != shopsmodels.ShopTypeTurbo && shopType != shopsmodels.ShopTypeSupermarket
}
