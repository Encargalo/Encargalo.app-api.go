package errcustom

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound                 = errors.New("no encontrado")
	ErrUnexpectedError          = errors.New("error inesperado")
	ErrIncorrectAccessData      = errors.New("datos de acceso incorrectos")
	ErrPhoneAlreadyExist        = errors.New("el teléfono ya existe")
	ErrUnautorized              = errors.New("no autorizado")
	ErrOrderAlreadyExists       = errors.New("el pedido ya existe")
	ErrItemsNotFound            = errors.New("uno o más items no fueron encontrados")
	ErrBatchNotOpen             = errors.New("el lote no está abierto")
	ErrShopClosed               = errors.New("el local está cerrado")
	ErrInvalidCode              = errors.New("código de confirmación inválido")
	ErrCodeAlreadyUsed          = errors.New("el código de confirmación ya fue usado")
	ErrInvalidTransition        = errors.New("estado del pedido inválido para esta operación")
	ErrOrderAlreadyAssigned     = errors.New("el pedido ya está asignado a otro rider")
	ErrInsufficientBalance      = errors.New("saldo insuficiente")
	ErrInvalidBalanceAmount     = errors.New("monto de balance inválido")
	ErrWithdrawalCooldown       = errors.New("retiro no permitido: debe esperar 24 horas entre retiros")
	ErrInvalidStatus            = errors.New("estado del pedido inválido")
	ErrInvalidFlavors           = errors.New("uno o más sabores no pertenecen a la categoría del item")
	ErrInvalidAdditions         = errors.New("una o más adiciones no pertenecen al catálogo de adiciones del negocio")
	ErrInvalidSelection         = errors.New("límites de selección de sabores inválidos: min_selection y max_selection deben ser coherentes (min ≤ max); en single_select no pueden superar la cantidad de sabores del item")
	ErrCategoryWithoutFlavors   = errors.New("la categoría no admite sabores")
	ErrCategoryHasFlavors       = errors.New("la categoría tiene sabores y no se puede desactivar la selección de sabores")
	ErrInvalidDiscountValue     = errors.New("valor de descuento inválido para el discount_type indicado")
	ErrInvalidDiscountPercent   = errors.New("el porcentaje debe estar entre 1 y 99")
	ErrInvalidDiscountFixed     = errors.New("el monto fijo debe ser mayor a 0 y menor al precio del producto")
	ErrPriceChanged             = errors.New("el precio del pedido cambió desde la última vez que se mostró al cliente")
	ErrInvalidCategories        = errors.New("una o más categorías no pertenecen al negocio")
	ErrProductUnavailable       = errors.New("uno o más productos no están disponibles para ordenar")
	ErrQuoteNotFound            = errors.New("cotización no encontrada o expirada")
	ErrQuoteExpired             = errors.New("la cotización expiró; solicita una nueva antes de pagar")
	ErrQuoteConsumed            = errors.New("la cotización ya fue utilizada")
	ErrQuoteForbidden           = errors.New("la cotización no pertenece a este cliente")
	ErrRefundNotEligible        = errors.New("devolución no elegible")
	ErrPagoMovilRequired        = errors.New("debe registrar un pago móvil antes de solicitar devolución")
	ErrRefundAlreadyRequested   = errors.New("ya existe una solicitud de devolución para este crédito")
	ErrInvalidProofURL          = errors.New("url de comprobante inválida")
	ErrRefundAmountMismatch     = errors.New("el monto ingresado no coincide con la devolución")
	ErrRefundProofRequired      = errors.New("el comprobante de pago es requerido")
	ErrFinanceNotesRequired     = errors.New("notes es requerido")
	ErrRiderProfileIncomplete   = errors.New("expediente de rider incompleto")
	ErrRiderNotActive           = errors.New("cuenta de rider no activa")
	ErrInvalidDateRange         = errors.New("rango de fechas inválido")
	ErrUnknownETLEntity         = errors.New("entidad ETL desconocida")
	ErrTurboOutOfCoverage       = errors.New("la dirección está fuera de la cobertura Turbo (2 km)")
	ErrShopOnboardingIncomplete = errors.New("faltan datos del negocio para poder activarlo")
	ErrShopNotActive            = errors.New("el negocio no está activado")
	ErrShopOpen                 = errors.New("no se pueden editar los datos del negocio mientras el local está abierto")
	ErrInvalidShopType          = errors.New("shop_type inválido")
	ErrBcvRateUnavailable       = errors.New("tasa BCV no disponible; intenta de nuevo en unos minutos")
	ErrInvalidCoordinates       = errors.New("coordenadas inválidas")
	ErrInvalidSearchQuery       = errors.New("término de búsqueda inválido")
	ErrAccountDisabled          = errors.New("cuenta deshabilitada")
	ErrRiderBusy                = errors.New("ya tienes un viaje en curso")
	ErrMapsUnavailable          = errors.New("el servicio de mapas no respondió")
	ErrEmailAlreadyExist        = errors.New("el email ya existe")
	ErrInvalidCursor            = errors.New("cursor inválido")
	ErrInvalidFilter            = errors.New("filtro inválido")
	ErrCannotRevokeOwnStaff     = errors.New("no puedes quitarte el rol Staff a ti mismo")
	ErrLastStaffMember          = errors.New("debe quedar al menos un usuario con rol Staff")
	ErrInvalidTrackingType      = errors.New("type de tracking inválido")
	ErrTrackingParamsTooLarge   = errors.New("params de tracking demasiado grandes")
	ErrTripForbidden            = errors.New("el viaje no pertenece a este cliente")
	ErrTripNotCancellable       = errors.New("el viaje no se puede cancelar en su estado actual")
	ErrTripPaymentReviewPending = errors.New("no se puede cancelar el viaje mientras el pago móvil está en revisión")
)

var (
	ErrUploadingImage  = errors.New("error al subir la imagen")
	ErrConvertingImage = errors.New("error al convertir la imagen a bytes")
)

// Unexpected envuelve la causa real bajo ErrUnexpectedError.
// El cliente solo ve "error inesperado"; los logs pueden leer la causa con errors.Unwrap / Internal.
func Unexpected(cause error) error {
	if cause == nil {
		return ErrUnexpectedError
	}
	if errors.Is(cause, ErrUnexpectedError) {
		return cause
	}
	return fmt.Errorf("%w: %v", ErrUnexpectedError, cause)
}
