package mongo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"comics/domain"
	"comics/internal/metrics"
	"comics/internal/repo"
	"comics/internal/tracing"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

	err = cl.Ping(ctx)
	if err != nil {
		return nil, err
	}

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

func (r *UserRepo) withSpan(ctx context.Context, operation string, fn func(context.Context) error) error {
	ctx, span := r.tracer.StartSpan(ctx, operation)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}

	r.metrics.RecordQuery(time.Since(start), operation, err)
	return err
}

func (r *UserRepo) withRetry(_ context.Context, operation string, fn func() error) error {
	retry := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(15 * time.Second),
	)

	return backoff.Retry(func() error {
		if err := fn(); err != nil {
			r.metrics.RecordRetry(operation, false)
			if errors.Is(err, repo.ErrNotFound) {
				// Do not retry if the error is ErrNotFound
				return backoff.Permanent(err)
			}
			log.Printf("Operation %s failed, retrying: %v\n", operation, err)
			return err
		}
		r.metrics.RecordRetry(operation, true)
		return nil
	}, retry)
}

// GetByID retrieves a user by ID
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user := &domain.User{}

	err := r.withSpan(ctx, "GetByID", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetByID", func() error {

			err := r.coll.FindOne(ctx, map[string]interface{}{"_id": id}).Decode(user)
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				return repo.ErrNotFound
			}
			return err

		})
	})
	return user, err
}

// GetByEmail retrieves a user by email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{}

	err := r.withSpan(ctx, "GetByEmail", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetByEmail", func() error {

			// Find user by email
			err := r.coll.FindOne(ctx, map[string]interface{}{"email": email}).Decode(user)
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				return repo.ErrNotFound
			}
			return err

		})
	})
	return user, err
}

// GetByUsername retrieves a user by username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	user := &domain.User{}

	err := r.withSpan(ctx, "GetByUsername", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetByUsername", func() error {

			// Find user by username
			err := r.coll.FindOne(ctx, map[string]interface{}{"username": username}).Decode(user)
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				return repo.ErrNotFound
			}
			return err

		})
	})
	return user, err
}

// Create a new user
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	return r.withSpan(ctx, "Create", func(ctx context.Context) error {
		return r.withRetry(ctx, "Create", func() error {

			// Set creation timestamps
			user.CreatedAt = time.Now()
			user.UpdatedAt = user.CreatedAt

			// Insert the user
			_, err := r.coll.InsertOne(ctx, user)
			return err
		})
	})
}

// Update a user by ID, but only if the user exists
// fields that can be updated: username, email, role, active
func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	return r.withSpan(ctx, "Update", func(ctx context.Context) error {
		return r.withRetry(ctx, "Update", func() error {

			// Prepare updated user document
			update := bson.M{"$set": bson.M{
				"username":   user.Username,
				"email":      user.Email,
				"password":   user.Password,
				"role":       user.Role,
				"updated_at": time.Now(),
				"active":     user.Active, // User gets active by default when updating
			}}

			// Perform update on the provided ID
			result, err := r.coll.UpdateByID(ctx, user.ID, update)
			// Check if user was found and updated
			if result.MatchedCount == 0 {
				return repo.ErrNotFound
			}
			return err
		})
	})
}

// Delete a user by ID
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.withSpan(ctx, "Delete", func(ctx context.Context) error {
		return r.withRetry(ctx, "Delete", func() error {

			// Perform delete on given ID
			result, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
			// Check if user was found and deleted
			if result.DeletedCount == 0 {
				return repo.ErrNotFound
			}
			return err
		})
	})
}

// List retrieves a list of users with pagination
func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	var users []*domain.User
	var totalCount int64
	err := r.withSpan(ctx, "List", func(ctx context.Context) error {
		return r.withRetry(ctx, "List", func() error {

			// Calculate skip
			skip := int64((page - 1) * pageSize)

			// Count total documents
			var err error
			totalCount, err = r.coll.CountDocuments(ctx, bson.M{})
			if err != nil {
				return err
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
				return err
			}
			defer cursor.Close(ctx)

			// Decode results
			var users []*domain.User
			if err = cursor.All(ctx, &users); err != nil {
				return err
			}
			return nil
		})
	})

	return users, totalCount, err
}

// FindActiveUsersByRole retrieves a list of active users by role
func (r *UserRepo) FindActiveUsersByRole(ctx context.Context, role string) ([]*domain.User, error) {
	var users []*domain.User
	err := r.withSpan(ctx, "FindActiveUsersByRole", func(ctx context.Context) error {
		return r.withRetry(ctx, "FindActiveUsersByRole", func() error {

			// Define filter for active users with specific role
			filter := bson.M{
				"role":   role,
				"active": true,
			}

			// Prepare find options
			findOptions := options.Find().SetProjection(bson.D{{Key: "password", Value: 0}})

			// Find users matching the filter
			cursor, err := r.coll.Find(ctx, filter, findOptions)
			if err != nil {
				return err
			}
			defer cursor.Close(ctx)

			if err = cursor.All(ctx, &users); err != nil {
				return err
			}
			return nil
		})
	})

	return users, err
}

// PerformTransaction executes several ops within a MongoDB transaction
func (r *UserRepo) PerformTransaction(ctx context.Context, fn func(context.Context) error) error {
	return r.withSpan(ctx, "PerformTransaction", func(ctx context.Context) error {
		return r.withRetry(ctx, "PerformTransaction", func() error {

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
			return session.CommitTransaction(ctx)
		})
	})
}
