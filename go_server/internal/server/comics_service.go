package server

import (
	"context"
	"fmt"
	"time"

	"comics/internal/db"
	pb "comics/pb"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	tracer = otel.Tracer("comics-service")

	// Metrics
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "comics_request_duration_seconds",
			Help:    "Duration of comic service requests in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"method", "status"},
	)

	requestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "comics_requests_total",
			Help: "Total number of comic service requests",
		},
		[]string{"method", "status"},
	)
)

func init() {
	prometheus.MustRegister(requestDuration, requestTotal)
}

// comicsService implements the ComicService gRPC service
type comicsService struct {
	pb.UnimplementedComicServiceServer
	db *db.Database
}

// newComicsService creates a new comics service instance
func newComicsService(db *db.Database) *comicsService {
	return &comicsService{db: db}
}

// createResponseMetadata creates response metadata with timing information
func createResponseMetadata(ctx context.Context, startTime time.Time, code codes.Code) *pb.ResponseMetadata {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	return &pb.ResponseMetadata{
		RequestId:     getRequestID(ctx),
		StartTime:     timestamppb.New(startTime),
		EndTime:       timestamppb.Now(),
		StatusCode:    uint32(code),
		StatusMessage: code.String(),
		TraceId:       &traceID,
	}
}

// handleError creates an error response with proper metadata
func handleError(ctx context.Context, startTime time.Time, err error) (*pb.ComicResponse, error) {
	st, _ := status.FromError(err)
	errorMsg := st.Message()
	return &pb.ComicResponse{
		Metadata: createResponseMetadata(ctx, startTime, st.Code()),
		Error:    &errorMsg,
	}, err
}

