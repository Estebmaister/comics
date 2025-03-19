package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestLoggerMiddleware tests the LoggerMiddleware function
func TestLoggerMiddleware(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		query  string
		header string
		path   string
		body   []byte
		want   int
	}{
		{name: "Test logging", method: "GET", path: "/test", query: "?a=b", want: http.StatusOK},
		{name: "Test health", method: "GET", path: "/health", header: "req-xx21", want: http.StatusOK},
		{name: "Test body", method: "POST", path: "/body", body: []byte(`{"secret":"value"}`), want: http.StatusOK},
		{name: "Test error 400", method: "GET", path: "/none", want: http.StatusNotFound},
		{name: "Test error 500", method: "GET", path: "/panic", want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request and recorder
			req, _ := http.NewRequest(tt.method, tt.path+tt.query, bytes.NewReader(tt.body))
			if tt.header != "" {
				req.Header.Set("X-Request-ID", tt.header)
			}
			w := httptest.NewRecorder()
			r := gin.Default()

			// Call the middleware
			r.Use(LoggerMiddleware())
			if tt.want == http.StatusInternalServerError {
				r.Use(func() gin.HandlerFunc {
					return func(_ *gin.Context) {
						panic("simulated panic")
					}
				}())
			}

			// Create the handler and handle the request
			r.Handle(tt.method, tt.path, func(c *gin.Context) {
				c.JSON(tt.want, gin.H{"status": tt.want})
			})
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.want, w.Code)
		})
	}
}

// TestSetRequestID tests the SetRequestID function
func TestSetRequestID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		header string
		ctx    context.Context
	}{
		{name: "With header", header: "123"},
		{name: "Without header"},
		{name: "With context", ctx: context.WithValue(context.Background(), requestID{}, "456")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request and context
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.header != "" {
				req.Header.Set(keyHeaderRequestID, tt.header)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req
			if tt.ctx != nil {
				c.Request = c.Request.WithContext(tt.ctx)
			}

			// Call SetRequestID
			reqID := setRequestID(c)

			// Check the request ID
			if tt.header != "" {
				assert.Equal(t, tt.header, reqID)
			}
			if tt.ctx != nil {
				assert.Equal(t, tt.ctx.Value(requestID{}), reqID)
			}
			assert.NotEmpty(t, reqID)
			assert.Equal(t, reqID, c.Writer.Header().Get(keyHeaderRequestID))
		})
	}
}

// Test_sensitiveDataFilterToLog tests the sensitiveDataFilterToLog function
func Test_sensitiveDataFilterToLog(t *testing.T) {
	t.Parallel()

	logs := &logSink{}
	logger := zerolog.New(logs)

	tests := []struct {
		name string
		data []byte
		want string // Expected masked JSON string
	}{
		{
			name: "Mask sensitive data",
			data: []byte(`{"password":"secret", "api_key":"12345"}`),
			want: `{"password":"***PROTECTED***", "api_key":"***PROTECTED***"}`,
		},
		{
			name: "No sensitive data",
			data: []byte(`{"username":"user"}`),
			want: `{"username":"user"}`,
		},
		{
			name: "Invalid JSON",
			data: []byte(`not a json`),
			want: `{"level":"info"}`, // Expect the logger to remain unchanged
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sensitiveDataFilterToLog(tt.data, logger)
			got.Info().Msg("")

			// Compare the expected output with the actual output
			output, err := logs.Index(i)
			assert.NoError(t, err)
			assert.Contains(t, output, tt.want)
		})
	}
}

type logSink struct {
	logs []string
}

func (l *logSink) Write(p []byte) (n int, err error) {
	l.logs = append(l.logs, string(p))
	return len(p), nil
}

func (l *logSink) Index(i int) (string, error) {
	if i < 0 || i >= len(l.logs) {
		return "", fmt.Errorf("index out of bounds")
	}
	return l.logs[i], nil
}
