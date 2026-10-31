package models_test

import (
	"testing"

	"Encargalo.app-api.go/internal/shops/domain/entity"
	shopsmodels "Encargalo.app-api.go/internal/shops/domain/models/shops"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShopSpecialtyOrGeneral_NilDefaultsToGeneral(t *testing.T) {
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Local e2e", Type: "Supermarket"}

	assert.Equal(t, shopsmodels.ShopSpecialtyGeneral, shop.SpecialtyOrGeneral())
}

func TestShopSpecialtyOrGeneral_ReturnsSetValue(t *testing.T) {
	specialty := shopsmodels.ShopSpecialtyBakery
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Panadería El Trigo", Type: "Supermarket", Specialty: &specialty}

	assert.Equal(t, shopsmodels.ShopSpecialtyBakery, shop.SpecialtyOrGeneral())
}

func TestShopToDomainDTO_IncludesSpecialty(t *testing.T) {
	specialty := shopsmodels.ShopSpecialtyButcher
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Carnicería Don José", Type: "Supermarket", Specialty: &specialty}

	dto := shop.ToDomainDTO(entity.ShopDistance{ShopID: shop.ID})

	assert.Equal(t, shopsmodels.ShopSpecialtyButcher, dto.Specialty)
}

func TestShopToDomainDTO_NilSpecialtyDefaultsToGeneral(t *testing.T) {
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Local e2e", Type: "Supermarket"}

	dto := shop.ToDomainDTO(entity.ShopDistance{ShopID: shop.ID})

	assert.Equal(t, shopsmodels.ShopSpecialtyGeneral, dto.Specialty)
}

func TestShopToStaffDTO_IncludesSpecialty(t *testing.T) {
	specialty := shopsmodels.ShopSpecialtyLiquor
	shop := shopsmodels.Shop{ID: uuid.New(), Name: "Licores La Fuente", Type: "Supermarket", Specialty: &specialty}

	dto := shop.ToStaffDTO()

	assert.Equal(t, shopsmodels.ShopSpecialtyLiquor, dto.Specialty)
}
