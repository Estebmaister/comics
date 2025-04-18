// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: comics_service.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ComicService_CreateComic_FullMethodName     = "/comics.ComicService/CreateComic"
	ComicService_DeleteComic_FullMethodName     = "/comics.ComicService/DeleteComic"
	ComicService_UpdateComic_FullMethodName     = "/comics.ComicService/UpdateComic"
	ComicService_GetComicById_FullMethodName    = "/comics.ComicService/GetComicById"
	ComicService_GetComicByTitle_FullMethodName = "/comics.ComicService/GetComicByTitle"
	ComicService_GetComics_FullMethodName       = "/comics.ComicService/GetComics"
	ComicService_SearchComics_FullMethodName    = "/comics.ComicService/SearchComics"
)

// ComicServiceClient is the client API for ComicService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Comic Service definition
// Provides operations for managing and retrieving comics
type ComicServiceClient interface {
	// Creates a new comic in the system
	// Returns the created comic or error if validation fails
	CreateComic(ctx context.Context, in *CreateComicRequest, opts ...grpc.CallOption) (*ComicResponse, error)
	// Soft deletes a comic by marking it as deleted
	// Returns the updated comic or error if not found
	DeleteComic(ctx context.Context, in *DeleteComicRequest, opts ...grpc.CallOption) (*ComicResponse, error)
	// Updates an existing comic's information
	// Returns the updated comic or error if validation fails
	UpdateComic(ctx context.Context, in *UpdateComicRequest, opts ...grpc.CallOption) (*ComicResponse, error)
	// Retrieves a comic by its unique identifier
	// Returns the comic or error if not found
	GetComicById(ctx context.Context, in *GetComicByIdRequest, opts ...grpc.CallOption) (*ComicResponse, error)
	// Retrieves a comic by its title (exact match)
	// Returns the comic or error if not found
	GetComicByTitle(ctx context.Context, in *GetComicByTitleRequest, opts ...grpc.CallOption) (*ComicResponse, error)
	// Retrieves a paginated list of all comics
	// Returns the page of comics with total count information
	GetComics(ctx context.Context, in *GetComicsRequest, opts ...grpc.CallOption) (*ComicsResponse, error)
	// Searches comics using fuzzy title matching
	// Returns paginated results ordered by relevance
	SearchComics(ctx context.Context, in *SearchComicsRequest, opts ...grpc.CallOption) (*ComicsResponse, error)
}

type comicServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewComicServiceClient(cc grpc.ClientConnInterface) ComicServiceClient {
	return &comicServiceClient{cc}
}

