package repo

import (
	"context"

	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	tripsDtos "Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
)

type passengerLookup struct {
	profiles ridersPorts.RiderProfileRepo
}

func NewPassengerLookup(profiles ridersPorts.RiderProfileRepo) ridersPorts.PassengerLookup {
	return &passengerLookup{profiles: profiles}
}

// GetByCustomerID devuelve nombre, apellido y teléfono. El email
// de la cuenta se descarta aquí para no salir del puerto.
func (l *passengerLookup) GetByCustomerID(ctx context.Context, customerID uuid.UUID) (tripsDtos.PassengerContact, error) {
	name, lastName, phone, _, err := l.profiles.GetAccountBasics(ctx, customerID)
	if err != nil {
		return tripsDtos.PassengerContact{}, err
	}
	return tripsDtos.PassengerContact{Name: name, LastName: lastName, Phone: phone}, nil
}
