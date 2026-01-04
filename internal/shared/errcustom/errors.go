package errcustom

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrUnexpectedError     = errors.New("unexpected error")
	ErrIncorrectAccessData = errors.New("incorrect access data")
	ErrPhoneAlreadyExist   = errors.New("phone al ready exist")
	ErrUnautorized         = errors.New("unautorized")
)

var (
	ErrUploadingImage  = errors.New("error uploading image")
	ErrConvertingImage = errors.New("error converting image to bytes")
)
