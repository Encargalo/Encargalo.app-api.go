package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/app"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/labstack/echo/v4"
)

type OrderHandler interface {
	CreateOrder(c echo.Context) error
}

type orderHandler struct {
	svc  ports.OrdersApp
	svc2 app.AppManagerOrder
}

func NewOrderHandler(svc ports.OrdersApp, svc2 app.AppManagerOrder) OrderHandler {
	return &orderHandler{svc, svc2}
}

// CreateOrder godoc
// @Summary      Crear una nueva orden
// @Description  Crea una nueva orden con los productos e información de entrega proporcionados. El customer_id se obtiene automáticamente del token de sesión. Si el método de pago es PagoMovil, se debe enviar `pago_movil_data` y `payment_proof_image` (base64).
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        order  body      dtos.CreateOrder          true  "Datos de la orden"
// @Success      201    {object}  dtos.CreateOrderResponse  "Orden creada"
// @Failure      400    {object}  dtos.ErrorResponse        "Body inválido o validación fallida"
// @Failure      401    {string}  string                    "No autorizado"
// @Failure      409    {object}  dtos.ErrorResponse        "La orden ya existe"
// @Failure      422    {object}  dtos.ErrorResponse        "Uno o más productos/adiciones no encontrados"
// @Failure      500    {object}  dtos.ErrorResponse        "Error interno del servidor"
// @Router       /orders [post]
func (o *orderHandler) CreateOrder(c echo.Context) error {

	ctx := c.Request().Context()

	var order dtos.CreateOrder

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	order.CustomerID = session.UserID

	if err := c.Bind(&order); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := order.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	orderModel, err := o.svc.CreateOrder(ctx, &order)
	if err != nil {
		return err
	}

	if err = o.svc2.BroadcastOrder(*orderModel); err != nil {
		slog.Error("error broadcasting order to websockets", "error", err)
	}

	return c.JSON(http.StatusCreated, dtos.CreateOrderResponse{Message: "Order created successfully"})
}
