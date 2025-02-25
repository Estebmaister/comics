package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"comics/api/route"
	"comics/bootstrap"
	_ "comics/docs"
	repo "comics/repo/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// main serves the http app to authenticate request
//
//	@title						Comics API
//	@version					1.1
//	@description				Server documentation to query comics from the DB.
//	@termsOfService				http://swagger.io/terms/
//
//	@contact.name				Estebmaister
//	@contact.url				http://www.github.com/estebmaister
//	@contact.email				estebmaister@gmail.com
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@securityDefinitions.apikey	Bearer JWT
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space paste the JWT.
//
//	@host						localhost:8081
//	@BasePath					/
func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer stop()

	// app is the instance of the entire application, managing key resources throughout its lifecycle
	app := bootstrap.App(ctx)
	defer func() {
		app.CloseDBConnection(ctx)
	}()

	// Creating a gin instance
	if app.Env.AppEnv == bootstrap.Development {
		gin.SetMode(gin.ReleaseMode)
	}
	g := gin.New()
	g.Use(gin.Recovery())
	// Route binding
	route.Setup(app.Env, app.UserRepo, g)

	// Running the server
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- g.Run(app.Env.ServerAddress)
	}()

	// Initialize the database
	_, err := repo.NewSQLiteDB("../src/db/comics.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize SQLite database")
	}

	// Wait for interruption
	select {
	case err = <-srvErr:
		log.Fatal().Err(err).Msg("Error when starting HTTP server.")
		return
	case <-ctx.Done():
		// Stop receiving signal notifications as soon as possible.
		stop()
	}
	log.Info().Msg("Shutting down server...")
}
