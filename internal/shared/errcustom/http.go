package errcustom

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
)

var statusMap = map[error]int{
	ErrNotFound:                 http.StatusNotFound,
	ErrUnexpectedError:          http.StatusInternalServerError,
	ErrBcvRateUnavailable:       http.StatusServiceUnavailable,
	ErrIncorrectAccessData:      http.StatusUnprocessableEntity,
	ErrPhoneAlreadyExist:        http.StatusConflict,
	ErrUnautorized:              http.StatusUnauthorized,
	ErrUploadingImage:           http.StatusUnprocessableEntity,
	ErrConvertingImage:          http.StatusUnprocessableEntity,
	ErrOrderAlreadyExists:       http.StatusConflict,
	ErrItemsNotFound:            http.StatusUnprocessableEntity,
	ErrBatchNotOpen:             http.StatusUnprocessableEntity,
	ErrShopClosed:               http.StatusUnprocessableEntity,
	ErrInvalidCode:              http.StatusBadRequest,
	ErrCodeAlreadyUsed:          http.StatusConflict,
	ErrInvalidTransition:        http.StatusUnprocessableEntity,
	ErrOrderAlreadyAssigned:     http.StatusConflict,
	ErrInsufficientBalance:      http.StatusUnprocessableEntity,
	ErrInvalidBalanceAmount:     http.StatusBadRequest,
	ErrWithdrawalCooldown:       http.StatusTooManyRequests,
	ErrInvalidStatus:            http.StatusBadRequest,
	ErrInvalidFlavors:           http.StatusBadRequest,
	ErrInvalidAdditions:         http.StatusBadRequest,
	ErrInvalidSelection:         http.StatusBadRequest,
	ErrCategoryWithoutFlavors:   http.StatusBadRequest,
	ErrCategoryHasFlavors:       http.StatusBadRequest,
	ErrInvalidDiscountValue:     http.StatusBadRequest,
	ErrInvalidDiscountPercent:   http.StatusBadRequest,
	ErrInvalidDiscountFixed:     http.StatusBadRequest,
	ErrPriceChanged:             http.StatusConflict,
	ErrInvalidCategories:        http.StatusBadRequest,
	ErrProductUnavailable:       http.StatusUnprocessableEntity,
	ErrQuoteNotFound:            http.StatusNotFound,
	ErrQuoteExpired:             http.StatusGone,
	ErrQuoteConsumed:            http.StatusConflict,
	ErrQuoteForbidden:           http.StatusForbidden,
	ErrRefundNotEligible:        http.StatusUnprocessableEntity,
	ErrPagoMovilRequired:        http.StatusUnprocessableEntity,
	ErrRefundAlreadyRequested:   http.StatusConflict,
	ErrInvalidProofURL:          http.StatusBadRequest,
	ErrRefundAmountMismatch:     http.StatusUnprocessableEntity,
	ErrRefundProofRequired:      http.StatusBadRequest,
	ErrFinanceNotesRequired:     http.StatusBadRequest,
	ErrRiderProfileIncomplete:   http.StatusUnprocessableEntity,
	ErrRiderNotActive:           http.StatusForbidden,
	ErrTurboOutOfCoverage:       http.StatusUnprocessableEntity,
	ErrShopOnboardingIncomplete: http.StatusUnprocessableEntity,
	ErrShopNotActive:            http.StatusUnprocessableEntity,
	ErrShopOpen:                 http.StatusConflict,
	ErrInvalidShopType:          http.StatusBadRequest,
	ErrInvalidCoordinates:       http.StatusBadRequest,
	ErrInvalidSearchQuery:       http.StatusBadRequest,
	ErrAccountDisabled:          http.StatusForbidden,
	ErrRiderBusy:                http.StatusConflict,
	ErrEmailAlreadyExist:        http.StatusConflict,
	ErrInvalidCursor:            http.StatusBadRequest,
	ErrInvalidFilter:            http.StatusBadRequest,
	ErrCannotRevokeOwnStaff:     http.StatusBadRequest,
	ErrLastStaffMember:          http.StatusBadRequest,
	ErrInvalidTrackingType:      http.StatusBadRequest,
	ErrTrackingParamsTooLarge:   http.StatusBadRequest,
	ErrTripForbidden:            http.StatusForbidden,
	ErrTripNotCancellable:       http.StatusUnprocessableEntity,
	ErrTripPaymentReviewPending: http.StatusUnprocessableEntity,
}

