package server

import (
	"context"
	"fmt"
	"time"

	pb "comics/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxPageSize     = 100
	minTitleLength  = 1
	minSearchLength = 2
)

// validateCreateComicRequest validates a create comic request
func validateCreateComicRequest(req *pb.CreateComicRequest) error {
	if req.Comic == nil {
		return status.Error(codes.InvalidArgument, "comic is required")
	}

	if len(req.Comic.Titles) == 0 {
		return status.Error(codes.InvalidArgument, "at least one title is required")
	}

	for i, title := range req.Comic.Titles {
		if len(title) < minTitleLength {
			return status.Errorf(codes.InvalidArgument, "title at index %d is too short", i)
		}
	}

	if req.Comic.Author == "" {
		return status.Error(codes.InvalidArgument, "author is required")
	}

	if req.Comic.CurrentChap < 0 {
		return status.Error(codes.InvalidArgument, "current chapter cannot be negative")
	}

	if req.Comic.ViewedChap < 0 {
		return status.Error(codes.InvalidArgument, "viewed chapter cannot be negative")
	}

	return nil
}

// validateUpdateComicRequest validates an update comic request
func validateUpdateComicRequest(req *pb.UpdateComicRequest) error {
	if req.Id <= 0 {
		return status.Error(codes.InvalidArgument, "invalid comic ID")
	}

	if req.Comic == nil {
		return status.Error(codes.InvalidArgument, "comic is required")
	}

	if len(req.Comic.Titles) == 0 {
		return status.Error(codes.InvalidArgument, "at least one title is required")
	}

	for i, title := range req.Comic.Titles {
		if len(title) < minTitleLength {
			return status.Errorf(codes.InvalidArgument, "title at index %d is too short", i)
		}
	}

	if req.Comic.CurrentChap < 0 {
		return status.Error(codes.InvalidArgument, "current chapter cannot be negative")
	}

	if req.Comic.ViewedChap < 0 {
		return status.Error(codes.InvalidArgument, "viewed chapter cannot be negative")
	}

	return nil
}

// validatePagination validates pagination parameters
func validatePagination(pagination *pb.PaginationRequest) error {
	if pagination == nil {
		return status.Error(codes.InvalidArgument, "pagination is required")
	}

	if pagination.Page <= 0 {
		return status.Error(codes.InvalidArgument, "page must be greater than 0")
	}

	if pagination.PageSize <= 0 || pagination.PageSize > maxPageSize {
		return status.Errorf(codes.InvalidArgument, "page size must be between 1 and %d", maxPageSize)
	}

	return nil
}

// validateSearchRequest validates a search request
func validateSearchRequest(req *pb.SearchComicsRequest) error {
	if req.Query == "" {
		return status.Error(codes.InvalidArgument, "search query is required")
	}

	if len(req.Query) < minSearchLength {
		return status.Errorf(codes.InvalidArgument, "search query must be at least %d characters", minSearchLength)
	}

	if err := validatePagination(req.Pagination); err != nil {
		return err
	}

	return nil
}

// validateSortOrder validates the sort order
func validateSortOrder(order pb.ComicSortOrder) error {
	switch order {
	case pb.ComicSortOrder_UNSPECIFIED,
		pb.ComicSortOrder_TITLE_ASC,
		pb.ComicSortOrder_TITLE_DESC,
		pb.ComicSortOrder_UPDATED_ASC,
		pb.ComicSortOrder_UPDATED_DESC,
		pb.ComicSortOrder_RELEVANCE:
		return nil
	default:
		return status.Error(codes.InvalidArgument, "invalid sort order")
	}
}

// getRequestID extracts the request ID from the metadata
func getRequestID(ctx context.Context) string {
	if md, ok := ctx.Value("metadata").(*pb.RequestMetadata); ok && md != nil {
		return md.RequestId
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
