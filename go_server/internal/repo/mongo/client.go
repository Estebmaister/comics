package mongo

import (
	"context"
	"fmt"
	"time"

	"comics/internal/metrics"
	"comics/internal/repo"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client interface for advanced database client operations
type Client interface {
	Disconnect(ctx context.Context) error
	Ping(ctx context.Context) error

	Database(dbName string) Database

	// Session management
	StartSession() (*mongo.Session, error)
	UseSession(ctx context.Context, fn func(ctx context.Context) error) error
	UseSessionWithOptions(ctx context.Context, opts *options.SessionOptionsBuilder,
		fn func(ctx context.Context) error) error

	// Health
	IsConnected() bool

	// Readiness and liveness probes
	WaitForConnection(timeout time.Duration) error
}

// Implementation example (partial)
type mongoClient struct {
	cl *mongo.Client
}

// newMongoClient creates a new MongoDB client with advanced configuration
func newMongoClient(_ context.Context, cfg *repo.DBConfig, dbMetrics *metrics.Metrics,
) (*mongoClient, error) {
	// Prepare connection URI
	uri := fmt.Sprintf(
		"mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=Sandbox",
		cfg.User, cfg.Pass, cfg.Addr)

	if cfg.User == "" || cfg.Pass == "" {
		uri = cfg.Addr
	}

	// Prepare client options
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(uint64(cfg.MaxPoolSize)).
		SetMinPoolSize(uint64(cfg.MinPoolSize)).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetPoolMonitor(newPoolMonitor(dbMetrics)).
		SetCompressors([]string{"zstd", "zlib", "snappy"})

	// Create MongoDB client
	cl, err := mongo.Connect(clientOptions)

	// Create mongoClient wrapper
	mongoClient := &mongoClient{
		cl: cl,
	}

	return mongoClient, err
}

// newPoolMonitor creates a new PoolMonitor instance
func newPoolMonitor(metrics *metrics.Metrics) *event.PoolMonitor {
	if metrics == nil {
		return nil
	}
	return &event.PoolMonitor{
		Event: func(evt *event.PoolEvent) {
			switch evt.Type {
			case event.ConnectionCreated:
				metrics.RecordConnection(evt.Error)
			case event.ConnectionClosed:
				metrics.CloseConnection(evt.Error)
			case event.ConnectionCheckOutFailed:
				metrics.RecordConnection(evt.Error)
			case event.ConnectionCheckedOut:
				metrics.RetrieveConnection()
			case event.ConnectionCheckedIn:
				metrics.ReleaseConnection()
			}
		},
	}
}

// Ping checks the connection to the MongoDB server
func (mc *mongoClient) Ping(ctx context.Context) error {
	return mc.cl.Ping(ctx, nil)
}

// Database returns a specific database from the client
func (mc *mongoClient) Database(dbName string) Database {
	return &mongoDatabase{db: mc.cl.Database(dbName)}
}

// Disconnect closes the MongoDB connection
func (mc *mongoClient) Disconnect(ctx context.Context) error { return mc.cl.Disconnect(ctx) }

// IsConnected checks if the client is connected to the database by pinging it
func (mc *mongoClient) IsConnected() bool {
	err := mc.Ping(context.Background())
	return err == nil
}

// StartSession begins a new MongoDB session
func (mc *mongoClient) StartSession() (*mongo.Session, error) { return mc.cl.StartSession() }

// UseSession uses a session to execute a function
func (mc *mongoClient) UseSession(ctx context.Context, fn func(ctx context.Context) error) error {
	return mc.cl.UseSession(ctx, fn)
}

// UseSessionWithOptions uses a session with options to execute a given function
func (mc *mongoClient) UseSessionWithOptions(ctx context.Context,
	opts *options.SessionOptionsBuilder, fn func(ctx context.Context) error) error {
	return mc.cl.UseSessionWithOptions(ctx, opts, fn)
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
