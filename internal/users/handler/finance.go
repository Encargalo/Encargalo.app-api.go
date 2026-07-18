package handler

import (
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	_ "Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/labstack/echo/v4"
)

type CustomerFinanceHandler interface {
	GetBalance(c echo.Context) error
}

type customerFinanceHandler struct {
	svc ports.CustomerFinanceApp
}

func NewCustomerFinanceHandler(svc ports.CustomerFinanceApp) CustomerFinanceHandler {
	return &customerFinanceHandler{svc}
}

// GetBalance godoc
// @Summary      Balance del cliente
// @Description  Retorna el balance en USD del cliente y los últimos 10 movimientos.
// @Tags         Users Finance
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dto.CustomerBalanceResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno"
// @Router       /users/balance [get]
func (h *customerFinanceHandler) GetBalance(c echo.Context) error {
	ctx := c.Request().Context()
	session, ok := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)
	if !ok {
		return errcustom.ErrUnautorized
	}

	resp, err := h.svc.GetBalance(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return c.JSON(http.StatusOK, resp)
}
