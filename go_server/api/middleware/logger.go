package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	msg = "HTTP request"
)

var (
	// List of sensitive keys to mask
	sensitiveKeys = []string{"password", "api_key", "token", "secret"}
)

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
		c.Next()

		// Apply sensitive data filter
		duration := time.Since(start)
		// requestID := c.Writer.Header().Get("Request-Id")
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}
		err := c.Errors.Last()

		// Build log
		logCtx := log.With()
		if gjson.ValidBytes(bodyBytes) { // Ensure RawJSON only gets valid JSON
			filteredBody := SensitiveDataFilter(bodyBytes)
			logCtx = log.With().RawJSON("body", filteredBody)
		}
		log := logCtx.Err(err).
			// Str("request_id", requestID).
			Str("client_ip", clientIP).
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("duration", duration).Logger()

		// Log the request
		switch {
		case status >= 400 && status < 500:
			log.Warn().Msg(msg)
		case status >= 500:
			log.Error().Msg(msg)
		default:
			log.Info().Msg(msg)
		}
	}
}

// SensitiveDataFilter efficiently masks sensitive fields without full JSON parsing
func SensitiveDataFilter(data []byte) []byte {
	filteredData := data
	for _, key := range sensitiveKeys {
		if gjson.GetBytes(filteredData, key).Exists() {
			filteredData, _ = sjson.SetBytes(filteredData, key, "***PROTECTED***")
		}
	}
	return filteredData
}
