// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package grpc provides the gRPC and Connect RPC implementation of the
// ArtifactService. It acts as a bridge between the network protocol and
// the internal storage backend.
package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/hmsoft0815/mlcartifact/internal/auth"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
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

// resolveUserID returns the verified UserID from the authenticated identity if present,
// otherwise falling back to the requested user ID.
func resolveUserID(ctx context.Context, requested string) string {
	if id := auth.AuthIdentityFromContext(ctx); id != nil && id.UserID != "" {
		return id.UserID
	}
	return requested
}

// Write handles the creation or update of an artifact.
// It maps the proto metadata and content to the storage.Write method.
func (s *Server) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteResponse, error) {
	userID := resolveUserID(ctx, req.UserId)
	slog.Info("gRPC Write request", "filename", req.Filename, "vpath", req.VirtualPath, "user_id", userID)

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
		userID,
		req.Description,
		metadata,
		req.VirtualPath,
	)

	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserID) || errors.Is(err, storage.ErrPathEscape) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
		}
		slog.Error("failed to write artifact", "error", err, "user_id", userID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to write artifact"))
	}

	return &pb.WriteResponse{
		Id:          meta.ID,
		Filename:    meta.Filename,
		Uri:         fmt.Sprintf("mlcartifact://%s", meta.ID),
		ExpiresAt:   meta.ExpiresAt.Format(time.RFC3339),
		VirtualPath: meta.VirtualPath,
	}, nil
}

// Read retrieves an artifact's content and metadata by ID, filename or virtual path.
func (s *Server) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	userID := resolveUserID(ctx, req.UserId)
	slog.Info("gRPC Read request", "id", req.Id, "user_id", userID)

	content, meta, err := s.Store.Read(req.Id, userID)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserID) || errors.Is(err, storage.ErrPathEscape) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
		}
		if errors.Is(err, storage.ErrArtifactNotFound) || err.Error() == "artifact not found" {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("artifact not found"))
		}
		slog.Error("failed to read artifact", "error", err, "id", req.Id, "user_id", userID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to read artifact"))
	}

	return &pb.ReadResponse{
		Content:     content,
		MimeType:    meta.MimeType,
		Filename:    meta.Filename,
		VirtualPath: meta.VirtualPath,
	}, nil
}

// Delete removes an artifact permanently.
func (s *Server) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID := resolveUserID(ctx, req.UserId)
	slog.Info("gRPC Delete request", "id", req.Id, "user_id", userID)
	deleted, err := s.Store.Delete(req.Id, userID)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserID) || errors.Is(err, storage.ErrPathEscape) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
		}
		slog.Error("failed to delete artifact", "error", err, "id", req.Id, "user_id", userID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to delete artifact"))
	}
	if !deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("artifact not found"))
	}
	return &pb.DeleteResponse{Deleted: deleted}, nil
}

// List returns a paginated list of artifacts or a virtual directory listing.
func (s *Server) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	userID := resolveUserID(ctx, req.UserId)
	slog.Info("gRPC List request", "user_id", userID, "vdir", req.DirPath, "source", req.Source)
	var items []*storage.ArtifactMetadata
	var err error
	if req.Source == "" {
		items, err = s.Store.List(userID, int(req.Limit), int(req.Offset), req.DirPath)
	} else {
		// The store knows nothing about sources, so filter here and paginate afterwards.
		items, err = s.Store.List(userID, 0, 0, req.DirPath)
		items = paginate(filterSource(items, req.Source), int(req.Limit), int(req.Offset))
	}
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserID) || errors.Is(err, storage.ErrPathEscape) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
		}
		slog.Error("failed to list artifacts", "error", err, "user_id", userID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to list artifacts"))
	}

	var pbItems []*pb.ArtifactInfo
	for _, item := range items {
		pbItems = append(pbItems, &pb.ArtifactInfo{
			Id:          item.ID,
			Filename:    item.Filename,
			MimeType:    item.MimeType,
			Source:      item.Source,
			UserId:      item.UserID,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			ExpiresAt:   item.ExpiresAt.Format(time.RFC3339),
			SizeBytes:   item.SizeBytes,
			VirtualPath: item.VirtualPath,
			IsDirectory: item.MimeType == "directory",
		})
	}

	return &pb.ListResponse{Items: pbItems}, nil
}

// Patch updates part of an artifact's content.
func (s *Server) Patch(ctx context.Context, req *pb.PatchRequest) (*pb.PatchResponse, error) {
	userID := resolveUserID(ctx, req.UserId)
	slog.Info("gRPC Patch request", "id", req.Id, "user_id", userID)
	newSize, err := s.Store.Patch(req.Id, userID, req.Content, int(req.LineStart), int(req.LineEnd), req.Append)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserID) || errors.Is(err, storage.ErrPathEscape) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
		}
		if errors.Is(err, storage.ErrArtifactNotFound) || err.Error() == "artifact not found" {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("artifact not found"))
		}
		if errors.Is(err, storage.ErrBinaryFile) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot patch binary file: patch is only supported for text files"))
		}
		slog.Error("failed to patch artifact", "error", err, "id", req.Id, "user_id", userID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to patch artifact"))
	}

	return &pb.PatchResponse{
		Success:   true,
		NewSize:   newSize,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// Find searches for artifacts by virtual path pattern.
func (s *Server) Find(ctx context.Context, req *pb.FindRequest) (*pb.ListResponse, error) {
	userID := resolveUserID(ctx, req.UserId)
	slog.Info("gRPC Find request", "pattern", req.Pattern, "user_id", userID)
	items, err := s.Store.Find(userID, req.Pattern)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidUserID) || errors.Is(err, storage.ErrPathEscape) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
		}
		slog.Error("failed to find artifacts", "error", err, "user_id", userID)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to find artifacts"))
	}

	var pbItems []*pb.ArtifactInfo
	for _, item := range items {
		pbItems = append(pbItems, &pb.ArtifactInfo{
			Id:          item.ID,
			Filename:    item.Filename,
			MimeType:    item.MimeType,
			Source:      item.Source,
			UserId:      item.UserID,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			ExpiresAt:   item.ExpiresAt.Format(time.RFC3339),
			SizeBytes:   item.SizeBytes,
			VirtualPath: item.VirtualPath,
		})
	}

	return &pb.ListResponse{Items: pbItems}, nil
}

// filterSource keeps the artifacts written by source. Directory entries have
// no source and are always kept, so a VFS listing stays navigable.
func filterSource(items []*storage.ArtifactMetadata, source string) []*storage.ArtifactMetadata {
	var out []*storage.ArtifactMetadata
	for _, it := range items {
		if it.Source == source || it.MimeType == "directory" {
			out = append(out, it)
		}
	}
	return out
}

func paginate(items []*storage.ArtifactMetadata, limit, offset int) []*storage.ArtifactMetadata {
	if offset >= len(items) {
		return nil
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}
