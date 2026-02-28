// Copyright (c) 2026 Michael Lechner. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// Package mlcartifact provides a gRPC client for the artifact storage service.
//
// The artifact service allows AI agents and tools to persist files (reports,
// code, data) across tool invocations via a shared storage backend. Artifacts
// are identified by a unique ID and can be scoped to a specific user.
//
// # Basic Usage
//
//	client, err := mlcartifact.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	resp, err := client.Write(ctx, "report.md", []byte("# Hello"),
//	    mlcartifact.WithMimeType("text/markdown"),
//	)
//
// # Configuration
//
// The gRPC target address is read from the ARTIFACT_GRPC_ADDR environment
// variable. If not set, it defaults to ":9590".
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

// Close releases the underlying gRPC connection. Must be called when the
// client is no longer needed to avoid resource leaks.
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

// WriteOption configures a WriteRequest before it is sent to the server.
type WriteOption func(*pb.WriteRequest)

// WithMimeType sets the MIME type for the artifact. If not provided, the
// server will attempt to detect it from the filename extension.
func WithMimeType(mt string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.MimeType = mt
	}
}

// WithExpiresHours sets the number of hours until the artifact is
// automatically deleted by the server's cleanup routine.
func WithExpiresHours(h int32) WriteOption {
	return func(r *pb.WriteRequest) {
		r.ExpiresHours = h
	}
}

// WithSource tags the artifact with a source identifier (e.g. the tool or
// agent that created it). Overrides the ARTIFACT_SOURCE environment variable.
func WithSource(s string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Source = s
	}
}

// WithUserID scopes the artifact to a specific user. Overrides the
// ARTIFACT_USER_ID environment variable.
func WithUserID(id string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.UserId = id
	}
}

// WithMetadata attaches arbitrary key-value metadata to the artifact.
func WithMetadata(m map[string]string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Metadata = m
	}
}

// WithDescription adds a human-readable description to the artifact.
func WithDescription(d string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Description = d
	}
}

// ReadOption configures a ReadRequest before it is sent to the server.
type ReadOption func(*pb.ReadRequest)

// WithReadUserID scopes the read operation to a specific user's storage.
func WithReadUserID(id string) ReadOption {
	return func(r *pb.ReadRequest) {
		r.UserId = id
	}
}

// ListOption configures a ListRequest before it is sent to the server.
type ListOption func(*pb.ListRequest)

// WithLimit sets the maximum number of artifacts to return.
func WithLimit(limit int32) ListOption {
	return func(r *pb.ListRequest) {
		r.Limit = limit
	}
}

// WithOffset skips the first n artifacts in the result (for pagination).
func WithOffset(offset int32) ListOption {
	return func(r *pb.ListRequest) {
		r.Offset = offset
	}
}

// WithListUserID scopes the list operation to a specific user's storage.
func WithListUserID(id string) ListOption {
	return func(r *pb.ListRequest) {
		r.UserId = id
	}
}

// DeleteOption configures a DeleteRequest before it is sent to the server.
type DeleteOption func(*pb.DeleteRequest)

// WithDeleteUserID scopes the delete operation to a specific user's storage.
func WithDeleteUserID(id string) DeleteOption {
	return func(r *pb.DeleteRequest) {
		r.UserId = id
	}
}
