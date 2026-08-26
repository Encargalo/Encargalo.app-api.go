package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"

	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"

	"Encargalo.app-api.go/internal/auth/domain/models"
	portsAuth "Encargalo.app-api.go/internal/auth/domain/ports"
)

type UsersHandler interface {
	RegisterCustomer(e echo.Context) error
	SearchCustomer(e echo.Context) error
	UpdateCustomer(e echo.Context) error
	UpdatePassword(e echo.Context) error
	DeleteAccount(e echo.Context) error
}

type customersHandler struct {
	customerApp ports.UsersApp
	auth        portsAuth.AuthApp
	jwt         jwt.Sessions
	cookie      cookie.CookieHandler
}

func NewUsersHandler(customerApp ports.UsersApp, auth portsAuth.AuthApp, jwt jwt.Sessions, cookie cookie.CookieHandler) UsersHandler {
	return &customersHandler{customerApp, auth, jwt, cookie}
}

// RegisterCustomer godoc
// @Summary      Registrar un nuevo cliente
// @Description  Registra un nuevo cliente en el sistema. Al completarse, crea automáticamente la sesión y establece la cookie de autenticación.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        customer  body      dto.RegisterCustomer  true  "Datos del cliente"
// @Success      201       {string}  string                "customer successfully registered"
// @Failure      400       {string}  string                "Campo inválido o body vacío"
// @Failure      409       {string}  string                "El teléfono ya está registrado"
// @Failure      500       {string}  string                "Error interno del servidor"
// @Router       /users [post]
func (c *customersHandler) RegisterCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer := dto.RegisterCustomer{}

	if err := e.Bind(&customer); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := customer.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.customerApp.RegisterCustomer(ctx, customer); err != nil {
		return errcustom.ToHTTPError(err)
	}

	sessionId, err := c.auth.SignInCustomer(ctx, customer.Phone, customer.Password)
	if err != nil {
		applog.Error(ctx, "auto-login tras registro falló", err, "phone", customer.Phone)
		return response.Send(e, http.StatusCreated, "customer successfully registered")
	}

	jwtSession, err := c.jwt.CreateSession(sessionId)
	if err != nil {
		return errcustom.ErrUnexpectedError
	}

	e.SetCookie(c.cookie.CreateCookieSession(jwtSession))

	return response.Send(e, http.StatusCreated, "customer successfully registered")
}

// SearchCustomer godoc
// @Summary      Obtener perfil del usuario autenticado
// @Description  Retorna los datos del usuario autenticado sin importar su rol. Para usuarios con rol OrderManager u Owner, la respuesta incluye adicionalmente los campos `shop_id` y `shop`. Si el usuario solicitó eliminar su cuenta, incluye `scheduled_deletion_at` con la fecha en la que se finalizará la eliminación de no volver a iniciar sesión antes.
// @Tags         Users
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dto.CustomerResponse  "Datos del usuario"
// @Failure      401  {string}  string                "No autorizado"
// @Failure      404  {string}  string                "Usuario no encontrado"
// @Failure      500  {string}  string                "Error interno del servidor"
// @Router       /users [get]
func (c *customersHandler) SearchCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	profile, err := c.customerApp.SearchCustomerProfileByID(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return response.Send(e, http.StatusOK, profile.ToDomainDTO())
}

// UpdateCustomer godoc
// @Summary      Actualiza la información del usuario autenticado
// @Description  Actualización parcial: solo se modifican los campos presentes en el cuerpo de la solicitud (name, sur_name, phone, email son todos opcionales), el resto de la cuenta queda intacto.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        customer  body      dto.UpdateCustomer  true  "Datos del usuario a actualizar"
// @Success      200       {string}  string              "customer updated success"
// @Failure      400       {string}  string              "Error de validación o formato inválido"
// @Failure      401       {string}  string              "No autorizado"
// @Failure      404       {string}  string              "Usuario no encontrado"
// @Failure      409       {string}  string              "El teléfono ya está registrado"
// @Failure      500       {string}  string              "Error interno del servidor"
// @Router       /users [put]
func (c *customersHandler) UpdateCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer := dto.UpdateCustomer{}

	if err := e.Bind(&customer); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := customer.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if err := c.customerApp.UpdateCustomer(ctx, session.UserID, customer); err != nil {
		return errcustom.ToHTTPError(err)
	}

	return response.Send(e, http.StatusOK, "customer updated success")
}

// UpdatePassword godoc
// @Summary      Actualiza la contraseña del usuario autenticado
// @Description  Permite al usuario autenticado actualizar su contraseña, validando el formato y los requisitos establecidos.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        password  body      dto.UpdatePassword  true  "Nueva contraseña"
// @Success      200       {string}  string              "password updated success"
// @Failure      400       {string}  string              "Error de validación o formato inválido"
// @Failure      401       {string}  string              "No autorizado"
// @Failure      404       {string}  string              "Usuario no encontrado"
// @Failure      500       {string}  string              "Error interno del servidor"
// @Router       /users/change-password [put]
func (c *customersHandler) UpdatePassword(e echo.Context) error {

	ctx := e.Request().Context()

	pass := dto.UpdatePassword{}

	if err := e.Bind(&pass); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := pass.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if err := c.customerApp.UpdatePassword(ctx, session.UserID, pass); err != nil {
		return errcustom.ToHTTPError(err)
	}

	return response.Send(e, http.StatusOK, "password updated success")
}

// DeleteAccount godoc
// @Summary      Solicitar la eliminación de la cuenta del usuario autenticado
// @Description  Marca la cuenta como "pendiente de eliminación" por 30 días y cierra la sesión en todos los dispositivos. Ningún dato se borra en este momento. Si el usuario vuelve a iniciar sesión dentro de ese plazo, la solicitud se cancela automáticamente. Si no vuelve, la cuenta se marca deleted_at de forma permanente sin perder ninguna columna.
// @Tags         Users
// @Produce      json
// @Security     SessionCookie
// @Success      200  {string}  string  "account scheduled for deletion"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /users [delete]
func (c *customersHandler) DeleteAccount(e echo.Context) error {

	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if err := c.customerApp.RequestAccountDeletion(ctx, session.UserID); err != nil {
		return errcustom.ToHTTPError(err)
	}

	if err := c.auth.DeleteAllSessionsForUser(ctx, session.UserID); err != nil {
		applog.Error(ctx, "no se pudieron invalidar las sesiones tras solicitar la eliminación de la cuenta", err, "user_id", session.UserID.String())
	}

	e.SetCookie(c.cookie.DeleteCookieSession())

	return response.Send(e, http.StatusOK, "account scheduled for deletion")
}
