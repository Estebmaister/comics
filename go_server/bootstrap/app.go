package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"comics/domain"
	"comics/internal/logger"
	comicrepo "comics/internal/repo/comics"
	"comics/internal/scrape"
	"comics/internal/usecase"

	"github.com/rs/zerolog/log"
)

// Closable defines a common interface for closing database connections
type Closable interface {
	Close(ctx context.Context) error
}

// ClosableUserStore defines
type ClosableUserStore interface {
	Closable
	domain.UserStore
}

type comicServiceCloser struct {
	domain.ComicUseCase
}

func (s comicServiceCloser) Close(_ context.Context) error {
	return s.ComicUseCase.Close()
}

// Application is the main application struct
type Application struct {
	Env          *Env
	UserRepo     ClosableUserStore
	ComicService domain.ComicUseCase
	ScrapeCoord  *scrape.Coordinator
	Shutters     []func(context.Context) error
}

// MustLoadApp loads the application from the environment variables
func MustLoadApp(ctx context.Context) Application {
	// Load environment variables
	env := MustLoadEnv(ctx)

	// Context with timeout for the initialization
	ctx, cancel := context.WithTimeout(ctx, env.InitCtxTimeout)
	defer cancel()

	// Initialize logger
	log, loggerClose, err := logger.InitLogger(ctx, env.LoggerConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	// Initialize user repository
	userRepo, err := newRepo(ctx, env, mongoUserRepoType)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize user repo")
	}
	if err := userRepo.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping user repo")
	}

	comicService, scrapeCoord, err := newComicStore(env)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize comic store")
	}

	// Return the application
	return Application{
		Env:          env,
		UserRepo:     userRepo,
		ComicService: comicService,
		ScrapeCoord:  scrapeCoord,
		Shutters: []func(context.Context) error{
			comicServiceCloser{comicService}.Close,
			userRepo.Close,
			loggerClose,
		},
	}
}

func newComicStore(env *Env) (domain.ComicUseCase, *scrape.Coordinator, error) {
	var repo domain.ComicRepository
	var err error
	switch strings.ToLower(strings.TrimSpace(env.ComicsDBDriver)) {
	case "", "sqlite":
		repo, err = comicrepo.NewSQLiteComicRepository(env.ComicsSQLitePath)
	case "postgres", "postgresql":
		repo, err = comicrepo.NewPostgresComicRepository(env.ComicsPostgresURL)
	default:
		return nil, nil, fmt.Errorf("unknown comics DB driver %q", env.ComicsDBDriver)
	}
	if err != nil {
		return nil, nil, err
	}
	coord := scrape.NewCoordinator(repo)
	return usecase.NewComicService(repo, coord), coord, nil
}

// Close closes the application resources
func (app *Application) Close(ctx context.Context) {
	// Context with timeout for the shutdown
	ctx, cancel := context.WithTimeout(ctx, app.Env.InitCtxTimeout)
	defer cancel()

	// Shutdown each shutter in order
	for _, shutter := range app.Shutters {
		if err := shutter(ctx); err != nil {
			// Log the error if a shutter function fails
			log.Error().Err(err).Msg("Error during shutdown")
		}
	}
}
