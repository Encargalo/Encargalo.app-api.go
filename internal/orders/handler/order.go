package handler

import (
	"fmt"
	"net/http"

	"Encargalo.app-api.go/internal/orders/domain/dtos"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OrderHandler interface {
	CreateOrder(c echo.Context) error
}

type orderHandler struct {
	svc ports.OrdersApp
}

func NewOrderHandler(svc ports.OrdersApp) OrderHandler {
	return &orderHandler{svc}
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
// @Failure      500  {object}  dtos.ErrorResponse "Unexpected internal server error"
// @Router       /orders [post]
func (o *orderHandler) CreateOrder(c echo.Context) error {

	ctx := c.Request().Context()

	var order dtos.CreateOrder

	id, ok := ctx.Value("customer_id").(uuid.UUID)
	if !ok {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	order.CustomerID = id

	if err := c.Bind(&order); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := order.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := o.svc.CreateOrder(ctx, &order); err != nil {
		switch err.Error() {
		case "order already exists":
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		case "one or more items not found":
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusCreated, "Order created successfully")
}
