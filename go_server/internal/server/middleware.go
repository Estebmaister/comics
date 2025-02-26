package server

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	logger := log.With().Str("method", info.FullMethod).Logger()
	newCtx := logger.WithContext(ctx)

	h, err := handler(newCtx, req)
	duration := time.Since(start)

	logger.Info().Err(err).Dur("duration", duration).Msg("gRPC request")

	return h, err
}

func SensitiveDataInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	// Example: Remove sensitive data from logs
	if info.FullMethod == "/package.Service/Login" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			md.Delete("authorization")
		}
	}
	return handler(ctx, req)
}

// TODO: Add logging interceptor and check following format

// lis, err := net.Listen("tcp", cfg.GRPCServerURL)
// if err != nil {
// 		logger.Fatal().Err(err).Msg("Failed to listen")
// }

// grpcServer := grpc.NewServer(
// 		grpc.UnaryInterceptor(loggingInterceptor),
// )

// // Register your gRPC services here
// if err := grpcServer.Serve(lis); err != nil {
// 		logger.Fatal().Err(err).Msg("Failed to serve gRPC server")
// }
