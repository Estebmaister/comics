package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"comics/internal/metrics"
	"comics/internal/tracer"
	"comics/pkg/pb"

	"github.com/cenkalti/backoff/v4"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file" // migrate file driver
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	namespace = "comics_repo"
	// requires migrate file driver on imports
	migrationSource = "file://internal/repo/sql/migrations"
)

var (
	backoffMinInterval = 500 * time.Millisecond
	backoffTimeout     = 5 * time.Second
)

// Implement UserStore methods for UserRepo
var _ Closable = (*ComicsRepo)(nil)

// ComicsRepo implements UserStore for PostgreSQL
type ComicsRepo struct {
	cl      *pgxpool.Pool
	metrics *metrics.Metrics
	tracer  *tracer.Tracer
}

// DefaultConfig returns a DBConfig empty usable struct
func DefaultConfig() *DBConfig {
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Caller().Msgf(".env file not found")
	}
	cfg := &DBConfig{
		Addr:            os.Getenv("PG_ADDR"),
		User:            os.Getenv("PG_USER"),
		Pass:            os.Getenv("PG_PASS"),
		Name:            os.Getenv("PG_NAME"),
		MaxPoolSize:     100,
		MaxConnIdleTime: 5 * time.Minute,
	}
	if cfg.Name == "" {
		cfg.Name = "comics_db"
	}
	if cfg.Host == "" && cfg.Port == 0 {
		addr := strings.Split(cfg.Addr, ":")
		if len(addr) == 2 {
			if len(addr[0]) == 0 {
				cfg.Host = "localhost"
			}
			cfg.Host = addr[0]
			port, err := strconv.Atoi(addr[1])
			if err != nil {
				port = 5432
			}
			cfg.Port = port
		}
	}
	return cfg
}

// NewComicsRepo creates a new PostgreSQL-based comic repository
func NewComicsRepo(ctx context.Context, cfg *DBConfig, tpCfg *tracer.Config) (*ComicsRepo, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if tpCfg == nil {
		tpCfg = tracer.DefaultTracerConfig()
	}

	connStr := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s",
		cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Name) // sslmode=disable
	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %v", err)
	}

	// Ensure pool sizes are within the int32 range
	if cfg.MaxPoolSize > math.MaxInt32 {
		return nil, fmt.Errorf("MaxPoolSize exceeds int32 range")
	}
	if cfg.MinPoolSize > math.MaxInt32 {
		return nil, fmt.Errorf("MinPoolSize exceeds int32 range")
	}

	// Configure connection pool settings
	poolCfg.MaxConns = int32(cfg.MaxPoolSize)
	poolCfg.MinConns = int32(cfg.MinPoolSize)
	poolCfg.MaxConnLifetime = cfg.MaxConnLifeTime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	cl, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Initialize tracer
	tracer, err := tracer.NewTracer(ctx, tpCfg, namespace)
	if err != nil {
		return nil, fmt.Errorf("error creating tracer: %w", err)
	}

	// Initialize metrics
	metrics := metrics.NewMetrics(tpCfg.ServiceName, namespace)

	// Set backoff timeout
	backoffTimeout = cfg.BackoffTimeout

	// Create repository
	repo := &ComicsRepo{
		cl:      cl,
		metrics: metrics,
		tracer:  tracer,
	}

	// Test connection with retry
	if err := repo.pingWithRetry(ctx); err != nil {
		return nil, err
	}

	// Run migrations
	if err := repo.runMigrations(ctx, cfg.Name); err != nil {
		return nil, err
	}

	return repo, nil
}

// Close closes the repository and shuts down the tracer
func (r *ComicsRepo) Close(ctx context.Context) error {
	r.cl.Close()
	err := r.tracer.Shutdown(ctx)
	if err != nil {
		return err
	}
	log.Info().Msg("Comics repo shutdown successfull")
	return nil
}

// Metrics returns a snapshot of the repository's metrics
func (r *ComicsRepo) Metrics() *metrics.Snapshot {
	log.Debug().Msgf("Comics repo db metrics: %v", r.cl.Stat())
	return r.metrics.GetSnapshot()
}

