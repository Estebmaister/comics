package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"comics/api/route"
	"comics/bootstrap"
	_ "comics/docs"
	"comics/internal/repo/sql/sqlite"

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
	app := bootstrap.MustLoadApp(ctx)

	// Creating a gin instance
	if app.Env.AppEnv.IsProduction() || app.Env.AppEnv.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}
	g := gin.New()
	g.Use(gin.Recovery())
	// Route binding
	route.Setup(app.Env, app.UserRepo, g)

	// Running the server
	srvErr := make(chan error, 1)
	go func() {
		// check if certs exist:
		certFile, keyFile := "../tls/comics.crt", "../tls/comics.key"
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			log.Warn().Msg("Failed to load X509 key pair, serving HTTP instead")
			srvErr <- g.Run(app.Env.AddressHTTP + ":" + app.Env.PortHTTP)
			return
		}
		srvErr <- g.RunTLS(app.Env.AddressHTTP+":"+app.Env.PortHTTP, certFile, keyFile)
	}()

	// Initialize the file comics database
	sqliteDBPath := "../src/db/comics.db"
	_, sqliteErr := sqlite.NewSQLiteUserRepo(sqliteDBPath)
	if sqliteErr != nil {
		log.Error().Err(sqliteErr).Msg("Failed to initialize SQLite database")
	}

	// Wait for interruption
	select {
	case err := <-srvErr:
		log.Error().Err(err).Msg("Error when starting HTTP server")
	case <-ctx.Done():
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// Create a new context for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), app.Env.InitCtxTimeout)
	defer cancel()
	log.Info().Msg("Shutting down application...")
	app.Close(shutdownCtx)
}
