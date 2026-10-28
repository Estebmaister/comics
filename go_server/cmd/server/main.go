package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	healthy := route.Setup(app.Env, app.UserRepo, app.ComicService, g)

	if app.Env.SQLiteWALCheckpointInterval > 0 {
		interval := app.Env.SQLiteWALCheckpointInterval
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					err := app.ComicService.CheckpointWAL(checkCtx)
					cancel()
					if err != nil {
						log.Warn().Err(err).Dur("interval", interval).Msg("Periodic SQLite WAL checkpoint failed")
					} else {
						log.Debug().Dur("interval", interval).Msg("Periodic SQLite WAL checkpoint completed")
					}
				}
			}
		}()
		log.Info().Dur("interval", interval).Msg("SQLite WAL checkpoint scheduler enabled")
	}

	if app.Env.ScrapeInterval > 0 {
		scheduler := scrape.NewPeriodicScheduler(
			scrape.PeriodicConfig{
				Interval:     app.Env.ScrapeInterval,
				StartupGrace: app.Env.ScrapeStartupGrace,
			},
			app.ScrapeCoord,
		)
		scheduler.Start(ctx)
		log.Info().
			Dur("interval", app.Env.ScrapeInterval).
			Dur("startup_grace", app.Env.ScrapeStartupGrace).
			Msg("Background scrape scheduler enabled")
	}

	addr := app.Env.AddressHTTP + ":" + app.Env.PortHTTP
	srv := &http.Server{
		Addr:              addr,
		Handler:           g,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		certFile, keyFile := app.Env.HTTPTLSCertFile, app.Env.HTTPTLSKeyFile
		if app.Env.HasHTTPSTLSFiles() {
			log.Info().Str("url", "https://"+addr).Msg("Serving HTTPS")
			srvErr <- srv.ListenAndServeTLS(certFile, keyFile)
			return
		}
		if certFile != "" || keyFile != "" {
			log.Warn().
				Str("cert", certFile).
				Str("key", keyFile).
				Msg("TLS certificate or key unavailable, serving HTTP instead")
		}
		log.Info().Str("url", "http://"+addr).Msg("Serving HTTP")
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption
	select {
	case err := <-srvErr:
		log.Error().Err(err).Msg("Error when starting HTTP server")
	case <-ctx.Done():
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), 10*time.Minute)
	defer cancel()
	log.Info().Msg("Shutting down application...")
	if healthy != nil {
		healthy.Stop()
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}
	app.Close(shutdownCtx)
}
