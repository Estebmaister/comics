package bootstrap

import (
	"comics/domain"
	"context"
	"log"
)

type Application struct {
	Env      *Env
	UserRepo domain.UserStore
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

func (app *Application) CloseDBConnection() {
	CloseMongoDBConnection(app.UserRepo)
}
