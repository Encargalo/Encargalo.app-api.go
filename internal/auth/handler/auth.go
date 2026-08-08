package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/auth/handler/request"
	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Auth interface {
	SignInCustomer(e echo.Context) error
	SignInEmployee(e echo.Context) error
	SignInRider(e echo.Context) error
	SignInStaff(e echo.Context) error
	SignInGoogle(e echo.Context) error
	Logout(e echo.Context) error
	VerifySession(e echo.Context) error
}

type auth struct {
	svc    ports.AuthApp
	jwt    jwt.Sessions
	cookie cookie.CookieHandler
}

func NewAuthHandler(svc ports.AuthApp, jwt jwt.Sessions, cookie cookie.CookieHandler) Auth {
	return &auth{svc, jwt, cookie}
}

// SignInCustomer godoc
// @Summary      Iniciar sesión como cliente
// @Description  Autentica a un cliente usando su número de teléfono y contraseña.
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInRequest  true  "Datos de inicio de sesión"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      422    {object}  map[string]string  "Credenciales incorrectas"
// @Failure      429    {object}  map[string]string  "Demasiados intentos, intenta de nuevo más tarde"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/users [post]
func (a *auth) SignInCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	var signIn request.SignInRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInCustomer(ctx, signIn.Phone, signIn.Password)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	jwtSession, err := a.jwt.CreateSession(sessionID)
	if err != nil {
		return errcustom.ToHTTPError(errcustom.ErrUnexpectedError)
	}

	e.SetCookie(a.cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")
}

// SignInEmployee godoc
// @Summary      Iniciar sesión como empleado
// @Description  Autentica a un empleado usando su número de teléfono y contraseña.
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInRequest  true  "Datos de inicio de sesión"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      422    {object}  map[string]string  "Credenciales incorrectas"
// @Failure      429    {object}  map[string]string  "Demasiados intentos, intenta de nuevo más tarde"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/employees [post]
func (a *auth) SignInEmployee(e echo.Context) error {

	ctx := e.Request().Context()

	var signIn request.SignInRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInEmployee(ctx, signIn.Phone, signIn.Password)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	jwtSession, err := a.jwt.CreateSession(sessionID)
	if err != nil {
		return errcustom.ToHTTPError(errcustom.ErrUnexpectedError)
	}

	e.SetCookie(a.cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")
}

// SignInRider godoc
// @Summary      Iniciar sesión como rider
// @Description  Autentica a un rider usando su número de teléfono y contraseña.
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInRequest  true  "Datos de inicio de sesión"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      422    {object}  map[string]string  "Credenciales incorrectas"
// @Failure      429    {object}  map[string]string  "Demasiados intentos, intenta de nuevo más tarde"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/riders [post]
func (a *auth) SignInRider(e echo.Context) error {
	ctx := e.Request().Context()

	var signIn request.SignInRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInRider(ctx, signIn.Phone, signIn.Password)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	jwtSession, err := a.jwt.CreateSession(sessionID)
	if err != nil {
		return errcustom.ToHTTPError(errcustom.ErrUnexpectedError)
	}

	e.SetCookie(a.cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")
}

// SignInStaff godoc
// @Summary      Iniciar sesión como staff
// @Description  Autentica a un miembro del staff (CRM) usando teléfono y contraseña.
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInRequest  true  "Datos de inicio de sesión"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      422    {object}  map[string]string  "Credenciales incorrectas"
// @Failure      429    {object}  map[string]string  "Demasiados intentos, intenta de nuevo más tarde"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/staff [post]
func (a *auth) SignInStaff(e echo.Context) error {
	ctx := e.Request().Context()

	var signIn request.SignInRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInStaff(ctx, signIn.Phone, signIn.Password)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	jwtSession, err := a.jwt.CreateSession(sessionID)
	if err != nil {
		return errcustom.ToHTTPError(errcustom.ErrUnexpectedError)
	}

	e.SetCookie(a.cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")
}

// SignInGoogle godoc
// @Summary      Iniciar sesión o registrarse con Google
// @Description  Verifica un id_token de Google emitido por la app móvil. Si el email ya
// @Description  pertenece a un customer, inicia sesión; si no existe, crea la cuenta
// @Description  automáticamente (sin teléfono, que se completa luego desde el perfil).
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInGoogleRequest  true  "id_token de Google"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      401    {object}  map[string]string  "id_token inválido o expirado"
// @Failure      429    {object}  map[string]string  "Demasiados intentos, intenta de nuevo más tarde"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/google [post]
func (a *auth) SignInGoogle(e echo.Context) error {

	ctx := e.Request().Context()

	var signIn request.SignInGoogleRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInGoogle(ctx, signIn.IDToken)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	jwtSession, err := a.jwt.CreateSession(sessionID)
	if err != nil {
		return errcustom.ToHTTPError(errcustom.ErrUnexpectedError)
	}

	e.SetCookie(a.cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")
}

// VerifySession godoc
// @Summary      Verificar sesión (uso interno del reverse proxy)
// @Description  Endpoint interno llamado por el ForwardAuth de Traefik en cada
// @Description  request. Valida el JWT de la cookie encargalo-session contra
// @Description  la sesión activa y, si es válida, devuelve el header
// @Description  X-Encargalo-Session con el ActiveSession serializado en JSON,
// @Description  que Traefik reenvía al backend real. No debe llamarse desde
// @Description  clientes de negocio (frontend/apps) — solo desde el gateway.
// @Tags         Internal
// @Success      200  "sesión válida, JSON en el header X-Encargalo-Session"
// @Failure      401  {string}  string  "sesión inválida, ausente o expirada"
// @Router       /internal/verify-session [post]
func (a *auth) VerifySession(e echo.Context) error {
	req := e.Request()

	var tokenString string
	if c, err := e.Cookie("encargalo-session"); err == nil {
		tokenString = c.Value
	}

	claims, err := a.jwt.ValidateToken(tokenString)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "token inválido")
	}

	rawID, ok := claims["session_id"].(string)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "session_id inválido")
	}

	sessionID, err := uuid.Parse(rawID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "session_id inválido")
	}

	session, err := a.svc.SearchSessions(req.Context(), sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "sesión no encontrada")
	}

	if time.Now().After(session.ExpiresAt) {
		return echo.NewHTTPError(http.StatusUnauthorized, "sesión expirada")
	}

	raw, err := json.Marshal(session)
	if err != nil {
		return errcustom.ToHTTPError(errcustom.ErrUnexpectedError)
	}

	e.Response().Header().Set(sharedmw.SessionHeader, string(raw))
	return e.NoContent(http.StatusOK)
}

type LogoutResponse struct {
	Message string `json:"message"`
}

// Logout godoc
// @Summary      Cerrar sesión
// @Description  Elimina la sesión activa del cliente o empleado autenticado y borra la cookie de autenticación.
// @Tags         Auth
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  handler.LogoutResponse
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Sesión no encontrada"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /auth/logout [delete]
func (s *auth) Logout(e echo.Context) error {

	ctx := e.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	if err := s.svc.DeleteSession(ctx, session.ID); err != nil {
		return errcustom.ToHTTPError(err)
	}

	e.SetCookie(s.cookie.DeleteCookieSession())

	return e.JSON(http.StatusOK, LogoutResponse{
		Message: "session deleted success",
	})
}
