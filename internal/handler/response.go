// Package handler provides unified response building utilities for API handlers.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/types"
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

// BadRequest sends a bad request error response
func BadRequest(c *gin.Context, message string) {
	Error(c, types.ErrInvalidRequest, message)
}

// Accepted sends a 202 Accepted response
func Accepted[T any](c *gin.Context, data T) {
	c.JSON(http.StatusAccepted, types.NewSuccessResponse(data, getRequestID(c)))
}
