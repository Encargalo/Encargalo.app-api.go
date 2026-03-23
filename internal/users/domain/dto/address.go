package dto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type Addresses []Address

var addressTypes = map[string]func() any{
	"Apartment":   func() any { return &Apartment{} },
	"Condominium": func() any { return &Condominium{} },
	"Home":        func() any { return &Home{} },
}

type Address struct {
	Type         string  `json:"type" validate:"required" example:"Casa"`
	Label        string  `json:"label" validate:"required" example:"Oficina"`
	Address      string  `json:"address" validate:"required" example:"Calle 123 # 45-67"`
	Latitude     float64 `json:"lat" validate:"required,latitude" example:"4.609710"`
	Longitude    float64 `json:"long" validate:"required,longitude" example:"-74.081750"`
	Instructions string  `json:"instructions" validate:"omitempty" example:"Tocar el timbre azul"`
	Extra        any     `json:"extra" validate:"required"`
}

type Apartment struct {
	Building  string `json:"building" example:"Edificio Central"`
	Apartment string `json:"apartment" validate:"required" example:"Apto 101, Torre B"`
}

type Condominium struct {
	Name   string `json:"name" validate:"required" example:"Condominio Los Pinos"`
	Number string `json:"number" validate:"required" example:"Casa 12, Sector A"`
}

type Home struct {
	Details string `json:"details" example:"Casa de dos pisos con jardin"`
}

func (a *Address) UnmarshalJSON(data []byte) error {

	type Alias Address
	aux := &struct {
		Extra json.RawMessage `json:"extra"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type == "" {
		return errors.New("missing field: type")
	}

	constructor, ok := addressTypes[aux.Type]
	if !ok {
		return fmt.Errorf("unknown type: %s", aux.Type)
	}

	a.Extra = constructor()

	if len(aux.Extra) > 0 {
		if err := json.Unmarshal(aux.Extra, a.Extra); err != nil {
			return fmt.Errorf("invalid extra data: %w", err)
		}
	}

	return nil
}

func (a *Address) BuildExtra() {
	constructor, ok := addressTypes[a.Type]
	if !ok {
		a.Extra = nil
		return
	}
	a.Extra = constructor()
}

func (a *Address) Validate() error {
	_ = conform.Struct(context.Background(), a)

	if err := validate.Struct(a); err != nil {
		return err
	}

	if a.Extra != nil {
		if err := validate.Struct(a.Extra); err != nil {
			return err
		}
	}

	return nil
}

func (c *Addresses) Add(address Address) error {
	address.BuildExtra()

	if err := address.Validate(); err != nil {
		return err
	}

	*c = append(*c, address)
	return nil
}
