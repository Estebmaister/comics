package bootstrap

import (
	"context"
	"log"
	"time"

	"comics/domain"
)

// Closable defines a common interface for closing database connections
type Closable interface {
	Close(ctx context.Context, duration time.Duration) error
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
	userRepo, err := NewRepo(app.Env)
	if err != nil {
		log.Fatalf("Failed to initialize user repo: %s", err)
	}
	app.UserRepo = userRepo
	return *app
}

func (app *Application) CloseDBConnection(ctx context.Context, duration time.Duration) {
	CloseConnection(ctx, app.UserRepo, duration)
}
