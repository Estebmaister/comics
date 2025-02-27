package bootstrap

import (
	"context"

	"comics/domain"
	"comics/internal/logger"
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
}

func App(ctx context.Context) Application {
	app := &Application{}
	app.Env = MustLoadEnv(ctx)
// Initialize logger
	log, shutLogger, err := logger.InitLogger(ctx, app.Env.LoggerConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize logger sender")
	}
	defer shutLogger()
	userRepo, err := NewRepo(app.Env)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize user repo")
	}
	app.UserRepo = userRepo
	return *app
}

func (app *Application) CloseDBConnection(ctx context.Context) {
	CloseConnection(ctx, app.UserRepo)
}
