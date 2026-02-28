// Copyright (c) 2026 Michael Lechner. All rights reserved.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/hmsoft0815/mlcartifact/cmd/server/internal/storage"
)

// WriteArtifactArgs defines the input for saving an artifact.
type WriteArtifactArgs struct {
	Filename       string                 `json:"filename"`
	Content        string                 `json:"content"`
	Description    string                 `json:"description,omitempty"`
	MimeType       string                 `json:"mime_type,omitempty"`
	ExpiresInHours int                    `json:"expires_in_hours,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
}

var store = storage.NewStore(".artifacts")

// SetStore updates the store used by handlers.
func SetStore(s *storage.Store) {
	store = s
}

const errInvalidArgs = "invalid arguments: "

// MCPListLimit defines the maximum number of artifacts returned via MCP.
var MCPListLimit = 100

// SetMCPListLimit updates the default limit for listing artifacts.
func SetMCPListLimit(limit int) {
	MCPListLimit = limit
}

// WriteArtifact saves a file to the artifacts directory and returns a reference.
func WriteArtifact(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 1. Run Cleanup first to keep house clean
	store.Cleanup()

	// 2. Parse arguments
	var args WriteArtifactArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	if args.Filename == "" || args.Content == "" {
		return mcp.NewToolResultText("filename and content are required"), nil
	}

	// 3. Write via shared store
	meta, err := store.Write(
		args.Filename,
		[]byte(args.Content),
		args.MimeType,
		args.ExpiresInHours,
		"mcp-tool", // source
		args.UserID,
		args.Description,
		args.Metadata,
	)

	if err != nil {
		return nil, fmt.Errorf("storage error: %w", err)
	}

	// 4. Build response
	fileTag := fmt.Sprintf("<file id=\"%s\" type=\"%s\">%s</file>", meta.ID, meta.MimeType, meta.Filename)
	res := map[string]interface{}{
		"id":         meta.ID,
		"filename":   meta.Filename,
		"mime_type":  meta.MimeType,
		"expires_at": meta.ExpiresAt.Format(time.RFC3339),
		"reference":  fileTag,
	}
	resBytes, _ := json.MarshalIndent(res, "", "  ")

	slog.Info("artifact saved via MCP", "id", meta.ID, "filename", meta.Filename)

	return mcp.NewToolResultText(string(resBytes)), nil
}

// ReadArtifactArgs defines the input for reading an artifact.
type ReadArtifactArgs struct {
	ID     string `json:"id"`
	UserID string `json:"user_id,omitempty"`
}

// ReadArtifact retrieves an artifact's content.
func ReadArtifact(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args ReadArtifactArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	if args.ID == "" {
		return mcp.NewToolResultText("id is required"), nil
	}

	content, meta, err := store.Read(args.ID, args.UserID)
	if err != nil {
		return mcp.NewToolResultText("error reading artifact: " + err.Error()), nil
	}

	slog.Info("artifact read via MCP", "id", meta.ID, "filename", meta.Filename)

	// We return the content directly as text if possible, or as a message
	return mcp.NewToolResultText(string(content)), nil
}

// ListArtifactsArgs defines the input for listing artifacts.
type ListArtifactsArgs struct {
	UserID string `json:"user_id,omitempty"`
}

// ListArtifacts returns all artifacts.
func ListArtifacts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args ListArtifactsArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	json.Unmarshal(argBytes, &args)

	// We limit MCP results as LLMs don't need huge lists.
	items, err := store.List(args.UserID, MCPListLimit, 0)
	if err != nil {
		return mcp.NewToolResultText("error listing artifacts: " + err.Error()), nil
	}

	resBytes, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(resBytes)), nil
}

// DeleteArtifactArgs defines the input for deleting an artifact.
type DeleteArtifactArgs struct {
	ID     string `json:"id"`
	UserID string `json:"user_id,omitempty"`
}

// DeleteArtifact removes an artifact.
func DeleteArtifact(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args DeleteArtifactArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	if args.ID == "" {
		return mcp.NewToolResultText("id is required"), nil
	}

	deleted, err := store.Delete(args.ID, args.UserID)
	if err != nil {
		return mcp.NewToolResultText("error deleting artifact: " + err.Error()), nil
	}

	if !deleted {
		return mcp.NewToolResultText("artifact not found"), nil
	}

	slog.Info("artifact deleted via MCP", "id", args.ID)
	return mcp.NewToolResultText("artifact deleted successfully"), nil
}
