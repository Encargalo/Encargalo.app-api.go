package dtos

import "context"

// ReportLocationRequest reporta la ubicación del rider fuera del WebSocket —
// la app la manda también en background, cuando el socket no está vivo. Por
// eso trae `mode` explícito: sin conexión no hay de dónde inferirlo.
type ReportLocationRequest struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude" example:"10.4806"`
	Longitude float64 `json:"longitude" validate:"required,longitude" example:"-66.9036"`
	Mode      string  `json:"mode" validate:"required,oneof=orders trips" example:"trips" enums:"orders,trips"`
}

func (r *ReportLocationRequest) Validate() error {
	return applicationValidate.StructCtx(context.Background(), r)
}
