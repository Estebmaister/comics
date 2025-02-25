package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	_ "comics/internal/logger"
	"comics/internal/server"
	"comics/internal/tracing"

	"github.com/rs/zerolog/log"
)

func main() {
	// Create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tracer
	tp, err := tracing.NewTracer(ctx, tracing.TracerConfig{
		Endpoint:    "http://localhost:14268/api/traces",
		ServiceName: "comics-server",
		Sampler:     100,
	}, "grpc")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tracer")
	}
	defer tp.Shutdown(ctx)

	// Create server with default config
	srv, err := server.New(ctx, server.DefaultConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}
	defer srv.Shutdown(ctx)

	// Register gRPC services
	srv.RegisterServices()

	// Start metrics server
	if err := srv.StartMetricsServer(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start metrics server")
	}

	// Start gRPC server
	lis, err := srv.GRPCListener()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create listener")
	}

	go func() {
		log.Info().Msgf("Starting gRPC server on %s", lis.Addr().String())
		if err := srv.GRPCServer().Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
