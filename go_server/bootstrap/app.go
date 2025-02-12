package bootstrap

import (
	"comics/domain"
)

type Application struct {
	Env      *Env
	UserRepo domain.UserStore
}

func App() Application {
	app := &Application{}
	app.Env = NewEnv()
	app.UserRepo = NewMongoRepo(app.Env)
	return *app
}

func (app *Application) CloseDBConnection() {
	CloseMongoDBConnection(app.UserRepo)
}
