package errcustom

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

var statusMap = map[error]int{
	ErrNotFound:             http.StatusNotFound,
	ErrUnexpectedError:      http.StatusInternalServerError,
	ErrIncorrectAccessData:  http.StatusUnprocessableEntity,
	ErrPhoneAlreadyExist:    http.StatusConflict,
	ErrUnautorized:          http.StatusUnauthorized,
	ErrUploadingImage:       http.StatusUnprocessableEntity,
	ErrConvertingImage:      http.StatusUnprocessableEntity,
	ErrOrderAlreadyExists:   http.StatusConflict,
	ErrItemsNotFound:        http.StatusUnprocessableEntity,
	ErrBatchNotOpen:         http.StatusUnprocessableEntity,
	ErrInvalidCode:          http.StatusBadRequest,
	ErrCodeAlreadyUsed:      http.StatusConflict,
	ErrInvalidTransition:    http.StatusUnprocessableEntity,
	ErrOrderAlreadyAssigned: http.StatusConflict,
	ErrRiderInRiskZone:      http.StatusForbidden,
	ErrInsufficientBalance:  http.StatusUnprocessableEntity,
	ErrInvalidStatus:        http.StatusBadRequest,
}

func ToHTTPError(err error) error {
	for sentinel, code := range statusMap {
		if errors.Is(err, sentinel) {
			return echo.NewHTTPError(code, err.Error())
		}
	}
	return echo.NewHTTPError(http.StatusInternalServerError, ErrUnexpectedError.Error())
}

func HTTPErrorHandler(err error, c echo.Context) {
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		c.JSON(he.Code, he.Message) //nolint:errcheck
		return
	}

	for sentinel, code := range statusMap {
		if errors.Is(err, sentinel) {
			c.JSON(code, err.Error()) //nolint:errcheck
			return
		}
	}

	c.JSON(http.StatusInternalServerError, ErrUnexpectedError.Error()) //nolint:errcheck
}