func (c *comicServiceClient) CreateComic(ctx context.Context, in *CreateComicRequest, opts ...grpc.CallOption) (*ComicResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicResponse)
	err := c.cc.Invoke(ctx, ComicService_CreateComic_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *comicServiceClient) DeleteComic(ctx context.Context, in *DeleteComicRequest, opts ...grpc.CallOption) (*ComicResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicResponse)
	err := c.cc.Invoke(ctx, ComicService_DeleteComic_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *comicServiceClient) UpdateComic(ctx context.Context, in *UpdateComicRequest, opts ...grpc.CallOption) (*ComicResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicResponse)
	err := c.cc.Invoke(ctx, ComicService_UpdateComic_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *comicServiceClient) GetComicById(ctx context.Context, in *GetComicByIdRequest, opts ...grpc.CallOption) (*ComicResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicResponse)
	err := c.cc.Invoke(ctx, ComicService_GetComicById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *comicServiceClient) GetComicByTitle(ctx context.Context, in *GetComicByTitleRequest, opts ...grpc.CallOption) (*ComicResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicResponse)
	err := c.cc.Invoke(ctx, ComicService_GetComicByTitle_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *comicServiceClient) GetComics(ctx context.Context, in *GetComicsRequest, opts ...grpc.CallOption) (*ComicsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicsResponse)
	err := c.cc.Invoke(ctx, ComicService_GetComics_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *comicServiceClient) SearchComics(ctx context.Context, in *SearchComicsRequest, opts ...grpc.CallOption) (*ComicsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ComicsResponse)
	err := c.cc.Invoke(ctx, ComicService_SearchComics_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ComicServiceServer is the server API for ComicService service.
// All implementations must embed UnimplementedComicServiceServer
// for forward compatibility.
//
// Comic Service definition
// Provides operations for managing and retrieving comics
type ComicServiceServer interface {
	// Creates a new comic in the system
	// Returns the created comic or error if validation fails
	CreateComic(context.Context, *CreateComicRequest) (*ComicResponse, error)
	// Soft deletes a comic by marking it as deleted
	// Returns the updated comic or error if not found
	DeleteComic(context.Context, *DeleteComicRequest) (*ComicResponse, error)
	// Updates an existing comic's information
	// Returns the updated comic or error if validation fails
	UpdateComic(context.Context, *UpdateComicRequest) (*ComicResponse, error)
	// Retrieves a comic by its unique identifier
	// Returns the comic or error if not found
	GetComicById(context.Context, *GetComicByIdRequest) (*ComicResponse, error)
	// Retrieves a comic by its title (exact match)
	// Returns the comic or error if not found
	GetComicByTitle(context.Context, *GetComicByTitleRequest) (*ComicResponse, error)
	// Retrieves a paginated list of all comics
	// Returns the page of comics with total count information
	GetComics(context.Context, *GetComicsRequest) (*ComicsResponse, error)
	// Searches comics using fuzzy title matching
	// Returns paginated results ordered by relevance
	SearchComics(context.Context, *SearchComicsRequest) (*ComicsResponse, error)
	mustEmbedUnimplementedComicServiceServer()
}

// UnimplementedComicServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedComicServiceServer struct{}

func (UnimplementedComicServiceServer) CreateComic(context.Context, *CreateComicRequest) (*ComicResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateComic not implemented")
}
func (UnimplementedComicServiceServer) DeleteComic(context.Context, *DeleteComicRequest) (*ComicResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteComic not implemented")
}
func (UnimplementedComicServiceServer) UpdateComic(context.Context, *UpdateComicRequest) (*ComicResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateComic not implemented")
}
func (UnimplementedComicServiceServer) GetComicById(context.Context, *GetComicByIdRequest) (*ComicResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetComicById not implemented")
}
func (UnimplementedComicServiceServer) GetComicByTitle(context.Context, *GetComicByTitleRequest) (*ComicResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetComicByTitle not implemented")
}
func (UnimplementedComicServiceServer) GetComics(context.Context, *GetComicsRequest) (*ComicsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetComics not implemented")
}
func (UnimplementedComicServiceServer) SearchComics(context.Context, *SearchComicsRequest) (*ComicsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchComics not implemented")
}
func (UnimplementedComicServiceServer) mustEmbedUnimplementedComicServiceServer() {}
func (UnimplementedComicServiceServer) testEmbeddedByValue()                      {}

// UnsafeComicServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ComicServiceServer will
// result in compilation errors.
type UnsafeComicServiceServer interface {
	mustEmbedUnimplementedComicServiceServer()
}

func RegisterComicServiceServer(s grpc.ServiceRegistrar, srv ComicServiceServer) {
	// If the following call pancis, it indicates UnimplementedComicServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ComicService_ServiceDesc, srv)
}

func _ComicService_CreateComic_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateComicRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).CreateComic(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_CreateComic_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).CreateComic(ctx, req.(*CreateComicRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ComicService_DeleteComic_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteComicRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).DeleteComic(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_DeleteComic_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).DeleteComic(ctx, req.(*DeleteComicRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ComicService_UpdateComic_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateComicRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).UpdateComic(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_UpdateComic_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).UpdateComic(ctx, req.(*UpdateComicRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ComicService_GetComicById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetComicByIdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).GetComicById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_GetComicById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).GetComicById(ctx, req.(*GetComicByIdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ComicService_GetComicByTitle_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetComicByTitleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).GetComicByTitle(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_GetComicByTitle_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).GetComicByTitle(ctx, req.(*GetComicByTitleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ComicService_GetComics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetComicsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).GetComics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_GetComics_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).GetComics(ctx, req.(*GetComicsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ComicService_SearchComics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchComicsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ComicServiceServer).SearchComics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ComicService_SearchComics_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ComicServiceServer).SearchComics(ctx, req.(*SearchComicsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ComicService_ServiceDesc is the grpc.ServiceDesc for ComicService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ComicService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "comics.ComicService",
	HandlerType: (*ComicServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateComic",
			Handler:    _ComicService_CreateComic_Handler,
		},
		{
			MethodName: "DeleteComic",
			Handler:    _ComicService_DeleteComic_Handler,
		},
		{
			MethodName: "UpdateComic",
			Handler:    _ComicService_UpdateComic_Handler,
		},
		{
			MethodName: "GetComicById",
			Handler:    _ComicService_GetComicById_Handler,
		},
		{
			MethodName: "GetComicByTitle",
			Handler:    _ComicService_GetComicByTitle_Handler,
		},
		{
			MethodName: "GetComics",
			Handler:    _ComicService_GetComics_Handler,
		},
		{
			MethodName: "SearchComics",
			Handler:    _ComicService_SearchComics_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "comics_service.proto",
}
