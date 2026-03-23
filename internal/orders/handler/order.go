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
// @Summary      Create a new order
// @Description  Creates a new order with the provided details
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        order  body      dtos.CreateOrder  true  "Order payload"
// @Success      201    {string}  string  "Order created successfully"
// @Failure      400  {object}  dtos.ErrorResponse "Invalid request body or validation failed"
// @Failure      409  {object}  dtos.ErrorResponse "Order already exists"
// @Failure      422  {object}  dtos.ErrorResponse "One or more items/additions not found"
// @Failure      500  {object}  dtos.ErrorResponse "Unexpected internal server error"
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

	return c.JSON(http.StatusCreated, "Order created successfully")
}
