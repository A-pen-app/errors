package errors

import (
	"errors"
	
	"github.com/A-pen-app/logging"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Wrap wraps an error with additional context data.
// Works for both business logic errors and system errors.
func Wrap(err error, keyValues ...any) error {
	if err == nil {
		return nil
	}
	data := parseKeyValues(keyValues)
	return &AppError{
		cause: err,
		data:  data,
	}
}

// HandlerFunc defines a Gin handler function that returns an error.
type HandlerFunc func(*gin.Context) error

// Handle wraps a HandlerFunc to automatically handle errors using the unified error handling system.
func Handle(fn HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := fn(ctx); err != nil {
			handleError(ctx, err)
		}
	}
}

// handleError processes an error and sends a structured JSON response to the client.
// It separates internal error context (logged) from external API messages (sent to frontend).
func handleError(ctx *gin.Context, err error) {
	if err == nil {
		return
	}

	// Extract actual error for key and status determination
	var actualErr error
	var details map[string]any

	var appErr *AppError
	if errors.As(err, &appErr) {
		actualErr = appErr.Unwrap()
		details = appErr.Data()
	} else {
		actualErr = err
		details = make(map[string]any)
	}

	// Unified processing
	errorKey := string(getKey(actualErr))
	status := getHTTPStatusCode(actualErr)
	logging.Error(ctx.Request.Context(), err.Error())

	// Get request ID for tracing
	requestID := ""
	if spanCtx := trace.SpanContextFromContext(ctx.Request.Context()); spanCtx.IsValid() && spanCtx.TraceID().IsValid() {
		requestID = spanCtx.TraceID().String()
	}

	// Send error response
	ctx.AbortWithStatusJSON(status, httpError{
		Code:      errorKey,
		Message:   actualErr.Error(),
		Details:   details,
		RequestID: requestID,
	})
}
