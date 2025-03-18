package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// StartMetricsServer starts the metrics and health check server
func (s *Server) StartMetricsServer(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register metrics handler
	mux.Handle("/metrics", promhttp.Handler())

	// Register health check handlers
	mux.HandleFunc("/health/live", s.healthChecker.LivenessHandler())
	mux.HandleFunc("/health/ready", s.healthChecker.ReadinessHandler())

	// Start health checker
	s.healthChecker.Start()

	// Update server mux
	s.healthServer.Handler = mux

	// Start server in a goroutine
	go func() {
		log.Info().Msgf("Starting metrics server on port %d", s.config.MetricsPort)
		if err := s.healthServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Error().Err(err).Caller().Msg("Metrics server failed")
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		if err := s.healthServer.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Caller().Msg("Error shutting down metrics server")
		}
	}()

	return nil
}

// MetricsAddr returns the metrics server address
func (s *Server) MetricsAddr() string {
	return fmt.Sprintf(":%d", s.config.MetricsPort)
}
