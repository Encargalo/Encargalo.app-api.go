package errcustom

import (
	"errors"
	"net/http"

	"Encargalo.app-api.go/internal/shared/applog"
	"github.com/labstack/echo/v4"
)

var statusMap = map[error]int{
	ErrNotFound:               http.StatusNotFound,
	ErrUnexpectedError:        http.StatusInternalServerError,
	ErrIncorrectAccessData:    http.StatusUnprocessableEntity,
	ErrPhoneAlreadyExist:      http.StatusConflict,
	ErrUnautorized:            http.StatusUnauthorized,
	ErrUploadingImage:         http.StatusUnprocessableEntity,
	ErrConvertingImage:        http.StatusUnprocessableEntity,
	ErrOrderAlreadyExists:     http.StatusConflict,
	ErrItemsNotFound:          http.StatusUnprocessableEntity,
	ErrBatchNotOpen:           http.StatusUnprocessableEntity,
	ErrInvalidCode:            http.StatusBadRequest,
	ErrCodeAlreadyUsed:        http.StatusConflict,
	ErrInvalidTransition:      http.StatusUnprocessableEntity,
	ErrOrderAlreadyAssigned:   http.StatusConflict,
	ErrInsufficientBalance:    http.StatusUnprocessableEntity,
	ErrInvalidBalanceAmount:   http.StatusBadRequest,
	ErrWithdrawalCooldown:     http.StatusTooManyRequests,
	ErrInvalidStatus:          http.StatusBadRequest,
	ErrInvalidFlavors:         http.StatusBadRequest,
	ErrInvalidAdditions:       http.StatusBadRequest,
	ErrInvalidSelection:       http.StatusBadRequest,
	ErrCategoryWithoutFlavors: http.StatusBadRequest,
	ErrCategoryHasFlavors:     http.StatusBadRequest,
	ErrInvalidDiscountValue:   http.StatusBadRequest,
	ErrPriceChanged:           http.StatusConflict,
	ErrInvalidCategories:      http.StatusBadRequest,
	ErrProductUnavailable:     http.StatusUnprocessableEntity,
	ErrQuoteNotFound:          http.StatusNotFound,
	ErrQuoteExpired:           http.StatusGone,
	ErrQuoteConsumed:          http.StatusConflict,
	ErrQuoteForbidden:         http.StatusForbidden,
}

func ToHTTPError(err error) error {
	for sentinel, code := range statusMap {
		if errors.Is(err, sentinel) {
			return echo.NewHTTPError(code, err.Error())
		}
	}
	return echo.NewHTTPError(http.StatusInternalServerError, ErrUnexpectedError.Error())
}

// HTTPStatus resuelve el código HTTP de un error (para access log antes del ErrorHandler).
func HTTPStatus(err error) int {
	status, _ := resolveHTTPError(err)
	return status
}

func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	status, message := resolveHTTPError(err)
	applog.HTTPFailure(c, status, err)

	_ = c.JSON(status, message)
}

func resolveHTTPError(err error) (int, any) {
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		msg := he.Message
		if msg == nil {
			msg = http.StatusText(he.Code)
		}
		return he.Code, msg
	}

	for sentinel, code := range statusMap {
		if errors.Is(err, sentinel) {
			return code, err.Error()
		}
	}

	return http.StatusInternalServerError, ErrUnexpectedError.Error()
}
