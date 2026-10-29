package dtos

import "context"

// SetAvailabilityRequest pausa o reanuda la recepción de nuevas órdenes y
// viajes. El rider sigue conectado y conserva lo que ya tiene asignado.
type SetAvailabilityRequest struct {
	AcceptingNewWork bool `json:"accepting_new_work" example:"false"`
}

func (r *SetAvailabilityRequest) Validate() error {
	return applicationValidate.StructCtx(context.Background(), r)
}
