// Copyright (c) 2026 Michael Lechner. All rights reserved.
package mlcartifact

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "github.com/hmsoft0815/mlcartifact/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	cli  pb.ArtifactServiceClient
}

// NewClient creates a new gRPC client for the artifact service.
// It reads ARTIFACT_GRPC_ADDR from env, defaulting to :9590.
func NewClient(opts ...grpc.DialOption) (*Client, error) {
	addr := os.Getenv("ARTIFACT_GRPC_ADDR")
	if addr == "" {
		addr = ":9590"
	}
	return NewClientWithAddr(addr, opts...)
}

// NewClientWithAddr creates a new gRPC client for the artifact service with a specific address.
func NewClientWithAddr(addr string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial artifact service at %s: %w", addr, err)
	}

	return &Client{
		conn: conn,
		cli:  pb.NewArtifactServiceClient(conn),
	}, nil
}

// NewClientWithService creates a client with a pre-configured gRPC service implementation.
// Useful for testing with mocks.
func NewClientWithService(service pb.ArtifactServiceClient) *Client {
	return &Client{
		cli: service,
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Write saves an artifact to the shared store.
func (c *Client) Write(ctx context.Context, filename string, content []byte, opts ...WriteOption) (*pb.WriteResponse, error) {
	req := &pb.WriteRequest{
		Filename: filename,
		Content:  content,
		Source:   os.Getenv("ARTIFACT_SOURCE"), // Default source from env
		UserId:   os.Getenv("ARTIFACT_USER_ID"),
	}

	for _, opt := range opts {
		opt(req)
	}

	return c.cli.Write(ctx, req)
}

// Read retrieves an artifact from the shared store.
func (c *Client) Read(ctx context.Context, idOrFilename string, opts ...ReadOption) (*pb.ReadResponse, error) {
	req := &pb.ReadRequest{
		Id:     idOrFilename,
		UserId: os.Getenv("ARTIFACT_USER_ID"),
	}
	for _, opt := range opts {
		opt(req)
	}

	return c.cli.Read(ctx, req)
}

// List returns all artifacts for the current user.
func (c *Client) List(ctx context.Context, userID string, opts ...ListOption) (*pb.ListResponse, error) {
	req := &pb.ListRequest{
		UserId: userID,
	}
	if req.UserId == "" {
		req.UserId = os.Getenv("ARTIFACT_USER_ID")
	}

	for _, opt := range opts {
		opt(req)
	}

	return c.cli.List(ctx, req)
}

// Delete removes an artifact from the shared store.
func (c *Client) Delete(ctx context.Context, idOrFilename string, opts ...DeleteOption) (*pb.DeleteResponse, error) {
	req := &pb.DeleteRequest{
		Id:     idOrFilename,
		UserId: os.Getenv("ARTIFACT_USER_ID"),
	}
	for _, opt := range opts {
		opt(req)
	}

	return c.cli.Delete(ctx, req)
}

type WriteOption func(*pb.WriteRequest)

func WithMimeType(mt string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.MimeType = mt
	}
}

func WithExpiresHours(h int32) WriteOption {
	return func(r *pb.WriteRequest) {
		r.ExpiresHours = h
	}
}

func WithSource(s string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Source = s
	}
}

func WithUserID(id string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.UserId = id
	}
}

func WithMetadata(m map[string]string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Metadata = m
	}
}

func WithDescription(d string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Description = d
	}
}

// Read options
type ReadOption func(*pb.ReadRequest)

func WithReadUserID(id string) ReadOption {
	return func(r *pb.ReadRequest) {
		r.UserId = id
	}
}

// List options
type ListOption func(*pb.ListRequest)

func WithLimit(limit int32) ListOption {
	return func(r *pb.ListRequest) {
		r.Limit = limit
	}
}

func WithOffset(offset int32) ListOption {
	return func(r *pb.ListRequest) {
		r.Offset = offset
	}
}

func WithListUserID(id string) ListOption {
	return func(r *pb.ListRequest) {
		r.UserId = id
	}
}

// Delete options
type DeleteOption func(*pb.DeleteRequest)

func WithDeleteUserID(id string) DeleteOption {
	return func(r *pb.DeleteRequest) {
		r.UserId = id
	}
}
