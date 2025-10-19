package errcustom

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrUnexpectedError     = errors.New("unexpected error")
	ErrIncorrectAccessData = errors.New("incorrect access data")
)
