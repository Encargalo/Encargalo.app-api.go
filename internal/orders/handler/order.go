package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OrderHandler interface {
	CreateOrder(c echo.Context) error
	ConfirmPickup(c echo.Context) error
	ConfirmDelivery(c echo.Context) error
}

type orderHandler struct {
	svc  ports.OrdersApp
	svc2 ports.AppManagerOrder
}

func NewOrderHandler(svc ports.OrdersApp, svc2 ports.AppManagerOrder) OrderHandler {
	return &orderHandler{svc, svc2}
}

// CreateOrder godoc
// @Summary      Crear una nueva orden
// @Description  Crea una nueva orden con los productos e información de entrega proporcionados. El customer_id se obtiene automáticamente del token de sesión. Se debe enviar `payment_proof_image` en base64.
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

// ConfirmPickup godoc
// @Summary      Confirmar recogida del pedido
// @Description  El local introduce el código de 6 dígitos que le muestra el rider para confirmar que está presente. Transiciona el pedido de "In Preparation" o "Ready" a "On The Way" y genera el código de entrega para el usuario.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path      string                   true  "ID del pedido"
// @Param        body  body      dtos.ConfirmCodeRequest  true  "Código de recogida"
// @Success      200   {object}  map[string]string        "Pedido en camino"
// @Failure      400   {object}  dtos.ErrorResponse       "Código inválido"
// @Failure      409   {object}  dtos.ErrorResponse       "Código ya utilizado"
// @Failure      422   {object}  dtos.ErrorResponse       "Estado del pedido incorrecto"
// @Failure      404   {object}  dtos.ErrorResponse       "Pedido no encontrado"
// @Router       /orders/{id}/confirm-pickup [post]
func (o *orderHandler) ConfirmPickup(c echo.Context) error {
	ctx := c.Request().Context()

	orderID, err := parseOrderID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var req dtos.ConfirmCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	order, err := o.svc.ConfirmPickup(ctx, orderID, req.Code)
	if err != nil {
		return err
	}

	if err := o.svc2.BroadcastOrder(order); err != nil {
		slog.Error("error broadcasting order after confirm-pickup", "error", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Order is now on the way"})
}

// ConfirmDelivery godoc
// @Summary      Confirmar entrega del pedido
// @Description  El rider introduce el código de 6 dígitos que le muestra el usuario para confirmar la entrega. Transiciona el pedido de "On The Way" a "Completed".
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path      string                   true  "ID del pedido"
// @Param        body  body      dtos.ConfirmCodeRequest  true  "Código de entrega"
// @Success      200   {object}  map[string]string        "Pedido entregado"
// @Failure      400   {object}  dtos.ErrorResponse       "Código inválido"
// @Failure      409   {object}  dtos.ErrorResponse       "Código ya utilizado"
// @Failure      422   {object}  dtos.ErrorResponse       "Estado del pedido incorrecto"
// @Failure      404   {object}  dtos.ErrorResponse       "Pedido no encontrado"
// @Router       /orders/{id}/confirm-delivery [post]
func (o *orderHandler) ConfirmDelivery(c echo.Context) error {
	ctx := c.Request().Context()

	orderID, err := parseOrderID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var req dtos.ConfirmCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := o.svc.ConfirmDelivery(ctx, orderID, req.Code); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Order delivered successfully"})
}

func parseOrderID(c echo.Context) (uuid.UUID, error) {
	return uuid.Parse(c.Param("id"))
}
