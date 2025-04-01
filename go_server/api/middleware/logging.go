package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"comics/internal/tracer"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	msg = "HTTP request"
)

var (
	// List of sensitive keys to mask
	sensitiveKeys      = []string{"password", "api_key", "token", "secret"}
	keyHeaderRequestID = "X-Request-ID"
	keyRequestID       = "request_id"
)

// RequestID
type requestID struct{}

// LoggerMiddleware logs HTTP requests
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Capture the request body
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body) // Read body
			// Reset body for downstream handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Extract request ID from headers or context
		reqID := setRequestID(c)

		// Add request ID and tracing info to logger
		ctx := c.Request.Context()
		logger := tracer.LoggerWithSpanFromCtx(ctx, log.Logger)
		logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str(keyRequestID, reqID)
		})

		// Set the logger on the context
		c.Request = c.Request.WithContext(logger.WithContext(ctx))

		defer func() {
			panicVal := recover()
			if panicVal != nil {
				err := fmt.Errorf("%v", panicVal)
				c.Error(err) // nolint:errcheck
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				c.Abort()
				defer panic(err)
			}

			path := c.Request.URL.Path
			status := c.Writer.Status()
			if status == http.StatusOK && (path == "/metrics" || path == "/health") {
				return // Skip logging for healthcheck
			}

			// Build log context
			raw := c.Request.URL.RawQuery
			if raw != "" {
				path = path + "?" + raw
			}

			// Build request log struct
			logger = sensitiveDataFilterToLog(bodyBytes, logger)
			logger = logger.With().Err(c.Errors.Last()).
				Str("client_ip", c.ClientIP()).
				Str("method", c.Request.Method).
				Str("path", path).
				Int("status", status).
				Dur("duration", time.Since(start)).
				Logger()

			// Log the request
			switch {
			case status >= 400 && status < 500:
				logger.Warn().Msg(msg)
			case status >= 500:
				logger.Error().Msg(msg)
			default:
				logger.Info().Msg(msg)
			}
		}()

		// Call the next middleware
		c.Next()
	}
}

// setRequestID returns the request ID from the headers or the gin request context.
// if it's not found, a new one is generated and added to the headers and to the context.
func setRequestID(c *gin.Context) string {
	// Get the request ID from the headers
	reqID := c.Request.Header.Get(keyHeaderRequestID)

	// Get the request ID from the context
	ctx := c.Request.Context()
	if reqID == "" {
		reqID, ctx = GetRequestID(ctx)
	}

	// Add request ID to the request and response-writter header
	c.Request.Header.Add(keyHeaderRequestID, reqID)
	c.Writer.Header().Add(keyHeaderRequestID, reqID)
	// Add request ID to the request context
	c.Request = c.Request.WithContext(ctx)
	return reqID
}

// GetRequestID returns the request ID from the context or creates one
// and adds it to the context.
func GetRequestID(ctx context.Context) (string, context.Context) {
	reqID := ctx.Value(requestID{})
	if reqID != nil {
		return reqID.(string), ctx
	}
	newReqID := xid.New().String()
	ctx = context.WithValue(ctx, requestID{}, newReqID)
	return newReqID, ctx
}

// sensitiveDataFilterToLog efficiently masks sensitive fields without full JSON parsing
func sensitiveDataFilterToLog(data []byte, logger zerolog.Logger) zerolog.Logger {
	if !gjson.ValidBytes(data) {
		return logger
	}
	for _, key := range sensitiveKeys {
		if gjson.GetBytes(data, key).Exists() {
			data, _ = sjson.SetBytes(data, key, "***PROTECTED***")
		}
	}
	data = bytes.ReplaceAll(data, []byte("\n"), []byte(""))
	return logger.With().RawJSON("body", data).Logger()
}
