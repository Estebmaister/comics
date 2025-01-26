package db

import (
	"context"

	pb "comics/pkg/pb"
)

// ComicsStore defines the interface for comic operations
type ComicsStore interface {
	// Comic operations
	CreateComic(ctx context.Context, comic *pb.Comic) error
	UpdateComic(ctx context.Context, comic *pb.Comic) error
	DeleteComic(ctx context.Context, id int32) error
	GetComicById(ctx context.Context, id int32) (*pb.Comic, error)
	GetComics(ctx context.Context, page, pageSize int, trackedOnly, uncheckedOnly bool) ([]*pb.Comic, int, error)

	// Metrics and health
	GetMetrics() *Metrics
	Ping(ctx context.Context) error
	Close() error
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordQuery(duration float64, operation string, err error)
	RecordRetry(operation string, success bool)
	GetMetrics() *Metrics
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
