package bootstrap

import (
	"context"

	"comics/domain"
	"comics/internal/logger"

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

type Application struct {
	Env      *Env
	UserRepo ClosableUserStore
	Shutters []func(context.Context) error
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

	// Return the application
	return Application{
		Env:      env,
		UserRepo: userRepo,
		Shutters: []func(context.Context) error{
			userRepo.Close,
			loggerClose,
		},
	}
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