// Client returns the pool client
func (r *ComicsRepo) Client() *pgxpool.Pool { return r.cl }

// Ping checks if the database is up
func (r *ComicsRepo) Ping(ctx context.Context) error { return r.cl.Ping(ctx) }

func (r *ComicsRepo) pingWithRetry(ctx context.Context) error {
	return r.withRetry(ctx, "Ping", func() error { return r.Ping(ctx) })
}

func (r *ComicsRepo) runMigrations(_ context.Context, dataBaseName string) error {
	// Create a new pgx driver instance
	db := stdlib.OpenDBFromPool(r.cl)
	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("error creating migration driver: %w", err)
	}

	// Create a new migration instance
	m, err := migrate.NewWithDatabaseInstance(migrationSource, dataBaseName, driver)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error running migrations: %w", err)
	}

	return nil
}

func (r *ComicsRepo) withSpan(ctx context.Context, operation string, fn func(context.Context) error) error {
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

func (r *ComicsRepo) withRetry(ctx context.Context, operation string, fn func() error) error {
	expBackoff := backoff.NewExponentialBackOff(
		backoff.WithMaxElapsedTime(backoffTimeout),
		backoff.WithInitialInterval(backoffMinInterval),
	)
	ctxBackoff := backoff.WithContext(expBackoff, ctx)

	return backoff.Retry(func() error {
		if err := fn(); err != nil {
			r.metrics.RecordRetry(operation, false)
			if errors.Is(err, ErrNotFound) {
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

// CreateComic creates a new comic in the database
func (r *ComicsRepo) CreateComic(ctx context.Context, comic *pb.Comic) error {
	return r.withSpan(ctx, "CreateComic", func(ctx context.Context) error {
		return r.withRetry(ctx, "CreateComic", func() error {
			query := `
			INSERT INTO comics (
				titles, author, description, type, status, cover,
				current_chap, last_update, publishers, genres,
				track, viewed_chap, deleted
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id`

			publishers := make([]int32, len(comic.PublishedIn))
			for i, p := range comic.PublishedIn {
				publishers[i] = int32(p)
			}

			genres := make([]int32, len(comic.Genres))
			for i, g := range comic.Genres {
				genres[i] = int32(g)
			}

			return r.cl.QueryRow(
				ctx,
				query,
				comic.Titles,
				comic.Author,
				comic.Description,
				comic.ComType,
				comic.Status,
				comic.Cover,
				comic.CurrentChap,
				time.Now(),
				publishers,
				genres,
				comic.Track,
				comic.ViewedChap,
				comic.Deleted,
			).Scan(&comic.Id)
		})
	})
}

// UpdateComic updates an existing comic in the database
func (r *ComicsRepo) UpdateComic(ctx context.Context, comic *pb.Comic) error {
	return r.withSpan(ctx, "UpdateComic", func(ctx context.Context) error {
		return r.withRetry(ctx, "UpdateComic", func() error {
			query := `
			UPDATE comics SET
				titles = $1,
				author = $2,
				description = $3,
				type = $4,
				status = $5,
				cover = $6,
				current_chap = $7,
				last_update = $8,
				publishers = $9,
				genres = $10,
				track = $11,
				viewed_chap = $12,
				deleted = $13
			WHERE id = $14
			RETURNING id`

			publishers := make([]int32, len(comic.PublishedIn))
			for i, p := range comic.PublishedIn {
				publishers[i] = int32(p)
			}

			genres := make([]int32, len(comic.Genres))
			for i, g := range comic.Genres {
				genres[i] = int32(g)
			}

			var id int32
			err := r.cl.QueryRow(
				ctx,
				query,
				comic.Titles,
				comic.Author,
				comic.Description,
				comic.ComType,
				comic.Status,
				comic.Cover,
				comic.CurrentChap,
				time.Now(),
				publishers,
				genres,
				comic.Track,
				comic.ViewedChap,
				comic.Deleted,
				comic.Id,
			).Scan(&id)

			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		})
	})
}

// DeleteComic deletes an existing comic from the database
func (r *ComicsRepo) DeleteComic(ctx context.Context, id uint32) error {
	return r.withSpan(ctx, "DeleteComic", func(ctx context.Context) error {
		return r.withRetry(ctx, "DeleteComic", func() error {
			query := `
			UPDATE comics
			SET deleted = true, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING id`

			var deletedID int32
			err := r.cl.QueryRow(ctx, query, id).Scan(&deletedID)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		})
	})
}

// GetComicByID retrieves a comic by its ID
func (r *ComicsRepo) GetComicByID(ctx context.Context, id uint32) (*pb.Comic, error) {
	var comic pb.Comic
	err := r.withSpan(ctx, "GetComicByID", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetComicByID", func() error {
			query := `
			SELECT id, titles, author, description, type, status, cover, current_chap,
				last_update, publishers, genres, track, viewed_chap, deleted
			FROM comics
			WHERE id = $1`

			var lastUpdate time.Time
			var publishers, genres []int32

			err := r.cl.QueryRow(ctx, query, id).Scan(
				&comic.Id,
				&comic.Titles,
				&comic.Author,
				&comic.Description,
				&comic.ComType,
				&comic.Status,
				&comic.Cover,
				&comic.CurrentChap,
				&lastUpdate,
				&publishers,
				&genres,
				&comic.Track,
				&comic.ViewedChap,
				&comic.Deleted,
			)

			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrNotFound
				}
				return err
			}

			comic.LastUpdate = timestamppb.New(lastUpdate)
			comic.PublishedIn = make([]pb.Publisher, len(publishers))
			comic.Genres = make([]pb.Genre, len(genres))

			for i, p := range publishers {
				comic.PublishedIn[i] = pb.Publisher(p)
			}
			for i, g := range genres {
				comic.Genres[i] = pb.Genre(g)
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return &comic, nil
}

// GetComics retrieves a paginated list of comics
func (r *ComicsRepo) GetComics(ctx context.Context, page, pageSize int, trackedOnly, uncheckedOnly bool) ([]*pb.Comic, int, error) {
	var comics []*pb.Comic
	var total int

	err := r.withSpan(ctx, "GetComics", func(ctx context.Context) error {
		return r.withRetry(ctx, "GetComics", func() error {
			whereClause := "WHERE NOT deleted"
			if trackedOnly {
				whereClause += " AND track = true"
			}
			if uncheckedOnly {
				whereClause += " AND track = true AND viewed_chap < current_chap"
			}

			// Get total count
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM comics %s", whereClause)
			if err := r.cl.QueryRow(ctx, countQuery).Scan(&total); err != nil {
				return err
			}

			// Get paginated results
			query := fmt.Sprintf(`
				SELECT id, titles, author, description, type, status, cover, current_chap,
					last_update, publishers, genres, track, viewed_chap, deleted
				FROM comics
				%s
				ORDER BY last_update DESC
				LIMIT $1 OFFSET $2`, whereClause)

			offset := (page - 1) * pageSize
			rows, err := r.cl.Query(ctx, query, pageSize, offset)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var comic pb.Comic
				var lastUpdate time.Time
				var publishers, genres []int32

				err := rows.Scan(
					&comic.Id,
					&comic.Titles,
					&comic.Author,
					&comic.Description,
					&comic.ComType,
					&comic.Status,
					&comic.Cover,
					&comic.CurrentChap,
					&lastUpdate,
					&publishers,
					&genres,
					&comic.Track,
					&comic.ViewedChap,
					&comic.Deleted,
				)
				if err != nil {
					return err
				}

				comic.LastUpdate = timestamppb.New(lastUpdate)
				comic.PublishedIn = make([]pb.Publisher, len(publishers))
				comic.Genres = make([]pb.Genre, len(genres))

				for i, p := range publishers {
					comic.PublishedIn[i] = pb.Publisher(p)
				}
				for i, g := range genres {
					comic.Genres[i] = pb.Genre(g)
				}

				comics = append(comics, &comic)
			}

			return rows.Err()
		})
	})

	if err != nil {
		return nil, 0, err
	}
	return comics, total, nil
}

// SearchComics searches for comics by title, author, or genre
func (r *ComicsRepo) SearchComics(ctx context.Context, query string, page, pageSize int) (comics []*pb.Comic, total int, err error) {
	ctx, span := r.tracer.StartSpan(ctx, "SearchComics")
	defer span.End()

	// Validate input
	if query == "" {
		err = fmt.Errorf("search query cannot be empty")
		return comics, total, err
	}

	// Prepare the base query with search conditions
	baseQuery := `
		SELECT id, titles, author, description, type, status, cover, current_chap,
			last_update, publishers, genres, track, viewed_chap, deleted
		FROM comics
		WHERE 
			(	LOWER(titles) LIKE LOWER($1) OR 
				LOWER(author) LIKE LOWER($1) OR 
				LOWER(description) LIKE LOWER($1)	)
	`
	countQuery := `
		SELECT COUNT(*)
		FROM comics
		WHERE 
			(	LOWER(titles) LIKE LOWER($1) OR 
				LOWER(author) LIKE LOWER($1) OR 
				LOWER(description) LIKE LOWER($1)	)
	`

	// Prepare search pattern
	searchPattern := fmt.Sprintf("%%%s%%", query)

	// Count total matching records
	err = r.cl.QueryRow(ctx, countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting search results: %w", err)
	}

	// Prepare pagination
	offset := (page - 1) * pageSize
	baseQuery += ` LIMIT $2 OFFSET $3`

	// Execute search query
	rows, err := r.cl.Query(ctx, baseQuery, searchPattern, pageSize, offset)
	if err != nil {
		err = fmt.Errorf("error searching comics: %w", err)
		return comics, total, err
	}
	defer rows.Close()

	for rows.Next() {
		var comic pb.Comic
		var lastUpdate time.Time
		var publishers, genres []int32

		err := rows.Scan(
			&comic.Id,
			&comic.Titles,
			&comic.Author,
			&comic.Description,
			&comic.ComType,
			&comic.Status,
			&comic.Cover,
			&comic.CurrentChap,
			&lastUpdate,
			&publishers,
			&genres,
			&comic.Track,
			&comic.ViewedChap,
			&comic.Deleted,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning comic row: %w", err)
		}

		comic.LastUpdate = timestamppb.New(lastUpdate)
		comic.PublishedIn = make([]pb.Publisher, len(publishers))
		comic.Genres = make([]pb.Genre, len(genres))

		for i, p := range publishers {
			comic.PublishedIn[i] = pb.Publisher(p)
		}
		for i, g := range genres {
			comic.Genres[i] = pb.Genre(g)
		}

		comics = append(comics, &comic)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error in search results: %w", err)
	}

	return comics, total, nil
}

// GetComicsByTitle retrieves comics by title
func (r *ComicsRepo) GetComicsByTitle(ctx context.Context, title string) (comics []*pb.Comic, err error) {
	err = r.withSpan(ctx, "GetComicsByTitle", func(ctx context.Context) error {
		query := `
			SELECT id, titles, author, description, type, status, cover, current_chap,
				last_update, publishers, genres, track, viewed_chap, deleted
			FROM comics
			WHERE titles ILIKE $1 AND deleted = false`

		rows, err := r.cl.Query(ctx, query, "%"+title+"%")
		if err != nil {
			return fmt.Errorf("error searching comics: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var comic pb.Comic
			var lastUpdate time.Time
			var publishers, genres []int32

			err := rows.Scan(
				&comic.Id,
				&comic.Titles,
				&comic.Author,
				&comic.Description,
				&comic.ComType,
				&comic.Status,
				&comic.Cover,
				&comic.CurrentChap,
				&lastUpdate,
				&publishers,
				&genres,
				&comic.Track,
				&comic.ViewedChap,
				&comic.Deleted,
			)
			if err != nil {
				return fmt.Errorf("error scanning comic row: %w", err)
			}

			comic.LastUpdate = timestamppb.New(lastUpdate)
			comic.PublishedIn = make([]pb.Publisher, len(publishers))
			comic.Genres = make([]pb.Genre, len(genres))

			for i, p := range publishers {
				comic.PublishedIn[i] = pb.Publisher(p)
			}
			for i, g := range genres {
				comic.Genres[i] = pb.Genre(g)
			}

			comics = append(comics, &comic)
		}

		return nil
	})

	return comics, err
}
