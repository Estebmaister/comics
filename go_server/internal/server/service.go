package server

import (
	pb "comics/pb"

	"google.golang.org/grpc/reflection"
)

// RegisterServices registers all gRPC services
func (s *Server) RegisterServices() {
	// Register comics service
	pb.RegisterComicServiceServer(s.grpcServer, newComicsService(s.database))

	// Register reflection service for grpcurl
	reflection.Register(s.grpcServer)
}
