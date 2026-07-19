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
	ErrRefundNotEligible:      http.StatusUnprocessableEntity,
	ErrPagoMovilRequired:      http.StatusUnprocessableEntity,
	ErrRefundAlreadyRequested: http.StatusConflict,
	ErrInvalidProofURL:        http.StatusBadRequest,
	ErrRefundAmountMismatch:   http.StatusUnprocessableEntity,
	ErrRefundProofRequired:    http.StatusBadRequest,
	ErrRiderProfileIncomplete: http.StatusUnprocessableEntity,
	ErrRiderNotActive:         http.StatusForbidden,
}

func ToHTTPError(err error) error {
	if err == nil {
		return nil
	}
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		return he
	}
	for sentinel, code := range statusMap {
		if errors.Is(err, sentinel) {
			out := echo.NewHTTPError(code, sentinel.Error())
			// Conservar la cadena completa (p. ej. "error inesperado: pq: ...") en Internal para logs.
			if err.Error() != sentinel.Error() {
				_ = out.SetInternal(err)
			}
			return out
		}
	}
	out := echo.NewHTTPError(http.StatusInternalServerError, ErrUnexpectedError.Error())
	_ = out.SetInternal(err)
	return out
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
			// Nunca devolver err.Error() completo: puede incluir la causa interna.
			return code, sentinel.Error()
		}
	}

	return http.StatusInternalServerError, ErrUnexpectedError.Error()
}
