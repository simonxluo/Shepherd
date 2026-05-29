// Package types provides unified type definitions for the Shepherd system,
// eliminating type duplication across modules.
package types

import (
	"fmt"
	"time"
)

// NodeState represents the unified state of a node or client,
// replacing the separate NodeStatus and ClientStatus types.
type NodeState string

const (
	StateOffline NodeState = "offline"
	StateOnline  NodeState = "online"
	StateBusy    NodeState = "busy"
	StateError   NodeState = "error"
)

// String returns the string representation of the state
func (s NodeState) String() string {
	return string(s)
}

// ErrorCode represents unified error codes for the API layer.
type ErrorCode string

const (
	ErrNodeNotFound      ErrorCode = "NODE_NOT_FOUND"
	ErrInvalidRequest    ErrorCode = "INVALID_REQUEST"
	ErrConflict          ErrorCode = "CONFLICT"
	ErrTimeout           ErrorCode = "TIMEOUT"
	ErrCommandFailed     ErrorCode = "COMMAND_FAILED"
	ErrNotAuthenticated  ErrorCode = "NOT_AUTHENTICATED"
	ErrPermissionDenied  ErrorCode = "PERMISSION_DENIED"
	ErrResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrInternalError     ErrorCode = "INTERNAL_ERROR"
)

// String returns the string representation of the error code
func (e ErrorCode) String() string {
	return string(e)
}

// HTTPStatusCode returns the appropriate HTTP status code for the error
func (e ErrorCode) HTTPStatusCode() int {
	switch e {
	case ErrNodeNotFound:
		return 404
	case ErrInvalidRequest:
		return 400
	case ErrConflict:
		return 409
	case ErrTimeout:
		return 408
	case ErrNotAuthenticated:
		return 401
	case ErrPermissionDenied:
		return 403
	case ErrResourceExhausted:
		return 429
	default:
		return 500
	}
}

// ErrorInfo represents detailed error information for API responses.
type ErrorInfo struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// Error returns a formatted error message
func (e *ErrorInfo) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ResponseMeta represents metadata included in API responses.
type ResponseMeta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"requestId"`
	Latency   int64  `json:"latency,omitempty"` // milliseconds
}

// NewResponseMeta creates a new ResponseMeta with current timestamp
func NewResponseMeta(requestID string) *ResponseMeta {
	return &ResponseMeta{
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: requestID,
	}
}

// ApiResponse represents a unified API response format with generic type support.
type ApiResponse[T any] struct {
	Success  bool          `json:"success"`
	Data     T             `json:"data,omitempty"`
	Error    *ErrorInfo    `json:"error,omitempty"`
	Metadata *ResponseMeta `json:"metadata,omitempty"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse[T any](data T, requestID string) *ApiResponse[T] {
	return &ApiResponse[T]{
		Success:  true,
		Data:     data,
		Metadata: NewResponseMeta(requestID),
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(code ErrorCode, message string, requestID string) *ApiResponse[struct{}] {
	return &ApiResponse[struct{}]{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		Metadata: NewResponseMeta(requestID),
	}
}

// NewErrorResponseWithDetails creates an error API response with details
func NewErrorResponseWithDetails(code ErrorCode, message, details string, requestID string) *ApiResponse[struct{}] {
	return &ApiResponse[struct{}]{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Metadata: NewResponseMeta(requestID),
	}
}
