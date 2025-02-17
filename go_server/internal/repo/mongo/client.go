package mongo

import (
	"context"
	"fmt"
	"time"

	"comics/internal/repo"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client interface for advanced database client operations
type Client interface {
	Disconnect(ctx context.Context, duration time.Duration) error
	Ping(ctx context.Context) error

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
	metrics repo.MetricsCollector
}

// newMongoClient creates a new MongoDB client with advanced configuration
func newMongoClient(ctx context.Context, cfg *repo.DBConfig) (*mongoClient, error) {
	// Validate configuration
	if cfg == nil || cfg.Addr == "" {
		cfg = &repo.DBConfig{
			Addr:           "mongodb://localhost:27017",
			MaxPoolSize:    100,
			MinPoolSize:    0,
			MaxConnIdle:    5 * time.Minute,
			ConnectTimeout: 30 * time.Second,
		}
	}

	uri := fmt.Sprintf(
		"mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=Sandbox",
		cfg.User, cfg.Pass, cfg.Addr)

	if cfg.User == "" || cfg.Pass == "" {
		uri = cfg.Addr
	}

	// Create metrics collector
	metrics := repo.Metrics{}

	// Prepare client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize)).
		SetMaxConnIdleTime(cfg.MaxConnIdle).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetPoolMonitor(newPoolMonitor(&metrics)).
		SetCompressors([]string{"zstd", "zlib", "snappy"})

	// Record connection start time
	startTime := time.Now()

	// Create MongoDB client
	cl, err := mongo.Connect(clientOptions)

	// Create mongoClient wrapper
	mongoClient := &mongoClient{
		cl:      cl,
		metrics: &metrics,
	}

	// Record connection metrics
	mongoClient.Metrics().RecordConnection(time.Since(startTime), err)

	return mongoClient, err
}

// newPoolMonitor creates a new PoolMonitor instance
func newPoolMonitor(metrics repo.MetricsCollector) *event.PoolMonitor {
	return &event.PoolMonitor{
		Event: func(evt *event.PoolEvent) {
			switch evt.Type {
			case event.ConnectionCreated:
				metrics.RecordConnection(evt.Duration, evt.Error)
			case event.ConnectionClosed:
				metrics.CloseConnection(evt.Duration, evt.Error)
			case event.ConnectionCheckOutFailed:
				metrics.RecordConnection(0, evt.Error)
			case event.ConnectionCheckedOut:
				metrics.RetrieveConnection()
			case event.ConnectionCheckedIn:
				metrics.ReleaseConnection()
			}
		},
	}
}

// Metrics returns a MetricsCollector instance
func (mc *mongoClient) Metrics() repo.MetricsCollector {
	return mc.metrics
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
func (mc *mongoClient) Disconnect(ctx context.Context, duration time.Duration) error {
	err := mc.cl.Disconnect(ctx)
	mc.Metrics().CloseConnection(duration, err)
	return err
}

// IsConnected checks if the client is connected to the database by pinging it
func (mc *mongoClient) IsConnected() bool {
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
