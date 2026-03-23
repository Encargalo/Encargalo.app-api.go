package errcustom

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrUnexpectedError     = errors.New("unexpected error")
	ErrIncorrectAccessData = errors.New("incorrect access data")
	ErrPhoneAlreadyExist   = errors.New("phone already exist")
	ErrUnautorized         = errors.New("unauthorized")
	ErrOrderAlreadyExists  = errors.New("order already exists")
	ErrItemsNotFound       = errors.New("one or more items not found")
	ErrBatchNotOpen        = errors.New("batch not open")
)

var (
	ErrUploadingImage  = errors.New("error uploading image")
	ErrConvertingImage = errors.New("error converting image to bytes")
)
