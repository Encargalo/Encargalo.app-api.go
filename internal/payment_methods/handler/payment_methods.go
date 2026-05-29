package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/payment_methods/domain/models"
	"Encargalo.app-api.go/internal/payment_methods/domain/ports"
	"github.com/labstack/echo/v4"
)

type PaymentMethods interface {
	GetPaymentMethods(c echo.Context) error
}

type paymentMethods struct {
	app ports.PaymentMethodsApp
}

func NewPaymentMethodsHandler(app ports.PaymentMethodsApp) PaymentMethods {
	return &paymentMethods{app}
}

// GetPaymentMethods godoc
// @Summary      Obtener métodos de pago disponibles
// @Description  Retorna la lista de métodos de pago activos. Para PagoMovil se incluyen los campos requeridos al crear una orden.
// @Tags         Payment Methods
// @Produce      json
// @Security     SessionCookie
// @Success      200  {array}   models.PaymentMethod
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /payment-methods [get]
func (p *paymentMethods) GetPaymentMethods(c echo.Context) error {
	methods, err := p.app.GetActivePaymentMethods(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ([]models.PaymentMethod)(methods))
}
