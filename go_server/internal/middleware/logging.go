package middleware

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerLoggingInterceptor returns a new unary server interceptor for logging
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Initialize the logger
		logger := log.With().
			Str("method", info.FullMethod).
			Str("request_id", getRequestID(ctx)).
			Logger()

		// Log the request
		logger.Info().
			Interface("request", req).
			Msg("Received request")

		resp, err := handler(ctx, req)

		// Get the status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
		}

		// Log the response
		logger.Info().
			Dur("duration", time.Since(start)).
			Str("status", statusCode.String()).
			Interface("response", resp).
			Err(err).
			Msg("Completed request")

		return resp, err
	}
}

// StreamServerLoggingInterceptor returns a new stream server interceptor for logging
func StreamServerLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Initialize the logger
		logger := log.With().
			Str("method", info.FullMethod).
			Str("request_id", getRequestID(ss.Context())).
			Logger()

		// Log the stream start
		logger.Info().Msg("Starting stream")

		err := handler(srv, ss)

		// Log the stream end
		logger.Info().
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("Ending stream")

		return err
	}
}

// getRequestID extracts request ID from context or generates a new one
func getRequestID(ctx context.Context) string {
	// You can implement your own request ID extraction logic here
	// For now, we'll return a timestamp-based ID
	return time.Now().Format("20060102150405")
}
