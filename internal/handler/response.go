// Package api provides unified response building utilities for API handlers
// 这个包提供统一的响应构建工具，用于 API 处理器
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
)

// getRequestID gets the request ID from context, returns "unknown" if not set
func getRequestID(c *gin.Context) string {
	if requestID := c.GetString("requestId"); requestID != "" {
		return requestID
	}
	return "unknown"
}

// Success sends a successful API response with data
func Success[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(data, getRequestID(c)))
}

// SuccessWithMessage sends a successful API response with a message
func SuccessWithMessage(c *gin.Context, message string) {
	response := gin.H{"message": message}
	c.JSON(http.StatusOK, types.NewSuccessResponse(response, getRequestID(c)))
}

// Error sends an error API response
func Error(c *gin.Context, code types.ErrorCode, message string) {
	statusCode := code.HTTPStatusCode()
	c.JSON(statusCode, types.NewErrorResponse(code, message, getRequestID(c)))
}

// ErrorWithDetails sends an error API response with details
func ErrorWithDetails(c *gin.Context, code types.ErrorCode, message, details string) {
	statusCode := code.HTTPStatusCode()
	c.JSON(statusCode, types.NewErrorResponseWithDetails(code, message, details, getRequestID(c)))
}

// ValidationError sends a validation error response
func ValidationError(c *gin.Context, err error) {
	Error(c, types.ErrInvalidRequest, err.Error())
}

// NotFound sends a not found error response
func NotFound(c *gin.Context, resource string) {
	Error(c, types.ErrNodeNotFound, resource+" not found")
}

// InternalError sends an internal server error response
func InternalError(c *gin.Context, err error) {
	ErrorWithDetails(c, types.ErrInternalError, "Internal server error", err.Error())
}

// BadRequest sends a bad request error response
func BadRequest(c *gin.Context, message string) {
	Error(c, types.ErrInvalidRequest, message)
}

// Unauthorized sends an unauthorized error response
func Unauthorized(c *gin.Context, message string) {
	Error(c, types.ErrNotAuthenticated, message)
}

// Forbidden sends a forbidden error response
func Forbidden(c *gin.Context, message string) {
	Error(c, types.ErrPermissionDenied, message)
}

// Paginated sends a paginated API response
func Paginated[T any](c *gin.Context, data []T, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, types.NewPaginatedResponse(data, total, page, pageSize, getRequestID(c)))
}

// Accepted sends an accepted response (for async operations)
func Accepted(c *gin.Context, message string) {
	response := gin.H{"message": message, "status": "accepted"}
	c.JSON(http.StatusAccepted, types.NewSuccessResponse(response, getRequestID(c)))
}

// NoContent sends a no content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
