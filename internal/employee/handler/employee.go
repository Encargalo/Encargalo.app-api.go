package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
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

func (e *employyeHandler) SearchEmployee(c echo.Context) error {

	ctx := c.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No session found in context")
		return errcustom.ErrUnexpectedError
	}

	employee, err := e.svc.SearchEmployeeByID(ctx, session.UserID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, employee)
}
