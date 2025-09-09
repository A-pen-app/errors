# Errors

A unified error handling library for Go applications, providing structured error management for web APIs with context preservation and consistent HTTP responses.

## Features

- **Error Wrapping**: Add contextual data to errors without losing the original error chain
- **Unified HTTP Error Handling**: Automatic conversion of business logic errors to proper HTTP responses
- **Gin Integration**: Seamless integration with Gin web framework
- **OpenTelemetry Support**: Built-in request tracing and observability
- **Structured Logging**: Integration with logging package for consistent error logging
- **Validation Error Handling**: Automatic detection and handling of JSON binding and validation errors

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
errors.ErrorNotFound         // "data not found" -> 404
errors.ErrorNotAllowed       // "action not allowed" -> 403  
errors.ErrorWrongParams      // "wrong parameters" -> 400
errors.ErrorPermissionDenied // "permission denied" -> 403
errors.ErrorInternalError    // "internal system error" -> 500
```

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