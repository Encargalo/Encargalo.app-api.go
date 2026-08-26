package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Health struct {
	Code    int    `json:"status"`
	Message string `json:"message"`
}

// HealthCheck responde 200 cuando el proceso ya está escuchando.
// Railway lo consulta en cada deploy: sin 200 el deploy falla y se mantiene la versión anterior.
func HealthCheck(context echo.Context) error {
	response := &Health{
		Code:    http.StatusOK,
		Message: "Active!",
	}

	return context.JSON(http.StatusOK, response)
}
