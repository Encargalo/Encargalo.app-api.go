package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/dto"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/labstack/echo/v4"
)

type EmployyeHandler interface {
	SearchEmployee(c echo.Context) error
}

type employyeHandler struct {
	svc ports.EmployeeApp
}

func NewEmployyeHandler(svc ports.EmployeeApp) EmployyeHandler {
	return &employyeHandler{svc}
}

// SearchEmployee godoc
// @Summary      Obtener datos del empleado autenticado
// @Description  Retorna la información del empleado autenticado, incluyendo los datos de su tienda asociada. Requiere rol OrderManager.
// @Tags         Employee
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dto.EmployeeResponse  "Datos del empleado"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Empleado no encontrado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /employee [get]
func (e *employyeHandler) SearchEmployee(c echo.Context) error {

	ctx := c.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No session found in context")
		return errcustom.ErrUnexpectedError
	}

	var employee dto.EmployeeResponse
	employee, err := e.svc.SearchEmployeeByID(ctx, session.UserID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, employee)
}
