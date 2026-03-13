// Copyright (c) 2026 Michael Lechner. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// Package mlcartifact provides a high-level gRPC client for the mlcartifact service.
//
// The service allows AI agents and tools to persist artifacts (files, reports,
// data) in a shared storage backend. This enables state persistence and
// data sharing across different tool invocations or even different agents.
//
// # Connection Handling
//
// The client uses gRPC for communication. By default, it connects to ":9590"
// using insecure credentials, which is suitable for local development or
// internal network usage.
//
// # Configuration via Environment Variables
//
// Several environment variables are automatically respected by the client:
//
//   - ARTIFACT_GRPC_ADDR: The address of the gRPC server (default: ":9590").
//   - ARTIFACT_SOURCE: A default identifier for the source of artifacts (e.g. "my-agent").
//   - ARTIFACT_USER_ID: A default user ID to scope all operations to.
//
// # Scoping and Ownership
//
// Artifacts can be "global" (accessible to everyone) or scoped to a "UserID".
// If a UserID is provided (via environment or options), the server ensures
// that operations are restricted to that user's private storage area.
package mlcartifact

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strings"

	connect "connectrpc.com/connect"
	pb "github.com/hmsoft0815/mlcartifact/proto"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"golang.org/x/net/http2"
)

// Version is the current version of the library.
const Version = "0.3.1"

// Client is a gRPC/Connect client for the artifact service. It is thread-safe and can
// be shared across multiple goroutines.
type Client struct {
	httpClient *http.Client
	cli        protoconnect.ArtifactServiceClient
}

// NewClient creates a new client using settings from environment variables.
// It reads ARTIFACT_GRPC_ADDR, defaulting to ":9590" if not set.
func NewClient() (*Client, error) {
	addr := os.Getenv("ARTIFACT_GRPC_ADDR")
	if addr == "" {
		addr = ":9590"
	}
	return NewClientWithAddr(addr)
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*clientSettings)

type clientSettings struct {
	httpClient *http.Client
}

// WithHTTPClient provides a custom http.Client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(s *clientSettings) {
		s.httpClient = c
	}
}

// NewClientWithAddr creates a new client for a specific server address.
// It automatically supports HTTP/2 (H2C) and falls back to HTTP/1.1 if needed.
func NewClientWithAddr(addr string, opts ...ClientOption) (*Client, error) {
	settings := &clientSettings{}
	for _, opt := range opts {
		opt(settings)
	}

	if settings.httpClient == nil {
		// Default client with H2C support for cleartext HTTP/2
		settings.httpClient = &http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, network, addr)
				},
			},
		}
	}

	// Ensure addr has a scheme for Connect
	baseURL := addr
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return &Client{
		httpClient: settings.httpClient,
		cli:        protoconnect.NewArtifactServiceClient(settings.httpClient, baseURL),
	}, nil
}

// NewClientWithService creates a client wrapping an existing service implementation.
// This is primarily used for unit testing with mocked services.
func NewClientWithService(service protoconnect.ArtifactServiceClient) *Client {
	return &Client{
		cli: service,
	}
}

// Close is a no-op for the Connect client as it uses the shared http.Client.
func (c *Client) Close() error {
	return nil
}

// Write saves an artifact to the store.
//
// The filename should include a relevant extension (e.g., ".md", ".json") as the
// server uses it for MIME type detection if [WithMimeType] is not provided.
//
// The content is saved as raw bytes. By default, the client uses ARTIFACT_SOURCE
// and ARTIFACT_USER_ID environment variables for tagging and scoping.
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

	res, err := c.cli.Write(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// Read retrieves an artifact by its unique ID or its filename.
//
// If a filename is provided and multiple artifacts exist with that name, the
// server returns the most recent one. Use [WithReadUserID] to scope the lookup
// to a specific user's artifacts.
func (c *Client) Read(ctx context.Context, idOrFilename string, opts ...ReadOption) (*pb.ReadResponse, error) {
	req := &pb.ReadRequest{
		Id:     idOrFilename,
		UserId: os.Getenv("ARTIFACT_USER_ID"),
	}
	for _, opt := range opts {
		opt(req)
	}

	res, err := c.cli.Read(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// List returns metadata for artifacts, optionally filtered by user ID.
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

	res, err := c.cli.List(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// Delete removes an artifact and its metadata from the store.
func (c *Client) Delete(ctx context.Context, idOrFilename string, opts ...DeleteOption) (*pb.DeleteResponse, error) {
	req := &pb.DeleteRequest{
		Id:     idOrFilename,
		UserId: os.Getenv("ARTIFACT_USER_ID"),
	}
	for _, opt := range opts {
		opt(req)
	}

	res, err := c.cli.Delete(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// WriteOption is a functional option for configuring Write requests.
type WriteOption func(*pb.WriteRequest)

// WithMimeType explicitly sets the MIME type of the artifact.
func WithMimeType(mt string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.MimeType = mt
	}
}

// WithExpiresHours sets the time-to-live for the artifact in hours.
func WithExpiresHours(h int32) WriteOption {
	return func(r *pb.WriteRequest) {
		r.ExpiresHours = h
	}
}

// WithSource overrides the default source identifier.
func WithSource(s string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Source = s
	}
}

// WithUserID overrides the default user ID for the write operation.
func WithUserID(id string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.UserId = id
	}
}

// WithMetadata attaches custom metadata to the artifact.
func WithMetadata(m map[string]string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Metadata = m
	}
}

// WithDescription adds a descriptive text to the artifact metadata.
func WithDescription(d string) WriteOption {
	return func(r *pb.WriteRequest) {
		r.Description = d
	}
}

// ReadOption is a functional option for configuring Read requests.
type ReadOption func(*pb.ReadRequest)

// WithReadUserID specifies the user ID for the read operation.
func WithReadUserID(id string) ReadOption {
	return func(r *pb.ReadRequest) {
		r.UserId = id
	}
}

// ListOption is a functional option for configuring List requests.
type ListOption func(*pb.ListRequest)

// WithLimit restricts the number of items returned.
func WithLimit(limit int32) ListOption {
	return func(r *pb.ListRequest) {
		r.Limit = limit
	}
}

// WithOffset specifies the starting point for pagination.
func WithOffset(offset int32) ListOption {
	return func(r *pb.ListRequest) {
		r.Offset = offset
	}
}

// WithListUserID specifies the user ID for the list operation.
func WithListUserID(id string) ListOption {
	return func(r *pb.ListRequest) {
		r.UserId = id
	}
}

// DeleteOption is a functional option for configuring Delete requests.
type DeleteOption func(*pb.DeleteRequest)

// WithDeleteUserID specifies the user ID for the delete operation.
func WithDeleteUserID(id string) DeleteOption {
	return func(r *pb.DeleteRequest) {
		r.UserId = id
	}
}
