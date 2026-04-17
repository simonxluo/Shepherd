// Package types provides unified type definitions for the Shepherd system
// 这个包提供统一的类型定义，消除不同模块间的类型重复
package types

import (
	"fmt"
	"time"
)

// NodeState represents the unified state of a node or client
// 统一的节点状态，替代 NodeStatus 和 ClientStatus
type NodeState string

const (
	StateOffline  NodeState = "offline"
	StateOnline   NodeState = "online"
	StateBusy     NodeState = "busy"
	StateError    NodeState = "error"
	StateDegraded NodeState = "degraded"
	StateDisabled NodeState = "disabled"
)

// String returns the string representation of the state
func (s NodeState) String() string {
	return string(s)
}

// IsValid checks if the state is valid
func (s NodeState) IsValid() bool {
	switch s {
	case StateOffline, StateOnline, StateBusy, StateError, StateDegraded, StateDisabled:
		return true
	default:
		return false
	}
}

// ErrorCode represents unified error codes
// 统一的错误码定义
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

// ErrorInfo represents detailed error information
// 错误详细信息
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

// ResponseMeta represents metadata included in API responses
// API 响应元数据
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

// ApiResponse represents a unified API response format
// 统一的 API 响应格式，支持泛型类型
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

// PaginatedResponse represents a paginated API response
// 分页响应格式
type PaginatedResponse[T any] struct {
	Success  bool          `json:"success"`
	Data     []T           `json:"data,omitempty"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Error    *ErrorInfo    `json:"error,omitempty"`
	Metadata *ResponseMeta `json:"metadata,omitempty"`
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse[T any](data []T, total int64, page, pageSize int, requestID string) *PaginatedResponse[T] {
	return &PaginatedResponse[T]{
		Success:  true,
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Metadata: NewResponseMeta(requestID),
	}
}
