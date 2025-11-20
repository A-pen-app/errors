package errors

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ErrorType string

const (
	ErrorTypePost ErrorType = "post"
)

type ErrorCode string

const (
	KeyNotFound         ErrorCode = "NOT_FOUND"
	KeyNotAllowed       ErrorCode = "ACTION_NOT_ALLOWED"
	KeyWrongParams      ErrorCode = "WRONG_PARAMETER"
	KeyPermissionDenied ErrorCode = "PERMISSION_DENIED"
	KeyInternalError    ErrorCode = "INTERNAL_ERROR"
)

var (
	ErrorNotFound         = errors.New("data not found")
	ErrorNotAllowed       = errors.New("action not allowed")
	ErrorWrongParams      = errors.New("wrong parameters")
	ErrorPermissionDenied = errors.New("permission denied")
	ErrorInternalError    = errors.New("internal system error")
)

type ErrorMapping struct {
	Code       ErrorCode
	StatusCode int
}

var errorMappings = map[error]ErrorMapping{
	ErrorNotFound:         {KeyNotFound, http.StatusNotFound},
	ErrorNotAllowed:       {KeyNotAllowed, http.StatusForbidden},
	ErrorWrongParams:      {KeyWrongParams, http.StatusBadRequest},
	ErrorPermissionDenied: {KeyPermissionDenied, http.StatusForbidden},
	ErrorInternalError:    {KeyInternalError, http.StatusInternalServerError},
	sql.ErrNoRows:         {KeyNotFound, http.StatusNotFound},
}

type AppError struct {
	cause error
	data  map[string]any
}

func (e *AppError) Error() string {
	if len(e.data) == 0 {
		return e.cause.Error()
	}

	dataStr := make([]string, 0, len(e.data))
	for k, v := range e.data {
		dataStr = append(dataStr, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("%s: %s", e.cause.Error(), strings.Join(dataStr, " "))
}

func (e *AppError) Data() map[string]any {
	if e.data == nil {
		return make(map[string]any)
	}
	return e.data
}

func (e *AppError) Unwrap() error {
	return e.cause
}

type HttpError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id"`
}

// parseKeyValues converts logging-style key-value pairs into a map.
func parseKeyValues(keyValues []any) map[string]any {
	if len(keyValues) == 0 {
		return make(map[string]any)
	}

	data := make(map[string]any)
	for i := 0; i < len(keyValues)-1; i += 2 {
		if key, ok := keyValues[i].(string); ok {
			data[key] = keyValues[i+1]
		}
	}
	return data
}

// getErrorMapping returns the unified error mapping for a given error.
func getErrorMapping(err error) ErrorMapping {
	// Check for binding errors first
	if isBindingError(err) {
		return ErrorMapping{KeyWrongParams, http.StatusBadRequest}
	}

	if mapping, exists := errorMappings[err]; exists {
		return mapping
	}
	return ErrorMapping{KeyInternalError, http.StatusInternalServerError}
}
func isBindingError(err error) bool {
	if err == nil {
		return false
	}

	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError
	var validationErr validator.ValidationErrors
	var invalidValidationErr *validator.InvalidValidationError

	return errors.As(err, &syntaxErr) ||
		errors.As(err, &unmarshalErr) ||
		errors.As(err, &validationErr) ||
		errors.As(err, &invalidValidationErr)
}
