package id

import "github.com/google/uuid"

// New es el id que asigna el servidor antes de insertar. Es UUID v7 para que
// los índices B-tree reciban las filas nuevas agrupadas por tiempo.
// crypto/rand fallido es fatal, igual que uuid.New.
func New() uuid.UUID {
	v, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return v
}

// Ensure conserva un id que ya vino (el cliente, u otro paso) y solo genera
// uno nuevo cuando el campo sigue vacío. Así el DEFAULT de Postgres no corre.
func Ensure(v uuid.UUID) uuid.UUID {
	if v == uuid.Nil {
		return New()
	}
	return v
}