// CreateComic implements the CreateComic RPC method
func (s *comicsService) CreateComic(ctx context.Context, req *pb.CreateComicRequest) (*pb.ComicResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "CreateComic")
	defer span.End()

	// Add request attributes to span
	span.SetAttributes(
		attribute.String("comic.title", req.Comic.Titles[0]),
		attribute.String("comic.author", req.Comic.Author),
	)

	// Validate request
	if err := validateCreateComicRequest(req); err != nil {
		return handleError(ctx, startTime, err)
	}

	// Create comic in database
	err := s.db.CreateComic(ctx, req.Comic)
	if err != nil {
		return handleError(ctx, startTime, fmt.Errorf("failed to create comic: %w", err))
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("CreateComic", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("CreateComic", codes.OK.String()).Inc()

	return &pb.ComicResponse{
		Metadata: createResponseMetadata(ctx, startTime, codes.OK),
		Comic:    req.Comic,
	}, nil
}

// GetComicById implements the GetComicById RPC method
func (s *comicsService) GetComicById(ctx context.Context, req *pb.GetComicByIdRequest) (*pb.ComicResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "GetComicById")
	defer span.End()

	span.SetAttributes(attribute.Int("comic.id", int(req.Id)))

	// Validate request
	if req.Id <= 0 {
		return handleError(ctx, startTime, status.Error(codes.InvalidArgument, "invalid comic ID"))
	}

	// Get comic from database
	comic, err := s.db.GetComicById(ctx, req.Id)
	if err != nil {
		if err.Error() == "comic not found" {
			return handleError(ctx, startTime, status.Error(codes.NotFound, "comic not found"))
		}
		return handleError(ctx, startTime, fmt.Errorf("failed to get comic: %w", err))
	}

	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("GetComicById", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("GetComicById", codes.OK.String()).Inc()

	return &pb.ComicResponse{
		Metadata: createResponseMetadata(ctx, startTime, codes.OK),
		Comic:    comic,
	}, nil
}

// GetComics implements the GetComics RPC method
func (s *comicsService) GetComics(ctx context.Context, req *pb.GetComicsRequest) (*pb.ComicsResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "GetComics")
	defer span.End()

	// Set pagination attributes
	span.SetAttributes(
		attribute.Int("page", int(req.Pagination.Page)),
		attribute.Int("page_size", int(req.Pagination.PageSize)),
	)

	// Validate pagination
	if err := validatePagination(req.Pagination); err != nil {
		errMsg := err.Error()
		return &pb.ComicsResponse{
			Metadata: createResponseMetadata(ctx, startTime, codes.InvalidArgument),
			Error:    &errMsg,
		}, err
	}

	// Get comics from database
	comics, total, err := s.db.GetComics(ctx, int(req.Pagination.Page), int(req.Pagination.PageSize), false, false)
	if err != nil {
		errMsg := err.Error()
		return &pb.ComicsResponse{
			Metadata: createResponseMetadata(ctx, startTime, codes.Internal),
			Error:    &errMsg,
		}, err
	}

	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("GetComics", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("GetComics", codes.OK.String()).Inc()

	// Calculate pagination info
	totalCount := uint32(total)
	totalPages := (totalCount + req.Pagination.PageSize - 1) / req.Pagination.PageSize

	return &pb.ComicsResponse{
		Metadata:    createResponseMetadata(ctx, startTime, codes.OK),
		Comics:      comics,
		TotalCount:  &totalCount,
		TotalPages:  &totalPages,
		CurrentPage: &req.Pagination.Page,
	}, nil
}

// SearchComics implements the SearchComics RPC method
func (s *comicsService) SearchComics(ctx context.Context, req *pb.SearchComicsRequest) (*pb.ComicsResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "SearchComics")
	defer span.End()

	span.SetAttributes(
		attribute.String("query", req.Query),
		attribute.Int("page", int(req.Pagination.Page)),
		attribute.Int("page_size", int(req.Pagination.PageSize)),
	)

	// Validate request
	if err := validateSearchRequest(req); err != nil {
		errMsg := err.Error()
		return &pb.ComicsResponse{
			Metadata: createResponseMetadata(ctx, startTime, codes.InvalidArgument),
			Error:    &errMsg,
		}, err
	}

	// Search comics in database
	comics, total, err := s.db.SearchComics(ctx, req.Query, int(req.Pagination.Page), int(req.Pagination.PageSize))
	if err != nil {
		errMsg := err.Error()
		return &pb.ComicsResponse{
			Metadata: createResponseMetadata(ctx, startTime, codes.Internal),
			Error:    &errMsg,
		}, err
	}

	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("SearchComics", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("SearchComics", codes.OK.String()).Inc()

	totalCount := uint32(total)
	totalPages := (totalCount + req.Pagination.PageSize - 1) / req.Pagination.PageSize

	return &pb.ComicsResponse{
		Metadata:    createResponseMetadata(ctx, startTime, codes.OK),
		Comics:      comics,
		TotalCount:  &totalCount,
		TotalPages:  &totalPages,
		CurrentPage: &req.Pagination.Page,
	}, nil
}

// UpdateComic implements the UpdateComic RPC method
func (s *comicsService) UpdateComic(ctx context.Context, req *pb.UpdateComicRequest) (*pb.ComicResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "UpdateComic")
	defer span.End()

	span.SetAttributes(
		attribute.Int("comic.id", int(req.Id)),
		attribute.String("comic.title", req.Comic.Titles[0]),
	)

	// Validate request
	if err := validateUpdateComicRequest(req); err != nil {
		return handleError(ctx, startTime, err)
	}

	// Update comic in database
	err := s.db.UpdateComic(ctx, req.Comic)
	if err != nil {
		if err.Error() == "comic not found" {
			return handleError(ctx, startTime, status.Error(codes.NotFound, "comic not found"))
		}
		return handleError(ctx, startTime, fmt.Errorf("failed to update comic: %w", err))
	}

	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("UpdateComic", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("UpdateComic", codes.OK.String()).Inc()

	return &pb.ComicResponse{
		Metadata: createResponseMetadata(ctx, startTime, codes.OK),
		Comic:    req.Comic,
	}, nil
}

// DeleteComic implements the DeleteComic RPC method
func (s *comicsService) DeleteComic(ctx context.Context, req *pb.DeleteComicRequest) (*pb.ComicResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "DeleteComic")
	defer span.End()

	span.SetAttributes(attribute.Int("comic.id", int(req.Id)))

	// Validate request
	if req.Id <= 0 {
		return handleError(ctx, startTime, status.Error(codes.InvalidArgument, "invalid comic ID"))
	}

	// Delete comic from database
	err := s.db.DeleteComic(ctx, req.Id)
	if err != nil {
		if err.Error() == "comic not found" {
			return handleError(ctx, startTime, status.Error(codes.NotFound, "comic not found"))
		}
		return handleError(ctx, startTime, fmt.Errorf("failed to delete comic: %w", err))
	}

	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("DeleteComic", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("DeleteComic", codes.OK.String()).Inc()

	return &pb.ComicResponse{
		Metadata: createResponseMetadata(ctx, startTime, codes.OK),
	}, nil
}

// GetComicByTitle implements the GetComicByTitle RPC method
func (s *comicsService) GetComicByTitle(ctx context.Context, req *pb.GetComicByTitleRequest) (*pb.ComicResponse, error) {
	startTime := time.Now()
	ctx, span := tracer.Start(ctx, "GetComicByTitle")
	defer span.End()

	span.SetAttributes(attribute.String("comic.title", req.Title))

	// Validate request
	if req.Title == "" {
		return handleError(ctx, startTime, status.Error(codes.InvalidArgument, "title cannot be empty"))
	}

	// Get comic from database
	comic, err := s.db.GetComicByTitle(ctx, req.Title)
	if err != nil {
		if err.Error() == "comic not found" {
			return handleError(ctx, startTime, status.Error(codes.NotFound, "comic not found"))
		}
		return handleError(ctx, startTime, fmt.Errorf("failed to get comic: %w", err))
	}

	duration := time.Since(startTime).Seconds()
	requestDuration.WithLabelValues("GetComicByTitle", codes.OK.String()).Observe(duration)
	requestTotal.WithLabelValues("GetComicByTitle", codes.OK.String()).Inc()

	return &pb.ComicResponse{
		Metadata: createResponseMetadata(ctx, startTime, codes.OK),
		Comic:    comic,
	}, nil
}
