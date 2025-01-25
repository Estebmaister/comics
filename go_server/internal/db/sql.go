package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"comics/internal/metrics"
	"comics/internal/tracing"
	pb "comics/pb"

	"github.com/cenkalti/backoff/v4"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Database struct {
	db      *sql.DB
	metrics *metrics.Metrics
	tracer  *tracing.Tracer
}

type Config struct {
	Host            string
	Port            string
	Addr            string
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	JaegerEndpoint  string
}

func DefaultConfig() *Config {
	cfg := &Config{
		Host:            os.Getenv("PG_HOST"),
		Port:            os.Getenv("PG_PORT"),
		Addr:            os.Getenv("PG_ADDR"),
		User:            os.Getenv("PG_USER"),
		Password:        os.Getenv("PG_PASS"),
		Database:        os.Getenv("PG_NAME"),
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		JaegerEndpoint:  os.Getenv("JAEGER_ENDPOINT"),
	}
	if cfg.Host == "" && cfg.Port == "" {
		addr := strings.Split(cfg.Addr, ":")
		if len(addr) == 2 {
			cfg.Host = addr[0]
			cfg.Port = addr[1]
		}
	}
	return cfg
}

func NewDatabase(cfg *Config) (*Database, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	if cfg == nil {
		cfg = DefaultConfig()
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Initialize tracer
	tracer, err := tracing.NewTracer("comics-db", cfg.JaegerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error creating tracer: %v", err)
	}

	// Initialize metrics
	metrics := metrics.NewMetrics("comics_db")

	database := &Database{
		db:      db,
		metrics: metrics,
		tracer:  tracer,
	}

	// Test connection with retry
	if err := database.pingWithRetry(); err != nil {
		return nil, err
	}

	// Run migrations
	if err := database.runMigrations(); err != nil {
		return nil, err
	}

	return database, nil
}

func (db *Database) pingWithRetry() error {
	operation := func() error {
		return db.db.Ping()
	}

	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 1 * time.Minute

	return backoff.Retry(operation, retry)
}

func (db *Database) runMigrations() error {
	driver, err := postgres.WithInstance(db.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creating migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/db/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("error creating migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error running migrations: %v", err)
	}

	return nil
}

func (db *Database) withSpan(ctx context.Context, operation string, fn func(context.Context) error) error {
	ctx, span := db.tracer.StartSpan(ctx, operation)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start).Seconds()

	if err != nil {
		span.SetError(err)
	}

	db.metrics.RecordQuery(duration, operation, err)
	return err
}

func (db *Database) withRetry(_ context.Context, operation string, fn func() error) error {
	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 15 * time.Second

	err := backoff.Retry(func() error {
		if err := fn(); err != nil {
			db.metrics.RecordRetry(operation, false)
			log.Printf("Operation %s failed, retrying: %v", operation, err)
			return err
		}
		db.metrics.RecordRetry(operation, true)
		return nil
	}, retry)

	return err
}

