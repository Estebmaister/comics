package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"comics/api/route"
	"comics/bootstrap"
	_ "comics/docs"
	"comics/internal/scrape"

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
	route.Setup(app.Env, app.UserRepo, app.ComicService, g)

	if app.Env.ScrapeInterval > 0 {
		scheduler := scrape.NewPeriodicScheduler(
			scrape.PeriodicConfig{Interval: app.Env.ScrapeInterval},
			app.ScrapeCoord,
		)
		scheduler.Start(ctx)
		log.Info().Dur("interval", app.Env.ScrapeInterval).Msg("Background scrape scheduler enabled")
	}

	// Running the server
	srvErr := make(chan error, 1)
	go func() {
		certFile, keyFile := app.Env.HTTPTLSCertFile, app.Env.HTTPTLSKeyFile
		addr := app.Env.AddressHTTP + ":" + app.Env.PortHTTP
		if app.Env.HasHTTPSTLSFiles() {
			log.Info().Str("url", "https://"+addr).Msg("Serving HTTPS")
			srvErr <- g.RunTLS(addr, certFile, keyFile)
			return
		}
		if certFile != "" || keyFile != "" {
			log.Warn().
				Str("cert", certFile).
				Str("key", keyFile).
				Msg("TLS certificate or key unavailable, serving HTTP instead")
		}
		log.Info().Str("url", "http://"+addr).Msg("Serving HTTP")
		srvErr <- g.Run(addr)
	}()

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
