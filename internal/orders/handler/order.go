package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/dtos"
	modelsOrder "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OrderHandler interface {
	CreateQuote(c echo.Context) error
	CreateOrder(c echo.Context) error
	ConfirmPickup(c echo.Context) error
	ConfirmDelivery(c echo.Context) error
	GetOrderHistory(c echo.Context) error
}

type orderHandler struct {
	svc  ports.OrdersApp
	svc2 ports.AppManagerOrder
}

func NewOrderHandler(svc ports.OrdersApp, svc2 ports.AppManagerOrder) OrderHandler {
	return &orderHandler{svc, svc2}
}

// CreateQuote godoc
// @Summary      Crear cotización de orden (precio congelado)
// @Description  Calcula y congela el total (descuentos, promo, delivery, BCV) en Redis con TTL. El cliente debe pagar contra este quote y luego crear la orden con `quote_id`. `expires_at` = 30m; `proof_grace_until` = +15m si al crear ya envía comprobante.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        quote  body      dtos.CreateQuote     true  "Carrito a cotizar"
// @Success      201    {object}  dtos.QuoteResponse   "Cotización creada"
// @Failure      400    {object}  dtos.ErrorResponse   "Body invalido"
// @Failure      401    {string}  string               "No autorizado"
// @Failure      422    {object}  dtos.ErrorResponse   "Catalogo/lote no disponible"
// @Failure      500    {object}  dtos.ErrorResponse   "Error interno"
// @Router       /orders/quote [post]
func (o *orderHandler) CreateQuote(c echo.Context) error {
	ctx := c.Request().Context()

	var req dtos.CreateQuote
	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontro el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}
	req.CustomerID = session.UserID

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	quote, err := o.svc.CreateQuote(ctx, &req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, quote)
}

// CreateOrder godoc
// @Summary      Crear una nueva orden
// @Description  Preferido: enviar `quote_id` de POST /orders/quote (total congelado; shop/items/address salen del quote). Legacy: sin quote_id, enviar carrito + `expected_total_price` (409 si el precio cambió). Opcionalmente `balance_amount_usd`. Si queda resto Pago Móvil, requiere `payment_proof_image` y nace en Payment Review.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        order  body      dtos.CreateOrder          true  "Datos de la orden"
// @Success      201    {object}  dtos.CreateOrderResponse  "Orden creada"
// @Failure      400    {object}  dtos.ErrorResponse        "Body invalido o validacion fallida"
// @Failure      401    {string}  string                    "No autorizado"
// @Failure      409    {object}  dtos.ErrorResponse        "La orden ya existe, precio cambió, o quote ya usado"
// @Failure      410    {object}  dtos.ErrorResponse        "Quote expirado"
// @Failure      422    {object}  dtos.ErrorResponse        "Uno o mas productos/adiciones no encontrados"
// @Failure      500    {object}  dtos.ErrorResponse        "Error interno del servidor"
// @Router       /orders [post]
func (o *orderHandler) CreateOrder(c echo.Context) error {

	ctx := c.Request().Context()

	var order dtos.CreateOrder

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontro el usuario en el contexto")
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

	// Payment Review: only notify the customer. Shop sees the order after staff approval.
	if orderModel.Status == string(modelsOrder.StatusPaymentReview) {
		if err = o.svc2.BroadcastOrderToCustomer(*orderModel); err != nil {
			slog.Error("error broadcasting order to customer websocket", "error", err)
		}
	} else if err = o.svc2.BroadcastOrder(*orderModel); err != nil {
		slog.Error("error broadcasting order to websockets", "error", err)
	}

	return c.JSON(http.StatusCreated, dtos.CreateOrderResponse{
		Message:            "Order created successfully",
		TotalPrice:         orderModel.TotalPrice,
		IsLaunchPromo:      orderModel.IsLaunchPromo,
		PromoSubsidyAmount: orderModel.PromoSubsidyAmount,
	})
}

// ConfirmPickup godoc
// @Summary      Confirmar recogida del pedido
// @Description  El local introduce el codigo de 4 digitos que le muestra el rider para confirmar que esta presente. Transiciona el pedido de "In Preparation" o "Ready" a "On The Way" y genera el codigo de entrega para el usuario.
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path      string                   true  "ID del pedido"
// @Param        body  body      dtos.ConfirmCodeRequest  true  "Codigo de recogida"
// @Success      200   {object}  map[string]string        "Pedido en camino"
// @Failure      400   {object}  dtos.ErrorResponse       "Codigo invalido"
// @Failure      409   {object}  dtos.ErrorResponse       "Codigo ya utilizado"
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
// @Description  El rider introduce el codigo de 4 digitos que le muestra el usuario para confirmar la entrega. Transiciona el pedido de "On The Way" a "Completed".
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        id    path      string                   true  "ID del pedido"
// @Param        body  body      dtos.ConfirmCodeRequest  true  "Codigo de entrega"
// @Success      200   {object}  map[string]string        "Pedido entregado"
// @Failure      400   {object}  dtos.ErrorResponse       "Codigo invalido"
// @Failure      409   {object}  dtos.ErrorResponse       "Codigo ya utilizado"
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

	order, err := o.svc.ConfirmDelivery(ctx, orderID, req.Code)
	if err != nil {
		return err
	}

	if err := o.svc2.BroadcastOrder(order); err != nil {
		slog.Error("error broadcasting order after confirm-delivery", "error", err)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Order delivered successfully"})
}

// GetOrderHistory godoc
// @Summary      Historial de ordenes del customer
// @Description  Devuelve todas las ordenes del customer autenticado, incluyendo activas y completadas, ordenadas de mas reciente a mas antigua.
// @Tags         Orders
// @Produce      json
// @Security     SessionCookie
// @Success      200  {array}   dtos.OrderHistoryItem  "Historial de ordenes"
// @Failure      401  {string}  string                 "No autorizado"
// @Failure      500  {object}  dtos.ErrorResponse     "Error interno del servidor"
// @Router       /orders/history [get]
func (o *orderHandler) GetOrderHistory(c echo.Context) error {
	ctx := c.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontro el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	history, err := o.svc.GetOrderHistory(ctx, session.UserID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, history)
}

func parseOrderID(c echo.Context) (uuid.UUID, error) {
	return uuid.Parse(c.Param("id"))
}
