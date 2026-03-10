// Package api provides HTTP middleware for API handling
// 这个包提供 HTTP 中间件用于 API 处理
package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/types"
)

// RequestID middleware adds a unique request ID to each request
// RequestID 中间件为每个请求添加唯一 ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// ErrorHandler middleware handles errors that occur during request processing
// ErrorHandler 中间件处理请求处理过程中发生的错误
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.Errorf("Request error: %s", err.Error())

			// Try to determine the error type and respond appropriately
			switch e := err.Err.(type) {
			case *types.ErrorInfo:
				// Use the error info directly
				requestID := c.GetString("requestId")
				if requestID == "" {
					requestID = "unknown"
				}
				c.JSON(e.Code.HTTPStatusCode(), types.NewErrorResponse(
					e.Code,
					e.Message,
					requestID,
				))
			default:
				// Generic error response
				ErrorWithDetails(c, types.ErrInternalError, "Internal server error", err.Error())
			}
		}
	}
}

// RecoveryMiddleware handles panics and converts them to errors
// RecoveryMiddleware 处理 panic 并转换为错误
func RecoveryMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 获取调用栈信息
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])

				log.Errorf("Panic recovered: %v\nPath: %s\nMethod: %s\nStack:\n%s",
					r, c.Request.URL.Path, c.Request.Method, stack)
				ErrorWithDetails(c, types.ErrInternalError, "Internal server error", "A panic occurred")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CORSMiddleware adds CORS headers for cross-origin requests
// CORSMiddleware 添加 CORS 头用于跨域请求
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := ""

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" {
				allowed = true
				allowOrigin = "*"
				break
			}
			if allowedOrigin == origin {
				allowed = true
				allowOrigin = origin
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggerMiddleware logs request information
// LoggerMiddleware 记录请求信息
func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		log.Infof("API Request: method=%s path=%s status=%d latency=%v",
			c.Request.Method,
			path+"?"+raw,
			statusCode,
			latency,
		)

		// Add latency to response metadata if present
		if meta, exists := c.Get("responseMeta"); exists {
			if respMeta, ok := meta.(*types.ResponseMeta); ok {
				respMeta.Latency = latency.Milliseconds()
			}
		}
	}
}
