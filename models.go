package errors

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

type ErrorType string

const (
	ErrorTypePost  ErrorType = "post"
	ErrorTypeLogin ErrorType = "login"
)

type ErrorCode string

const (
	KeyNotFound            ErrorCode = "NOT_FOUND"
	KeyNotAllowed          ErrorCode = "ACTION_NOT_ALLOWED"
	KeyWrongParams         ErrorCode = "WRONG_PARAMETER"
	KeyUnauthorized        ErrorCode = "UNAUTHORIZED"
	KeyPermissionDenied    ErrorCode = "PERMISSION_DENIED"
	KeyUnprocessableEntity ErrorCode = "UNPROCESSABLE_ENTITY"
	KeyInternalError       ErrorCode = "INTERNAL_ERROR"
	KeyDuplicateEntry      ErrorCode = "DUPLICATE_ENTRY"
	KeyInsufficientQuota   ErrorCode = "INSUFFICIENT_QUOTA"
	KeyUserNotVerified     ErrorCode = "USER_NOT_VERIFIED"
	KeyUnsupported         ErrorCode = "UNSUPPORTED"
	KeyConflict            ErrorCode = "CONFLICT"
	KeyTooManyRequests     ErrorCode = "TOO_MANY_REQUESTS"
	KeyClientClosed        ErrorCode = "CLIENT_CLOSED_REQUEST"
)

// clientClosedRequest is nginx's 499: the client went away before the response.
const clientClosedRequest = 499

// sqlStateQueryCanceled is Postgres' code for a query cancelled on request.
const sqlStateQueryCanceled = "57014"

var (
	ErrorNotFound            = errors.New("data not found")
	ErrorNotAllowed          = errors.New("action not allowed")
	ErrorWrongParams         = errors.New("wrong parameters")
	ErrorUnauthorized        = errors.New("unauthorized")
	ErrorPermissionDenied    = errors.New("permission denied")
	ErrorUnprocessableEntity = errors.New("unprocessable entity")
	ErrorInternalError       = errors.New("internal system error")
	ErrorDuplicateEntry      = errors.New("duplicate entry")
	ErrorInsufficientQuota   = errors.New("insufficient quota")
	ErrorUserNotVerified     = errors.New("user not verified")
	ErrorUnsupported         = errors.New("unsupported")
	ErrorConflict            = errors.New("conflict")
	ErrorTooManyRequests     = errors.New("too many requests")
)

type ErrorMapping struct {
	Code       ErrorCode
	StatusCode int
}

var errorMappings = map[error]ErrorMapping{
	ErrorNotFound:            {KeyNotFound, http.StatusNotFound},
	ErrorNotAllowed:          {KeyNotAllowed, http.StatusForbidden},
	ErrorWrongParams:         {KeyWrongParams, http.StatusBadRequest},
	ErrorUnauthorized:        {KeyUnauthorized, http.StatusUnauthorized},
	ErrorPermissionDenied:    {KeyPermissionDenied, http.StatusForbidden},
	ErrorUnprocessableEntity: {KeyUnprocessableEntity, http.StatusUnprocessableEntity},
	ErrorInternalError:       {KeyInternalError, http.StatusInternalServerError},
	ErrorDuplicateEntry:      {KeyDuplicateEntry, http.StatusConflict},
	ErrorInsufficientQuota:   {KeyInsufficientQuota, http.StatusPaymentRequired},
	ErrorUserNotVerified:     {KeyUserNotVerified, http.StatusForbidden},
	ErrorUnsupported:         {KeyUnsupported, http.StatusUnprocessableEntity},
	ErrorConflict:            {KeyConflict, http.StatusConflict},
	ErrorTooManyRequests:     {KeyTooManyRequests, http.StatusTooManyRequests},
	sql.ErrNoRows:            {KeyNotFound, http.StatusNotFound},
}

// definedError is a service-specific error that carries its own response mapping.
type definedError struct {
	mapping ErrorMapping
	message string
}

func (e *definedError) Error() string {
	return e.message
}

var (
	definedCodesMu sync.Mutex
	// definedCodes holds every code a response can carry, so Define cannot reuse one.
	definedCodes = map[ErrorCode]struct{}{
		KeyNotFound:            {},
		KeyNotAllowed:          {},
		KeyWrongParams:         {},
		KeyUnauthorized:        {},
		KeyPermissionDenied:    {},
		KeyUnprocessableEntity: {},
		KeyInternalError:       {},
		KeyDuplicateEntry:      {},
		KeyInsufficientQuota:   {},
		KeyUserNotVerified:     {},
		KeyUnsupported:         {},
		KeyConflict:            {},
		KeyTooManyRequests:     {},
		KeyClientClosed:        {},
	}
)

// Define creates a service-specific error that Handle answers with the given code and HTTP status.
// The message is sent to the client, so it must be safe to expose.
// Call it from package-level var declarations: it panics on a blank or already used code,
// or a status outside 4xx and 5xx.
func Define(code ErrorCode, status int, message string) error {
	if strings.TrimSpace(string(code)) == "" {
		panic("errors: Define with a blank code")
	}
	if status < http.StatusBadRequest || status > 599 {
		panic(fmt.Sprintf("errors: Define %s with status %d, want 4xx or 5xx", code, status))
	}

	definedCodesMu.Lock()
	defer definedCodesMu.Unlock()
	if _, exists := definedCodes[code]; exists {
		panic(fmt.Sprintf("errors: Define %s, code already in use", code))
	}
	definedCodes[code] = struct{}{}

	return &definedError{
		mapping: ErrorMapping{code, status},
		message: message,
	}
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

	var defined *definedError
	if errors.As(err, &defined) {
		return defined.mapping
	}

	if mapping, exists := errorMappings[err]; exists {
		return mapping
	}
	return ErrorMapping{KeyInternalError, http.StatusInternalServerError}
}

// sqlStater is satisfied by lib/pq's and pgx's error types, so matching a
// SQLSTATE needs no driver dependency.
type sqlStater interface {
	SQLState() string
}

// IsClientCancellation reports whether the client cancelled the request itself:
// the request context is cancelled and the error is that cancellation.
func IsClientCancellation(ctx context.Context, err error) bool {
	if ctx.Err() != context.Canceled {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}

	var state sqlStater
	return errors.As(err, &state) && state.SQLState() == sqlStateQueryCanceled
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
