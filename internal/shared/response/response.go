package response

import "github.com/labstack/echo/v4"

const VersionHeader = "X-Api-Response-Version"
const VersionV2 = "2"

type Envelope struct {
	Data  any    `json:"data"`
	Error *Error `json:"error"`
	Meta  any    `json:"meta,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func IsV2(c echo.Context) bool {
	return c.Request().Header.Get(VersionHeader) == VersionV2
}

func Send(c echo.Context, status int, data any) error {
	if !IsV2(c) {
		return c.JSON(status, data)
	}
	return c.JSON(status, Envelope{Data: data})
}

func SendWithMeta(c echo.Context, status int, data any, meta any) error {
	if !IsV2(c) {
		return c.JSON(status, data)
	}
	return c.JSON(status, Envelope{Data: data, Meta: meta})
}

// v1Body es lo que se manda hoy sin envolver; code/message/details arman
// Envelope.Error cuando el cliente pide v2.
func SendError(c echo.Context, status int, v1Body any, code, message string, details any) error {
	if !IsV2(c) {
		return c.JSON(status, v1Body)
	}
	return c.JSON(status, Envelope{Error: &Error{Code: code, Message: message, Details: details}})
}
