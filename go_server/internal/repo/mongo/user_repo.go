package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"comics/domain"
	"comics/internal/metrics"
	"comics/internal/repo"
	"comics/internal/tracer"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
)

const (
	namespace = "user_repo"
)

var (
	backoffMinInterval = 500 * time.Millisecond
	backoffTimeout     = 5 * time.Second
)

// Implement UserStore methods for UserRepo
var _ domain.UserStore = (*UserRepo)(nil)
var _ repo.Closable = (*UserRepo)(nil)

// UserRepo implements UserStore for MongoDB
type UserRepo struct {
	coll    Collection
	cl      Client
	metrics *metrics.Metrics
	tracer  *tracer.Tracer
}

// DefaultConfig returns a default configuration
func DefaultConfig() *repo.DBConfig {
	return &repo.DBConfig{
		Addr:           "localhost:27017", // local connection
		Name:           "comics_db",
		TableUsers:     "users",
		MaxPoolSize:    100,              // default from mongo driver
		MinPoolSize:    0,                // default from mongo driver
		ConnectTimeout: 30 * time.Second, // default from mongo driver
	}
}

// NewUserRepo creates a new MongoDB-based user repository for a given database and collection
func NewUserRepo(ctx context.Context, cfg *repo.DBConfig, tpCfg *tracer.TracerConfig) (*UserRepo, error) {
	// Validate configuration
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if tpCfg == nil {
		tpCfg = tracer.DefaultTracerConfig()
	}

	// Initialize metrics
	metrics := metrics.NewMetrics(tpCfg.ServiceName, namespace)

	// Initialize tracer
	tracer, err := tracer.NewTracer(ctx, tpCfg, namespace)
	if err != nil {
		return nil, fmt.Errorf("error creating tracer: %w", err)
	}

	// Initialize connection pool
	cl, err := newMongoClient(ctx, cfg, metrics)
	if err != nil {
		return nil, err
	}

	// Set backoff timeout
	if cfg.BackoffTimeout != 0 {
		backoffTimeout = cfg.BackoffTimeout
	}
	// Return the UserRepo
	return &UserRepo{
		coll:    cl.Database(cfg.Name).Collection(cfg.TableUsers),
		cl:      cl,
		metrics: metrics,
		tracer:  tracer,
	}, nil
}

// Client return the internal client
func (r *UserRepo) Client() Client { return r.cl }

// Close disconnects the client
func (r *UserRepo) Close(ctx context.Context) error {
	err := errors.Join(r.tracer.Shutdown(ctx), r.cl.Disconnect(ctx))
	if err != nil {
		return err
	}
	log.Info().Msg("User repo shutdown successfull")
	return nil
}

// Ping checks if the database is up
func (r *UserRepo) Ping(ctx context.Context) error { return r.cl.Ping(ctx) }

// Metrics return the internal metrics struct
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
	} else {
		span.SetOk()
	}

	r.metrics.RecordQuery(time.Since(start), operation, err)
	return err
}

func (r *UserRepo) withRetry(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
	expBackoff := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(backoffTimeout),
		backoff.WithInitialInterval(backoffMinInterval),
	)
	ctxBackoff := backoff.WithContext(expBackoff, ctx)

	return backoff.Retry(func() error {
		if err := fn(ctx); err != nil {
			r.metrics.RecordRetry(operation, false)
			if errors.Is(err, repo.ErrNotFound) ||
				errors.Is(err, repo.ErrInvalidPageParams) ||
				errors.Is(err, repo.ErrInvalidArgument) ||
				errors.Is(err, context.Canceled) {
				// Do not retry if the error is ErrNotFound
				return backoff.Permanent(err)
			}
			log.Warn().Err(err).Caller().Msgf("Operation %s failed, retrying", operation)
			return err
		}
		r.metrics.RecordRetry(operation, true)
		return nil
	}, ctxBackoff)
}

// GetByID retrieves a user by ID
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user := &domain.User{}

	err := r.withSpan(ctx, "GetByID", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetByID", func(ctx context.Context) error {
			tracer.FromContext(ctx).SetTag("id", id.String())

			err := r.coll.FindOne(ctx, map[string]any{"_id": id}).Decode(user)
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				return repo.ErrNotFound
			}
			return err

		})
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

// GetByEmail retrieves a user by email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{}
	err := r.withSpan(ctx, "GetByEmail", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetByEmail", func(ctx context.Context) error {
			tracer.FromContext(ctx).SetTag("email", email)

			// Find user by email
			err := r.coll.FindOne(ctx, map[string]any{"email": email}).Decode(user)
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				return repo.ErrNotFound
			}
			tracer.FromContext(ctx).SetTag("id", user.ID.String())
			return err

		})
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

