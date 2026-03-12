// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package grpc provides the gRPC and Connect RPC implementation of the
// ArtifactService. It acts as a bridge between the network protocol and
// the internal storage backend.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hmsoft0815/mlcartifact/cmd/server/internal/storage"
	pb "github.com/hmsoft0815/mlcartifact/proto"
)

// Server implements the ArtifactService gRPC interface.
type Server struct {
	pb.UnimplementedArtifactServiceServer
	Store *storage.Store // The underlying file storage backend
}

// NewServer creates a new gRPC server instance with the provided store.
func NewServer(store *storage.Store) *Server {
	return &Server{Store: store}
}

// Write handles the creation or update of an artifact.
// It maps the proto metadata and content to the storage.Write method.
func (s *Server) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteResponse, error) {
	slog.Info("gRPC Write request", "filename", req.Filename, "source", req.Source, "user_id", req.UserId)

	// Map proto metadata to map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	meta, err := s.Store.Write(
		req.Filename,
		req.Content,
		req.MimeType,
		int(req.ExpiresHours),
		req.Source,
		req.UserId,
		"", // description (to be added to proto later if needed, for now from MCP)
		metadata,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to write artifact: %w", err)
	}

	return &pb.WriteResponse{
		Id:        meta.ID,
		Filename:  meta.Filename,
		Uri:       fmt.Sprintf("artifact://%s", meta.Filename),
		ExpiresAt: meta.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// Read retrieves an artifact's content and metadata by ID or filename.
func (s *Server) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	slog.Info("gRPC Read request", "id", req.Id, "user_id", req.UserId)

	content, meta, err := s.Store.Read(req.Id, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	return &pb.ReadResponse{
		Content:  content,
		MimeType: meta.MimeType,
		Filename: meta.Filename,
	}, nil
}

// Delete removes an artifact permanently.
func (s *Server) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	slog.Info("gRPC Delete request", "id", req.Id, "user_id", req.UserId)
	deleted, err := s.Store.Delete(req.Id, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete artifact: %w", err)
	}
	return &pb.DeleteResponse{Deleted: deleted}, nil
}

// List returns a paginated list of artifacts for a user.
func (s *Server) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	slog.Info("gRPC List request", "user_id", req.UserId, "limit", req.Limit, "offset", req.Offset)
	items, err := s.Store.List(req.UserId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	var pbItems []*pb.ArtifactInfo
	for _, item := range items {
		pbItems = append(pbItems, &pb.ArtifactInfo{
			Id:        item.ID,
			Filename:  item.Filename,
			MimeType:  item.MimeType,
			Source:    item.Source,
			UserId:    item.UserID,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
			ExpiresAt: item.ExpiresAt.Format(time.RFC3339),
		})
	}

	return &pb.ListResponse{Items: pbItems}, nil
}
