package errcustom

import "errors"

var (
	ErrNotFound               = errors.New("no encontrado")
	ErrUnexpectedError        = errors.New("error inesperado")
	ErrIncorrectAccessData    = errors.New("datos de acceso incorrectos")
	ErrPhoneAlreadyExist      = errors.New("el teléfono ya existe")
	ErrUnautorized            = errors.New("no autorizado")
	ErrOrderAlreadyExists     = errors.New("el pedido ya existe")
	ErrItemsNotFound          = errors.New("uno o más items no fueron encontrados")
	ErrBatchNotOpen           = errors.New("el lote no está abierto")
	ErrInvalidCode            = errors.New("código de confirmación inválido")
	ErrCodeAlreadyUsed        = errors.New("el código de confirmación ya fue usado")
	ErrInvalidTransition      = errors.New("estado del pedido inválido para esta operación")
	ErrOrderAlreadyAssigned   = errors.New("el pedido ya está asignado a otro rider")
	ErrInsufficientBalance    = errors.New("saldo insuficiente para el retiro")
	ErrWithdrawalCooldown     = errors.New("retiro no permitido: debe esperar 24 horas entre retiros")
	ErrInvalidStatus          = errors.New("estado del pedido inválido")
	ErrInvalidFlavors         = errors.New("uno o más sabores no pertenecen a la categoría del item")
	ErrInvalidAdditions       = errors.New("una o más adiciones no pertenecen al catálogo de adiciones del negocio")
	ErrInvalidSelection       = errors.New("límites de selección de sabores inválidos: min_selection y max_selection deben ser coherentes (min ≤ max); en single_select no pueden superar la cantidad de sabores del item")
	ErrCategoryWithoutFlavors = errors.New("la categoría no admite sabores")
	ErrCategoryHasFlavors     = errors.New("la categoría tiene sabores y no se puede desactivar la selección de sabores")
	ErrInvalidDiscountValue   = errors.New("valor de descuento inválido para el discount_type indicado")
	ErrPriceChanged           = errors.New("el precio del pedido cambió desde la última vez que se mostró al cliente")
	ErrInvalidCategories      = errors.New("una o más categorías no pertenecen al negocio")
	ErrProductUnavailable     = errors.New("uno o más productos no están disponibles para ordenar")
)

var (
	ErrUploadingImage  = errors.New("error al subir la imagen")
	ErrConvertingImage = errors.New("error al convertir la imagen a bytes")
)
