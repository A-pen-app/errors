# Errors

A unified error handling library for Go applications, providing structured error management for web APIs with context preservation and consistent HTTP responses.

## Features

- **Error Wrapping**: Add contextual data to errors without losing the original error chain
- **Unified HTTP Error Handling**: Automatic conversion of business logic errors to proper HTTP responses
- **Gin Integration**: Seamless integration with Gin web framework
- **OpenTelemetry Support**: Built-in request tracing and observability
- **Structured Logging**: Integration with logging package for consistent error logging
- **Validation Error Handling**: Automatic detection and handling of JSON binding and validation errors
- **Fallback Error Handling**: Undefined errors automatically mapped to 500 Internal Server Error
- **SQL Integration**: Built-in support for `sql.ErrNoRows` mapping to 404 Not Found

## Installation

```bash
go get github.com/A-pen-app/errors
```

## Usage

### Basic Error Wrapping

```go
import "github.com/A-pen-app/errors"

// Wrap errors with additional context
err := someOperation()
if err != nil {
    return errors.Wrap(err, "user_id", 123, "operation", "create_post")
}
```

### Gin Handler Integration

```go
import (
    "github.com/A-pen-app/errors"
    "github.com/gin-gonic/gin"
)

// Use Handle to automatically process errors
r.POST("/posts", errors.Handle(func(ctx *gin.Context) error {
    // Your handler logic here
    if someCondition {
        return errors.ErrorNotFound // Returns 404
    }
    
    if anotherCondition {
        return errors.Wrap(errors.ErrorWrongParams, "field", "title")
    }
    
    return nil // Success
}))
```

### Predefined Errors

The library provides common business logic errors:

```go
errors.ErrorNotFound         // "data not found" -> 404 NOT_FOUND
errors.ErrorNotAllowed       // "action not allowed" -> 403 ACTION_NOT_ALLOWED
errors.ErrorWrongParams      // "wrong parameters" -> 400 WRONG_PARAMETER
errors.ErrorPermissionDenied // "permission denied" -> 403 PERMISSION_DENIED
errors.ErrorInternalError    // "internal system error" -> 500 INTERNAL_ERROR
```

**Additional Supported Errors:**
- `sql.ErrNoRows` automatically mapped to `NOT_FOUND` (404)
- JSON binding/validation errors automatically mapped to `WRONG_PARAMETER` (400)
- Any undefined custom error defaults to `INTERNAL_ERROR` (500)

### Service-specific Errors

A service defines its own codes with `Define`, from package-level var declarations:

```go
var ErrTipSelf = errors.Define("TIP_SELF", http.StatusForbidden, "cannot tip yourself")
```

- `Handle` answers with that code and status even when the error is wrapped by `fmt.Errorf("%w")` or `errors.Wrap`
- The response message is always the defined message, so keep it safe to expose: inner `Wrap` data and `fmt.Errorf("%w")` text go only to the log, while the outermost `Wrap` data is still sent as `details`
- `Define` panics on a blank code, a code already in use (built-in or defined), or a status outside 4xx and 5xx
- Built-in errors still match only by identity: a built-in error wrapped by `fmt.Errorf("%w")` stays `INTERNAL_ERROR` (500)

The key-value pairs of the outermost `Wrap` are sent to the client as `details`. Pass that `Wrap` only what the client may see, and log the rest. For built-in and unmapped errors, inner `Wrap` data and `fmt.Errorf` text still show up in `message`.

### Error Response Format

All errors are returned as structured JSON:

```json
{
  "code": "NOT_FOUND",
  "message": "data not found",
  "details": {
    "user_id": 123,
    "operation": "create_post"
  },
  "request_id": "trace-id-from-opentelemetry"
}
```

## Code Structure and Error Flow

### Core Components

The error handling system consists of two main files:

#### `errors.go` - Error Processing Engine
- `Wrap()` function: Wraps errors with contextual data
- `Handle()` function: Gin middleware wrapper for automatic error handling  
- `handleError()` function: Core error processing that converts business errors to HTTP responses

#### `models.go` - Error Definitions and Mappings
- **Predefined Errors**: Standard business logic errors
- **Error Key Mapping**: Maps errors to API response codes
- **AppError Structure**: Wraps errors with additional context
- **HTTP Mapping Functions**: Convert errors to HTTP status codes

### Error Flow Process

1. **Error Creation**: Business logic creates errors using predefined constants or custom errors
2. **Error Wrapping**: `Wrap()` adds contextual data without losing the original error
3. **Error Handling**: `Handle()` catches returned errors from handlers
4. **Error Processing**: `handleError()` determines the appropriate HTTP response
5. **Response Generation**: Structured JSON response sent to client with logging

### Error Type Mappings

| Business Error | Error Key | HTTP Status |
|----------------|-----------|-------------|
| `ErrorNotFound` | `NOT_FOUND` | 404 |
| `ErrorNotAllowed` | `ACTION_NOT_ALLOWED` | 403 |
| `ErrorWrongParams` | `WRONG_PARAMETER` | 400 |
| `ErrorPermissionDenied` | `PERMISSION_DENIED` | 403 |
| `ErrorInternalError` | `INTERNAL_ERROR` | 500 |
| `sql.ErrNoRows` | `NOT_FOUND` | 404 |
| Binding Errors | `WRONG_PARAMETER` | 400 |
| Client Cancellations | `CLIENT_CLOSED_REQUEST` | 499 |
| **Any undefined error** | `INTERNAL_ERROR` | **500** |

### Error Handling Details

**Undefined Error Behavior:**
- Any error not explicitly defined in the error mapping will be treated as an `INTERNAL_ERROR`
- These undefined errors automatically receive HTTP status code `500` (Internal Server Error)
- The original error message is preserved and sent in the response
- All undefined errors are logged for debugging purposes

**Special Error Detection:**
- **JSON Binding Errors**: Automatically detected and mapped to `WRONG_PARAMETER` (400)
  - `json.SyntaxError`
  - `json.UnmarshalTypeError` 
  - `validator.ValidationErrors`
  - `validator.InvalidValidationError`
- **Client Cancellations**: Detected and mapped to `CLIENT_CLOSED_REQUEST` (499), logged at warn rather than error
  - Requires both a cancelled request context and an error that is the cancellation: `context.Canceled` through any wrapping, or SQLSTATE `57014`
  - A server-side `WithCancel`, a server deadline, or an unrelated error landing during a disconnect all stay `INTERNAL_ERROR` (500)
  - `IsClientCancellation(ctx, err)` is exported for handlers that respond without `Handle()`

**Error Context Preservation:**
- When using `Wrap()`, original error information is preserved
- Additional context data is stored separately and included in response details
- Error chain remains intact for proper error handling with `errors.As()` and `errors.Is()`