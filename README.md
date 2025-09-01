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