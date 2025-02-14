package repo

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("record not found")
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordQuery(duration time.Duration, err error)
	RecordRetry(success bool)
	RecordConnection(connectionTime time.Duration, err error)
	ReleaseConnection()
	Reset()
	GetStats() map[string]string
}

// TracingProvider defines the interface for distributed tracing
type TracingProvider interface {
	StartSpan(ctx context.Context, operation string) (context.Context, TraceSpan)
}

// TraceSpan represents a single operation span
type TraceSpan interface {
	End()
	SetError(err error)
	SetTag(key string, value interface{})
}
