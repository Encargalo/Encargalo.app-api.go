package models_test

import (
	"testing"

	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"github.com/stretchr/testify/assert"
)

func TestRiderApplicationRequest_Validate(t *testing.T) {
	cases := map[string]struct {
		req   dtos.RiderApplicationRequest
		valid bool
	}{
		"vacía":         {dtos.RiderApplicationRequest{}, true},
		"moto":          {dtos.RiderApplicationRequest{VehicleType: new("moto")}, true},
		"bici":          {dtos.RiderApplicationRequest{VehicleType: new("bici")}, true},
		"carro":         {dtos.RiderApplicationRequest{VehicleType: new("carro")}, true},
		"tipo vacío":    {dtos.RiderApplicationRequest{VehicleType: new("")}, false},
		"tipo inválido": {dtos.RiderApplicationRequest{VehicleType: new("camion")}, false},
		"color largo":   {dtos.RiderApplicationRequest{VehicleColor: new("azul eléctrico metalizado con franjas")}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.valid {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}
