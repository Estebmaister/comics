package repo

import (
	"comics/internal/tracing"
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("record not found")
	ErrInvalidPageParams = errors.New("invalid page parameters")
	ErrInvalidArgument   = errors.New("invalid argument")
)

// DBConfig holds the DB configuration
type DBConfig struct {
	// Tracing
	tracing.TracerConfig
	// Connection string
	Host string `mapstructure:"DB_HOST" default:"localhost"`
	Port int    `mapstructure:"DB_PORT" default:"5432"`
	Addr string `mapstructure:"DB_ADDR" default:""`
	User string `mapstructure:"DB_USER" default:"user"`
	Pass string `mapstructure:"DB_PASS" default:"pass"`
	Name string `mapstructure:"DB_NAME" default:"comics"`
	// Collection names
	TableUsers  string `mapstructure:"DB_TABLE_USERS" default:"users"`
	TableComics string `mapstructure:"DB_TABLE_COMICS" default:"comics"`
	// Connection configuration
	MaxPoolSize     int           `mapstructure:"DB_MAX_POOL_SIZE" default:"100"`
	MinPoolSize     int           `mapstructure:"DB_MIN_POOL_SIZE" default:"0"`
	MaxConnIdleTime time.Duration `mapstructure:"DB_MAX_CONN_IDLE_TIME" default:"5m"`
	MaxConnLifeTime time.Duration `mapstructure:"DB_MAX_CONN_LIFE_TIME" default:"60m"`
	ConnectTimeout  time.Duration `mapstructure:"DB_CONN_TIMEOUT" default:"30s"`
	BackoffTimeout  time.Duration `mapstructure:"DB_BACKOFF_TIMEOUT" default:"15s"`
}

// Closable defines a common interface for closing database connections
type Closable interface {
	Close(ctx context.Context) error
}

// TracingProvider defines the interface for distributed tracing
type TracingProvider interface {
	StartSpan(ctx context.Context, operation string) (context.Context, TraceSpan)
}

// TraceSpan represents a single operation span
type TraceSpan interface {
	End()
	SetError(err error)
	SetTag(key string, value any)
}