var codeMap = map[error]string{
	ErrNotFound:                 "not_found",
	ErrUnexpectedError:          "unexpected_error",
	ErrBcvRateUnavailable:       "bcv_rate_unavailable",
	ErrIncorrectAccessData:      "incorrect_access_data",
	ErrPhoneAlreadyExist:        "phone_already_exist",
	ErrUnautorized:              "unauthorized",
	ErrUploadingImage:           "uploading_image",
	ErrConvertingImage:          "converting_image",
	ErrOrderAlreadyExists:       "order_already_exists",
	ErrItemsNotFound:            "items_not_found",
	ErrBatchNotOpen:             "batch_not_open",
	ErrShopClosed:               "shop_closed",
	ErrInvalidCode:              "invalid_code",
	ErrCodeAlreadyUsed:          "code_already_used",
	ErrInvalidTransition:        "invalid_transition",
	ErrOrderAlreadyAssigned:     "order_already_assigned",
	ErrInsufficientBalance:      "insufficient_balance",
	ErrInvalidBalanceAmount:     "invalid_balance_amount",
	ErrWithdrawalCooldown:       "withdrawal_cooldown",
	ErrInvalidStatus:            "invalid_status",
	ErrInvalidFlavors:           "invalid_flavors",
	ErrInvalidAdditions:         "invalid_additions",
	ErrInvalidSelection:         "invalid_selection",
	ErrCategoryWithoutFlavors:   "category_without_flavors",
	ErrCategoryHasFlavors:       "category_has_flavors",
	ErrInvalidDiscountValue:     "invalid_discount_value",
	ErrInvalidDiscountPercent:   "invalid_discount_percent",
	ErrInvalidDiscountFixed:     "invalid_discount_fixed",
	ErrPriceChanged:             "price_changed",
	ErrInvalidCategories:        "invalid_categories",
	ErrProductUnavailable:       "product_unavailable",
	ErrQuoteNotFound:            "quote_not_found",
	ErrQuoteExpired:             "quote_expired",
	ErrQuoteConsumed:            "quote_consumed",
	ErrQuoteForbidden:           "quote_forbidden",
	ErrRefundNotEligible:        "refund_not_eligible",
	ErrPagoMovilRequired:        "pago_movil_required",
	ErrRefundAlreadyRequested:   "refund_already_requested",
	ErrInvalidProofURL:          "invalid_proof_url",
	ErrRefundAmountMismatch:     "refund_amount_mismatch",
	ErrRefundProofRequired:      "refund_proof_required",
	ErrFinanceNotesRequired:     "finance_notes_required",
	ErrRiderProfileIncomplete:   "rider_profile_incomplete",
	ErrRiderNotActive:           "rider_not_active",
	ErrTurboOutOfCoverage:       "turbo_out_of_coverage",
	ErrShopOnboardingIncomplete: "shop_onboarding_incomplete",
	ErrShopNotActive:            "shop_not_active",
	ErrShopOpen:                 "shop_open",
	ErrInvalidShopType:          "invalid_shop_type",
	ErrInvalidCoordinates:       "invalid_coordinates",
	ErrInvalidSearchQuery:       "invalid_search_query",
	ErrAccountDisabled:          "account_disabled",
	ErrRiderBusy:                "rider_busy",
	ErrEmailAlreadyExist:        "email_already_exist",
	ErrInvalidCursor:            "invalid_cursor",
	ErrInvalidFilter:            "invalid_filter",
	ErrCannotRevokeOwnStaff:     "cannot_revoke_own_staff",
	ErrLastStaffMember:          "last_staff_member",
	ErrInvalidTrackingType:      "invalid_tracking_type",
	ErrTrackingParamsTooLarge:   "tracking_params_too_large",
	ErrTripForbidden:            "trip_forbidden",
	ErrTripNotCancellable:       "trip_not_cancellable",
	ErrTripPaymentReviewPending: "trip_payment_review_pending",
}

func Code(sentinel error) string {
	if code, ok := codeMap[sentinel]; ok {
		return code
	}
	return "unexpected_error"
}

func codeForHTTPStatus(status int) string {
	return strings.ReplaceAll(strings.ToLower(http.StatusText(status)), " ", "_")
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
	return resolveError(err).Status
}

func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	info := resolveError(err)
	applog.HTTPFailure(c, info.Status, err)

	if !response.IsV2(c) {
		_ = c.JSON(info.Status, info.Message)
		return
	}

	_ = c.JSON(info.Status, response.Envelope{
		Error: &response.Error{Code: info.Code, Message: fmt.Sprint(info.Message)},
	})
}

type errorInfo struct {
	Status  int
	Message any
	Code    string
}

func resolveError(err error) errorInfo {
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		msg := he.Message
		if msg == nil {
			msg = http.StatusText(he.Code)
		}
		return errorInfo{Status: he.Code, Message: msg, Code: codeForHTTPStatus(he.Code)}
	}

	for sentinel, code := range statusMap {
		if errors.Is(err, sentinel) {
			// Nunca devolver err.Error() completo: puede incluir la causa interna.
			return errorInfo{Status: code, Message: sentinel.Error(), Code: Code(sentinel)}
		}
	}

	return errorInfo{Status: http.StatusInternalServerError, Message: ErrUnexpectedError.Error(), Code: "unexpected_error"}
}
