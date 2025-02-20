package mongo

import (
	"context"
	"fmt"
	"time"

	"comics/domain"
	"comics/internal/metrics"
	"comics/internal/repo"
	"comics/internal/tracing"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	tracerName = "user-db"
	namespace  = "comics_db"
	subsystem  = "user_repo"
)

// Implement UserStore methods for UserRepo
var _ domain.UserStore = (*UserRepo)(nil)
var _ repo.Closable = (*UserRepo)(nil)

// UserRepo implements UserStore for MongoDB
type UserRepo struct {
	coll    Collection
	cl      Client
	metrics *metrics.Metrics
	tracer  *tracing.Tracer
}

// Client return the internal client
func (r *UserRepo) Client() Client {
	return r.cl
}

// NewUserRepo creates a new MongoDB-based user repository for a given database and collection
func NewUserRepo(ctx context.Context, cfg *repo.DBConfig) (*UserRepo, error) {
	// Initialize metrics
	metrics := metrics.NewMetrics(namespace, subsystem)

	// Initialize tracer
	tracer, err := tracing.NewTracer(tracerName, cfg.JaegerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error creating tracer: %w", err)
	}

	cl, err := newMongoClient(ctx, cfg, metrics)
	if err != nil {
		return nil, err
	}

	// err = cl.Ping(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	return &UserRepo{
		coll:    cl.Database(cfg.Name).Collection(cfg.TableUsers),
		cl:      cl,
		metrics: metrics,
		tracer:  tracer,
	}, nil
}

// Close disconnects the client
func (r *UserRepo) Close(ctx context.Context, duration time.Duration) error {
	return r.cl.Disconnect(ctx)
}

// Metrics return the internal metrics
func (r *UserRepo) Metrics() *metrics.Metrics { return r.metrics }

// Metrics return the internal stats
func (r *UserRepo) GetStats() map[string]string { return r.metrics.GetStats() }

// GetByID retrieves a user by ID
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	// Record query start time
	startTime := time.Now()

	user := &domain.User{}
	err := r.coll.FindOne(ctx, map[string]interface{}{"_id": id}).Decode(user)

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "GetByID", err)

	if err != nil && err.Error() == mongo.ErrNoDocuments.Error() {
		return nil, repo.ErrNotFound
	}
	return user, err
}

// GetByEmail retrieves a user by email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Record query start time
	startTime := time.Now()

	user := &domain.User{}
	err := r.coll.FindOne(ctx, map[string]interface{}{"email": email}).Decode(user)

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "GetByEmail", err)

	if err != nil && err.Error() == mongo.ErrNoDocuments.Error() {
		return nil, repo.ErrNotFound
	}
	return user, err
}

// GetByUsername retrieves a user by username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	// Record query start time
	startTime := time.Now()

	user := &domain.User{}
	err := r.coll.FindOne(ctx, map[string]interface{}{"username": username}).Decode(user)

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "GetByUsername", err)

	if err != nil && err.Error() == mongo.ErrNoDocuments.Error() {
		return nil, repo.ErrNotFound
	}
	return user, err
}

// Create a new user
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	// Record query start time
	startTime := time.Now()

	// Set creation timestamps
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt

	// Insert the user
	_, err := r.coll.InsertOne(ctx, user)

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "Create", err)

	return err
}

// Update a user by ID, but only if the user exists
// fields that can be updated: username, email, role, active
func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	// Record query start time
	startTime := time.Now()

	// Prepare update document
	update := bson.M{
		"$set": bson.M{
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"updated_at": time.Now(),
			"active":     user.Active,
		},
	}

	// Perform update
	result, err := r.coll.UpdateByID(ctx, user.ID, update)

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "Update", err)

	// Check if user was found and updated
	if result.MatchedCount == 0 {
		return repo.ErrNotFound
	}

	return err
}

// Delete a user by ID
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	// Record query start time
	startTime := time.Now()

	// Perform delete
	result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "Delete", err)

	// Check if user was found and deleted
	if result.DeletedCount == 0 {
		return repo.ErrNotFound
	}

	return err
}

// List retrieves a list of users with pagination
func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	// Record query start time
	startTime := time.Now()

	// Calculate skip
	skip := int64((page - 1) * pageSize)

	// Count total documents
	totalCount, err := r.coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Prepare find options for pagination
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetProjection(bson.D{{Key: "password", Value: 0}})

	// Find users with pagination
	cursor, err := r.coll.Find(ctx, bson.D{}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	// Decode results
	var users []*domain.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "List", err)

	return users, totalCount, nil
}

// FindActiveUsersByRole retrieves a list of active users by role
func (r *UserRepo) FindActiveUsersByRole(ctx context.Context, role string) ([]*domain.User, error) {
	// Record query start time
	startTime := time.Now()

	// Define filter for active users with specific role
	filter := bson.M{
		"role":   role,
		"active": true,
	}

	// Prepare find options
	findOptions := options.Find()

	// Find users matching the filter
	cursor, err := r.coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "FindActiveUsersByRole", err)

	return users, nil
}

// PerformTransaction executes several ops within a MongoDB transaction
func (r *UserRepo) PerformTransaction(ctx context.Context, fn func(context.Context) error) error {
	// Record query start time
	startTime := time.Now()

	// Start session
	session, err := r.cl.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Start transaction
	err = session.StartTransaction()
	if err != nil {
		return err
	}

	// Execute function within transaction
	err = fn(ctx)
	if err != nil {
		session.AbortTransaction(ctx)
		return err
	}

	// Commit transaction
	err = session.CommitTransaction(ctx)

	// Record query metrics
	queryDuration := time.Since(startTime)
	r.Metrics().RecordQuery(queryDuration, "PerformTransaction", err)

	return err
}
