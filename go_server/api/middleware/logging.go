package middleware

import (
	"bytes"
	"context"
	"io"
	"time"

	"comics/internal/tracing"

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
		reqID := RequestID(c)

		// Add request ID and tracing info to logger
		ctx := c.Request.Context()
		logger := tracing.LoggerWithSpanFromCtx(ctx, log.Logger)
		logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str(keyRequestID, reqID)
		})

		// Set the logger on the context
		c.Request = c.Request.WithContext(logger.WithContext(ctx))

		// Call the next middleware
		c.Next()

		// Build log context
		status := c.Writer.Status()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		// Build request log struct
		logger = sensitiveDataFilterToLog(bodyBytes, logger)
		logger = logger.With().Err(c.Errors.Last()).
			// Str("request_id", c.Writer.Header().Get("Request-Id")).
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
	}
}

// RequestID returns the request ID from the headers or the gin request context.
// if it's not found, a new one is generated and added to the headers and to the context.
func RequestID(c *gin.Context) string {
	reqID := c.Request.Header.Get(keyHeaderRequestID)
	if reqID != "" {
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), requestID{}, reqID))
		return reqID
	}

	reqID = c.Request.Context().Value(requestID{}).(string)
	if reqID != "" {
		c.Writer.Header().Add(keyHeaderRequestID, reqID)
		return reqID
	}

	reqID = xid.New().String()
	c.Writer.Header().Add(keyHeaderRequestID, reqID)
	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), requestID{}, reqID))
	return reqID
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
	return logger.With().RawJSON("body", data).Logger()
}
