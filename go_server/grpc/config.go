package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"comics/grpc/middleware"
	"comics/internal/health"
	"comics/internal/repo"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Config holds all server configuration
type Config struct {
	GRPCPort         int
	MetricsPort      int
	DatabaseURL      string
	JaegerURL        string
	EnableTracing    bool
	PathToServerCert string
	PathToServerKey  string
}

// DefaultConfig returns a default server configuration
func DefaultConfig() *Config {
	return &Config{
		GRPCPort:      50051,
		MetricsPort:   2112,
		EnableTracing: true,
	}
}

// loadTLSCredentials loads the TLS credentials from the certificate and key files
func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("path/to/server-cert.pem", "path/to/server-key.pem")
	if err != nil {
		return nil, fmt.Errorf("cannot load server key pair: %w", err)
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	creds := credentials.NewTLS(config)
	return creds, nil
}

// Server represents our gRPC server instance
type Server struct {
	config        *Config
	grpcServer    *grpc.Server
	healthServer  *http.Server
	comicsRepo    *repo.ComicsRepo
	healthChecker *health.Checker
}

// New creates a new server instance
func New(ctx context.Context, cfg *Config) (*Server, error) {
	// Create repo-db connection
	comicsRepo, err := repo.NewComicsRepo(ctx, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create health checker
	healthChecker := health.NewHealthChecker(comicsRepo.Client())

	// Load server TLS credentials
	creds, err := loadTLSCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(middleware.UnaryServerLoggingInterceptor()),
		grpc.StreamInterceptor(middleware.StreamServerLoggingInterceptor()),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	// Create metrics/health server
	mux := http.NewServeMux()
	healthServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	return &Server{
		config:        cfg,
		grpcServer:    grpcServer,
		healthServer:  healthServer,
		comicsRepo:    comicsRepo,
		healthChecker: healthChecker,
	}, nil
}

// GRPCListener creates and returns a network listener for gRPC
func (s *Server) GRPCListener() (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPCPort))
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) {
	log.Info().Msg("Shutting down server...")
	s.grpcServer.GracefulStop()
	s.healthChecker.Stop()
	if err := s.comicsRepo.Close(ctx); err != nil {
		log.Error().Err(err).Caller().Msg("Error closing database connection")
	}
}

// Database returns the server's database instance
func (s *Server) Database() *repo.ComicsRepo {
	return s.comicsRepo
}

// GRPCServer returns the gRPC server instance
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}
