package repo

import (
	"comics/domain"
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("record not found")
)

// Client defines the interface for basic database clients
type Client interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Ping(ctx context.Context) error
}

// UserStore defines basic store needs
type UserStore interface {
	domain.UserStore
	Client() Client
}

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