func (db *Database) CreateComic(ctx context.Context, comic *pb.Comic) error {
	return db.withSpan(ctx, "CreateComic", func(ctx context.Context) error {
		return db.withRetry(ctx, "CreateComic", func() error {
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

			return db.db.QueryRowContext(
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

func (db *Database) UpdateComic(ctx context.Context, comic *pb.Comic) error {
	return db.withSpan(ctx, "UpdateComic", func(ctx context.Context) error {
		return db.withRetry(ctx, "UpdateComic", func() error {
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
			WHERE id = $14 AND NOT deleted
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
			err := db.db.QueryRowContext(
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

			if err == sql.ErrNoRows {
				return fmt.Errorf("comic not found")
			}
			return err
		})
	})
}

func (db *Database) DeleteComic(ctx context.Context, id uint32) error {
	return db.withSpan(ctx, "DeleteComic", func(ctx context.Context) error {
		return db.withRetry(ctx, "DeleteComic", func() error {
			query := `
			UPDATE comics
			SET deleted = true, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND NOT deleted
			RETURNING id`

			var deletedId int32
			err := db.db.QueryRowContext(ctx, query, id).Scan(&deletedId)
			if err == sql.ErrNoRows {
				return fmt.Errorf("comic not found")
			}
			return err
		})
	})
}

func (db *Database) GetComicById(ctx context.Context, id uint32) (*pb.Comic, error) {
	var comic pb.Comic
	err := db.withSpan(ctx, "GetComicById", func(ctx context.Context) error {
		return db.withRetry(ctx, "GetComicById", func() error {
			query := `
			SELECT id, titles, author, description, type, status, cover,
				   current_chap, last_update, publishers, genres,
				   track, viewed_chap, deleted
			FROM comics
			WHERE id = $1 AND NOT deleted`

			var lastUpdate time.Time
			var publishers, genres []int32

			err := db.db.QueryRowContext(ctx, query, id).Scan(
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
				if err == sql.ErrNoRows {
					return fmt.Errorf("comic not found")
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

func (db *Database) GetComics(ctx context.Context, page, pageSize int, trackedOnly, uncheckedOnly bool) ([]*pb.Comic, int, error) {
	var comics []*pb.Comic
	var total int

	err := db.withSpan(ctx, "GetComics", func(ctx context.Context) error {
		return db.withRetry(ctx, "GetComics", func() error {
			whereClause := "WHERE NOT deleted"
			if trackedOnly {
				whereClause += " AND track = true"
			}
			if uncheckedOnly {
				whereClause += " AND track = true AND viewed_chap < current_chap"
			}

			// Get total count
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM comics %s", whereClause)
			if err := db.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
				return err
			}

			// Get paginated results
			query := fmt.Sprintf(`
				SELECT id, titles, author, description, type, status, cover,
					   current_chap, last_update, publishers, genres,
					   track, viewed_chap, deleted
				FROM comics
				%s
				ORDER BY last_update DESC
				LIMIT $1 OFFSET $2`, whereClause)

			offset := (page - 1) * pageSize
			rows, err := db.db.QueryContext(ctx, query, pageSize, offset)
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

func (db *Database) SearchComics(ctx context.Context, query string, page, pageSize int) ([]*pb.Comic, int, error) {
	ctx, span := db.tracer.StartSpan(ctx, "SearchComics")
	defer span.End()

	// Validate input
	if query == "" {
		return nil, 0, fmt.Errorf("search query cannot be empty")
	}

	// Prepare the base query with search conditions
	baseQuery := `
		SELECT id, titles, author, description, type, status, cover, 
			   current_chap, last_update, publishers, genres, 
			   track, viewed_chap, deleted
		FROM comics
		WHERE 
			(LOWER(titles) LIKE LOWER($1) OR 
			 LOWER(author) LIKE LOWER($1) OR 
			 LOWER(description) LIKE LOWER($1))
	`
	countQuery := `
		SELECT COUNT(*)
		FROM comics
		WHERE 
			(LOWER(titles) LIKE LOWER($1) OR 
			 LOWER(author) LIKE LOWER($1) OR 
			 LOWER(description) LIKE LOWER($1))
	`

	// Prepare search pattern
	searchPattern := fmt.Sprintf("%%%s%%", query)

	// Count total matching records
	var total int
	err := db.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting search results: %v", err)
	}

	// Prepare pagination
	offset := (page - 1) * pageSize
	baseQuery += ` LIMIT $2 OFFSET $3`

	// Execute search query
	rows, err := db.db.QueryContext(ctx, baseQuery, searchPattern, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching comics: %v", err)
	}
	defer rows.Close()

	var comics []*pb.Comic
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
			return nil, 0, fmt.Errorf("error scanning comic row: %v", err)
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
		return nil, 0, fmt.Errorf("error in search results: %v", err)
	}

	return comics, total, nil
}

func (db *Database) GetComicByTitle(ctx context.Context, title string) (*pb.Comic, error) {
	var comic *pb.Comic
	err := db.withSpan(ctx, "GetComicByTitle", func(ctx context.Context) error {
		query := `
			SELECT id, titles, author, description, type, status, cover, 
			       current_chap, last_update, publishers, genres, 
			       track, viewed_chap, deleted
			FROM comics
			WHERE titles ILIKE $1 AND deleted = false
			LIMIT 1`

		comic = &pb.Comic{}
		var publishersStr, genresStr string

		err := db.db.QueryRowContext(ctx, query, "%"+title+"%").Scan(
			&comic.Id,
			&comic.Titles,
			&comic.Author,
			&comic.Description,
			&comic.ComType,
			&comic.Status,
			&comic.Cover,
			&comic.CurrentChap,
			&comic.LastUpdate,
			&publishersStr,
			&genresStr,
			&comic.Track,
			&comic.ViewedChap,
			&comic.Deleted,
		)

		if err == sql.ErrNoRows {
			return fmt.Errorf("comic not found")
		}
		if err != nil {
			return fmt.Errorf("error retrieving comic by title: %w", err)
		}

		// Convert publishers and genres strings to int32 slices
		if publishersStr != "" {
			publisherIDs := strings.Split(publishersStr, ",")
			comic.PublishedIn = make([]pb.Publisher, len(publisherIDs))
			for i, idStr := range publisherIDs {
				id, _ := strconv.Atoi(idStr)
				comic.PublishedIn[i] = pb.Publisher(id)
			}
		}

		if genresStr != "" {
			genreIDs := strings.Split(genresStr, ",")
			comic.Genres = make([]pb.Genre, len(genreIDs))
			for i, idStr := range genreIDs {
				id, _ := strconv.Atoi(idStr)
				comic.Genres[i] = pb.Genre(id)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return comic, nil
}

func (db *Database) GetMetrics() *metrics.MetricsSnapshot {
	return db.metrics.GetSnapshot()
}

func (db *Database) Ping(ctx context.Context) error {
	return db.withSpan(ctx, "Ping", func(ctx context.Context) error {
		return db.db.PingContext(ctx)
	})
}

func (db *Database) Close() error {
	ctx := context.Background()
	if err := db.tracer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down tracer: %v", err)
	}
	return db.db.Close()
}

func (db *Database) DB() *sql.DB {
	return db.db
}
