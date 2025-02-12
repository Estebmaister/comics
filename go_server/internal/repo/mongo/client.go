package mongo

import (
	"context"
	"fmt"
	"time"

	"comics/internal/repo"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// DatabaseConfig provides configuration options for the MongoDB client
type DatabaseConfig struct {
	MaxPoolSize    uint64        `env:"mongo_max_pool_size" default:"100"`
	MinPoolSize    uint64        `env:"mongo_min_pool_size" default:"0"`
	MaxConnIdle    time.Duration `env:"mongo_max_conn_idle" default:"5m"`
	ConnectTimeout time.Duration `env:"mongo_connect_timeout" default:"30s"`
	URI            string        `env:"mongo_uri" default:"mongodb://localhost:27017/comics"`
}

// Client interface for advanced database client operations
type Client interface {
	repo.Client

	Metrics() repo.MetricsCollector
	Database(dbName string) Database

	// Session management
	StartSession() (*mongo.Session, error)

	// Health
	IsConnected() bool

	// Readiness and liveness probes
	WaitForConnection(timeout time.Duration) error
}

// Implementation example (partial)
type mongoClient struct {
	cl      *mongo.Client
	metrics repo.Metrics
}

// NewMongoClient creates a new MongoDB client with advanced configuration
func NewMongoClient(ctx context.Context, cfg *DatabaseConfig, uri string) (*mongoClient, error) {
	// Validate configuration
	if cfg == nil {
		cfg = &DatabaseConfig{
			MaxPoolSize:    100,
			MinPoolSize:    0,
			MaxConnIdle:    5 * time.Minute,
			ConnectTimeout: 30 * time.Second,
			URI:            uri,
		}
	}

	// Prepare client options
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize)).
		SetMaxConnIdleTime(cfg.MaxConnIdle).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetCompressors([]string{"zstd", "zlib", "snappy"})

	// Record connection start time
	startTime := time.Now()

	// Create MongoDB client
	cl, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	// Create mongoClient wrapper
	mongoClient := &mongoClient{
		cl:      cl,
		metrics: repo.Metrics{},
	}

	// Record connection metrics
	mongoClient.Metrics().RecordConnection(time.Since(startTime), err)

	return mongoClient, nil
}

// Connect establishes a connection to the MongoDB server
func (mc *mongoClient) Connect(ctx context.Context) error {
	// Record connection start time
	startTime := time.Now()

	// Attempt to ping the server to verify connection
	err := mc.Ping(ctx)

	// Record connection metrics
	mc.Metrics().RecordConnection(time.Since(startTime), err)

	// If connection fails, return the error
	if err != nil {
		return fmt.Errorf("failed to establish MongoDB connection: %w", err)
	}

	return nil
}

// Ping checks the connection to the MongoDB server
func (mc *mongoClient) Ping(ctx context.Context) error {
	// Record query start time
	startTime := time.Now()

	// Attempt to ping
	err := mc.cl.Ping(ctx, nil)

	// Record query metrics
	mc.Metrics().RecordQuery(time.Since(startTime), err)

	return err
}

// Database returns a specific database from the client
func (mc *mongoClient) Database(dbName string) Database {
	return &mongoDatabase{db: mc.cl.Database(dbName)}
}

// Disconnect closes the MongoDB connection
func (mc *mongoClient) Disconnect(ctx context.Context) error {
	// Attempt to disconnect
	err := mc.cl.Disconnect(ctx)
	if err != nil {
		return err
	}

	// Release the connection
	mc.Metrics().ReleaseConnection()
	return nil
}

// IsConnected checks if the client is connected to the database
func (mc *mongoClient) IsConnected() bool {
	// Ping to check actual connection status
	err := mc.Ping(context.Background())
	return err == nil
}

// StartSession begins a new MongoDB session
func (mc *mongoClient) StartSession() (*mongo.Session, error) {
	return mc.cl.StartSession()
}

// WaitForConnection waits until a connection is established
func (mc *mongoClient) WaitForConnection(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if mc.IsConnected() {
				return nil
			}
		}
	}
}

// Metrics returns a MetricsCollector instance
func (mc *mongoClient) Metrics() repo.MetricsCollector {
	return &mc.metrics
}
