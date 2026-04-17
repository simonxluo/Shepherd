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
// 发送成功响应，携带数据
func Success[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, types.NewSuccessResponse(data, getRequestID(c)))
}

// SuccessWithMessage sends a successful API response with a message
// 发送成功响应，携带消息
func SuccessWithMessage(c *gin.Context, message string) {
	response := gin.H{"message": message}
	c.JSON(http.StatusOK, types.NewSuccessResponse(response, getRequestID(c)))
}

// Error sends an error API response
// 发送错误响应
func Error(c *gin.Context, code types.ErrorCode, message string) {
	statusCode := code.HTTPStatusCode()
	c.JSON(statusCode, types.NewErrorResponse(code, message, getRequestID(c)))
}

// ErrorWithDetails sends an error API response with details
// 发送带详情的错误响应
func ErrorWithDetails(c *gin.Context, code types.ErrorCode, message, details string) {
	statusCode := code.HTTPStatusCode()
	c.JSON(statusCode, types.NewErrorResponseWithDetails(code, message, details, getRequestID(c)))
}

// ValidationError sends a validation error response
// 发送验证错误响应
func ValidationError(c *gin.Context, err error) {
	Error(c, types.ErrInvalidRequest, err.Error())
}

// NotFound sends a not found error response
// 发送未找到错误响应
func NotFound(c *gin.Context, resource string) {
	Error(c, types.ErrNodeNotFound, resource+" not found")
}

// InternalError sends an internal server error response
// 发送内部服务器错误响应
func InternalError(c *gin.Context, err error) {
	ErrorWithDetails(c, types.ErrInternalError, "Internal server error", err.Error())
}

// BadRequest sends a bad request error response
// 发送错误请求响应
func BadRequest(c *gin.Context, message string) {
	Error(c, types.ErrInvalidRequest, message)
}

// Unauthorized sends an unauthorized error response
// 发送未授权错误响应
func Unauthorized(c *gin.Context, message string) {
	Error(c, types.ErrNotAuthenticated, message)
}

// Forbidden sends a forbidden error response
// 发送禁止访问错误响应
func Forbidden(c *gin.Context, message string) {
	Error(c, types.ErrPermissionDenied, message)
}

// Paginated sends a paginated API response
// 发送分页响应
func Paginated[T any](c *gin.Context, data []T, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, types.NewPaginatedResponse(data, total, page, pageSize, getRequestID(c)))
}

// Accepted sends an accepted response (for async operations)
// 发送已接受响应（用于异步操作）
func Accepted(c *gin.Context, message string) {
	response := gin.H{"message": message, "status": "accepted"}
	c.JSON(http.StatusAccepted, types.NewSuccessResponse(response, getRequestID(c)))
}

// NoContent sends a no content response
// 发送无内容响应
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