// GetByUsername retrieves a user by username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	user := &domain.User{}

	err := r.withSpan(ctx, "GetByUsername", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetByUsername", func(ctx context.Context) error {
			tracer.FromContext(ctx).SetTag("username", username)

			// Find user by username
			err := r.coll.FindOne(ctx, map[string]any{"username": username}).Decode(user)
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				return repo.ErrNotFound
			}
			tracer.FromContext(ctx).SetTag("id", user.ID.String())
			return err

		})
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

// Create a new user
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	return r.withSpan(ctx, "Create", func(ctx context.Context) error {
		return r.withRetry(ctx, "Create", func(ctx context.Context) error {
			if user == nil {
				return repo.ErrInvalidArgument
			}
			tracer.FromContext(ctx).SetTag("id", user.ID)
			log := zerolog.Ctx(ctx)
			log.Debug().Msgf("Creating user %s", user)

			if user.ID == uuid.Nil || user.Username == "" || user.Email == "" {
				return repo.ErrInvalidArgument
			}

			// Set creation timestamps
			user.CreatedAt = time.Now()
			user.UpdatedAt = user.CreatedAt

			// Insert the user
			_, err := r.coll.InsertOne(ctx, user)

			if user.Password == "" {
				log.Warn().Msgf("Password is empty for user: %v", user)
			}
			return err
		})
	})
}

// Update a user by ID, but only if the user exists
// fields that can be updated: username, email, role, active
func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	return r.withSpan(ctx, "Update", func(ctx context.Context) error {
		return r.withRetry(ctx, "Update", func(ctx context.Context) error {
			if user == nil {
				return repo.ErrInvalidArgument
			}
			tracer.FromContext(ctx).SetTag("id", user.ID.String())
			log := zerolog.Ctx(ctx)
			log.Debug().Msgf("Updating user %s", user)

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
			if err != nil {
				return err
			}
			// Check if user was found and updated
			if result.MatchedCount == 0 {
				return repo.ErrNotFound
			}
			return nil
		})
	})
}

// Delete a user by ID
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.withSpan(ctx, "Delete", func(ctx context.Context) error {
		return r.withRetry(ctx, "Delete", func(ctx context.Context) error {
			tracer.FromContext(ctx).SetTag("id", id.String())

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
		return r.withRetry(ctx, "List", func(ctx context.Context) error {
			tracer.FromContext(ctx).SetTag("page", page)
			tracer.FromContext(ctx).SetTag("page_size", pageSize)

			if page < 1 || pageSize < 1 {
				return repo.ErrInvalidPageParams
			}
			skip := int64((page - 1) * pageSize)

			// Count total documents
			var err error
			totalCount, err = r.coll.CountDocuments(ctx, bson.M{})
			if err != nil {
				return err
			}
			tracer.FromContext(ctx).SetTag("total_count", totalCount)

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
			return cursor.All(ctx, &users)
		})
	})

	return users, totalCount, err
}

// FindActiveUsersByRole retrieves a list of active users by role
func (r *UserRepo) FindActiveUsersByRole(ctx context.Context, role string) ([]*domain.User, error) {
	var users []*domain.User
	err := r.withSpan(ctx, "FindActiveUsersByRole", func(ctx context.Context) error {
		return r.withRetry(ctx, "FindActiveUsersByRole", func(ctx context.Context) error {
			tracer.FromContext(ctx).SetTag("role", role)

			// Define filter for active users with specific role
			filter := bson.M{
				"role":   role,
				"active": true,
			}

			// Prepare find options
			findOptions := options.Find().
				SetProjection(bson.D{{Key: "password", Value: 0}})

			// Find users matching the filter
			cursor, err := r.coll.Find(ctx, filter, findOptions)
			if err != nil {
				return err
			}
			defer cursor.Close(ctx)

			// Decode results
			if err = cursor.All(ctx, &users); err != nil {
				return err
			}

			tracer.FromContext(ctx).SetTag("users", len(users))
			return nil
		})
	})

	return users, err
}

// Tx executes several ops within a MongoDB transaction
func (r *UserRepo) Tx(ctx context.Context, fn func(context.Context) error) error {
	return r.withSpan(ctx, "Tx", func(ctx context.Context) error {

		// Set read concern option to majority to avoid reading uncommitted data
		txnOpts := options.Transaction().SetReadConcern(readconcern.Majority())
		opts := options.Session().SetDefaultTransactionOptions(txnOpts)

		return r.cl.UseSessionWithOptions(ctx, opts, func(ctx context.Context) error {
			ses := mongo.SessionFromContext(ctx)

			_, err := ses.WithTransaction(ctx, func(ctx context.Context) (any, error) {
				return nil, fn(ctx)
			})
			return err
		})
	})
}
