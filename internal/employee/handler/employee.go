package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
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

func (e *employyeHandler) SearchEmployee(c echo.Context) error {

	ctx := c.Request().Context()

	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No session found in context")
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	employee, err := e.svc.SearchEmployeeByID(ctx, session.UserID)
	if err != nil {
		switch err.Error() {
		case "not found":
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
		}
	}

	return c.JSON(http.StatusOK, employee)
}
