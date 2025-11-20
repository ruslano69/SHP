// pkg/converter/errors.go
package converter

import (
	"fmt"
)

type ErrorCode int

const (
	ErrParseFailed ErrorCode = iota + 1
	ErrValidationFailed
	ErrConversionFailed
	ErrTimeout
	ErrContextCanceled
	ErrInvalidInput
)

// Error структурированная ошибка
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
	Field   string // для field-specific ошибок
	Context map[string]interface{}
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

func (e *Error) WithField(field string) *Error {
	e.Field = field
	return e
}

func (e *Error) WithContext(key string, value interface{}) *Error {
	e.Context[key] = value
	return e
}
