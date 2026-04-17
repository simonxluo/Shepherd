// Package middleware provides HTTP middleware for the Shepherd API server.
// 这个包提供 HTTP 中间件用于 API 处理
package middleware

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.Errorf("Request error: %s", err.Error())

			switch e := err.Err.(type) {
			case *types.ErrorInfo:
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
				handler.ErrorWithDetails(c, types.ErrInternalError, "Internal server error", err.Error())
			}
		}
	}
}

func RecoveryMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])

				log.Errorf("Panic recovered: %v\nPath: %s\nMethod: %s\nStack:\n%s",
					r, c.Request.URL.Path, c.Request.Method, stack)
				handler.ErrorWithDetails(c, types.ErrInternalError, "Internal server error", "A panic occurred")
				c.Abort()
			}
		}()
		c.Next()
	}
}

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := ""

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

		if meta, exists := c.Get("responseMeta"); exists {
			if respMeta, ok := meta.(*types.ResponseMeta); ok {
				respMeta.Latency = latency.Milliseconds()
			}
		}
	}
}
