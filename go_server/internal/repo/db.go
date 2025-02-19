package repo

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("record not found")
)

// DBConfig holds the DB configuration
type DBConfig struct {
	// Connection string
	Host string `mapstructure:"DB_HOST"`
	Port string `mapstructure:"DB_PORT"`
	Addr string `mapstructure:"DB_ADDR"`
	User string `mapstructure:"DB_USER"`
	Pass string `mapstructure:"DB_PASS"`
	Name string `mapstructure:"DB_NAME"`
	// Collection names
	TableUsers  string `mapstructure:"DB_TABLE_USERS"`
	TableComics string `mapstructure:"DB_TABLE_COMICS"`
	// Pool configuration
	MaxPoolSize     int           `mapstructure:"DB_MAX_POOL_SIZE" default:"100"`
	MinPoolSize     int           `mapstructure:"DB_MIN_POOL_SIZE" default:"0"`
	MaxConnIdleTime time.Duration `mapstructure:"DB_MAX_CONN_IDLE_TIME" default:"5m"`
	MaxConnLifeTime time.Duration `mapstructure:"DB_MAX_CONN_LIFE_TIME" default:"60m"`
	ConnectTimeout  time.Duration `mapstructure:"DB_CONN_TIMEOUT" default:"30s"`
	JaegerEndpoint  string
}

// Closable defines a common interface for closing database connections
type Closable interface {
	Close(ctx context.Context, duration time.Duration) error
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
