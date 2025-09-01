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

type ErrorKey string

const (
	KeyNotFound         ErrorKey = "NOT_FOUND"
	KeyNotAllowed       ErrorKey = "ACTION_NOT_ALLOWED"
	KeyWrongParams      ErrorKey = "WRONG_PARAMETER"
	KeyPermissionDenied ErrorKey = "PERMISSION_DENIED"
	KeyInternalError    ErrorKey = "INTERNAL_ERROR"
)

var (
	ErrorNotFound         = errors.New("data not found")
	ErrorNotAllowed       = errors.New("action not allowed")
	ErrorWrongParams      = errors.New("wrong parameters")
	ErrorPermissionDenied = errors.New("permission denied")
	ErrorInternalError    = errors.New("internal system error")
)

var errorKeyMap = map[error]ErrorKey{
	ErrorNotFound:         KeyNotFound,
	ErrorNotAllowed:       KeyNotAllowed,
	ErrorWrongParams:      KeyWrongParams,
	ErrorPermissionDenied: KeyPermissionDenied,
	ErrorInternalError:    KeyInternalError,
	sql.ErrNoRows:         KeyNotFound,
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

type httpError struct {
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

// getKey returns the API key for a given error, or KeyInternalError if not found.
func getKey(err error) ErrorKey {
	// Check for binding errors first
	if isBindingError(err) {
		return KeyWrongParams
	}

	if key, exists := errorKeyMap[err]; exists {
		return key
	}
	return KeyInternalError
}

func getHTTPStatusCode(err error) int {
	switch {
	// 400 Bad Request - includes binding errors
	case errors.Is(err, ErrorWrongParams),
		isBindingError(err):
		return http.StatusBadRequest

	// 403 Forbidden
	case errors.Is(err, ErrorNotAllowed),
		errors.Is(err, ErrorPermissionDenied):
		return http.StatusForbidden

	// 404 Not Found
	case errors.Is(err, ErrorNotFound),
		errors.Is(err, sql.ErrNoRows):
		return http.StatusNotFound

	// 500 Internal Server Error
	default:
		return http.StatusInternalServerError
	}
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
