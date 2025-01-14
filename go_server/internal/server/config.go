package server

import (
	"fmt"
	"net"
	"net/http"

	"comics/internal/db"
	"comics/internal/health"
	"comics/internal/middleware"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// Config holds all server configuration
type Config struct {
	GRPCPort      int
	MetricsPort   int
	DatabaseURL   string
	JaegerURL     string
	EnableTracing bool
}

// DefaultConfig returns a default server configuration
func DefaultConfig() *Config {
	return &Config{
		GRPCPort:      50051,
		MetricsPort:   2112,
		EnableTracing: true,
	}
}

// Server represents our gRPC server instance
type Server struct {
	config       *Config
	grpcServer   *grpc.Server
	healthServer *http.Server
	database     *db.Database
	health       *health.HealthChecker
}

// New creates a new server instance
func New(cfg *Config) (*Server, error) {
	// Create database connection
	database, err := db.NewDatabase(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create health checker
	healthChecker := health.NewHealthChecker(database.DB())

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.UnaryServerLoggingInterceptor()),
		grpc.StreamInterceptor(middleware.StreamServerLoggingInterceptor()),
	)

	// Create metrics/health server
	mux := http.NewServeMux()
	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler: mux,
	}

	return &Server{
		config:       cfg,
		grpcServer:   grpcServer,
		healthServer: healthServer,
		database:     database,
		health:       healthChecker,
	}, nil
}

// GRPCListener creates and returns a network listener for gRPC
func (s *Server) GRPCListener() (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPCPort))
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	log.Info().Msg("Shutting down server...")
	s.grpcServer.GracefulStop()
	s.health.Stop()
	if err := s.database.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing database connection")
	}
}

// Database returns the server's database instance
func (s *Server) Database() *db.Database {
	return s.database
}

// GRPCServer returns the gRPC server instance
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}
