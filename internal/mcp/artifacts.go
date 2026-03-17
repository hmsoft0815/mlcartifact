// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package mcp provides the Model Context Protocol (MCP) tool implementations
// for the artifact service. These handlers allow LLMs to interact with the
// artifact store via the MCP framework.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
)

// WriteArtifactArgs defines the input for saving an artifact via MCP.
type WriteArtifactArgs struct {
	Filename       string                 `json:"filename"`         // Desired filename (e.g. "report.md")
	Content        string                 `json:"content"`          // Text content to store
	Description    string                 `json:"description,omitempty"` // Optional human-readable description
	MimeType       string                 `json:"mime_type,omitempty"`   // Optional MIME type (autodetected if empty)
	ExpiresInHours int                    `json:"expires_in_hours,omitempty"` // Hours until auto-deletion (default 24)
	Metadata       map[string]interface{} `json:"metadata,omitempty"`     // Arbitrary key-value pairs
	UserID         string                 `json:"user_id,omitempty"`      // Scopes the artifact to a specific user
	VirtualPath    string                 `json:"virtual_path,omitempty"` // Hierarchical path (VFS)
}

var store = storage.NewStore(".artifacts")

// SetStore updates the global store instance used by all MCP handlers.
func SetStore(s *storage.Store) {
	store = s
}

const errInvalidArgs = "invalid arguments: "

// MCPListLimit defines the default maximum number of artifacts returned via MCP.
var MCPListLimit = 100

// SetMCPListLimit updates the default limit for listing artifacts.
func SetMCPListLimit(limit int) {
	MCPListLimit = limit
}

// WriteArtifact is an MCP tool handler that saves a file to the artifacts store.
// It returns a JSON response containing the artifact ID and a reference tag.
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
		int(args.ExpiresInHours),
		"mcp-tool", // source
		args.UserID,
		args.Description,
		args.Metadata,
		args.VirtualPath,
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

	slog.Info("artifact saved via MCP", "id", meta.ID, "filename", meta.Filename, "vpath", meta.VirtualPath)

	return mcp.NewToolResultText(string(resBytes)), nil
}

// ReadArtifactArgs defines the input for reading an artifact via MCP.
type ReadArtifactArgs struct {
	ID     string `json:"id"`               // The ID or filename of the artifact
	UserID string `json:"user_id,omitempty"` // The user scope
}

// ReadArtifact is an MCP tool handler that retrieves an artifact's content.
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

// ListArtifactsArgs defines the input for listing artifacts via MCP.
type ListArtifactsArgs struct {
	UserID string `json:"user_id,omitempty"` // Optional user scope to filter results
}

	// ListArtifacts is an MCP tool handler that returns a list of available artifacts.
func ListArtifacts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args ListArtifactsArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	// We limit MCP results as LLMs don't need huge lists.
	items, err := store.List(args.UserID, int(MCPListLimit), 0, "")
	if err != nil {
		return mcp.NewToolResultText("error listing artifacts: " + err.Error()), nil
	}

	resBytes, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(resBytes)), nil
}

// DeleteArtifactArgs defines the input for deleting an artifact via MCP.
type DeleteArtifactArgs struct {
	ID     string `json:"id"`               // The ID or filename of the artifact to delete
	UserID string `json:"user_id,omitempty"` // The user scope
}

// DeleteArtifact is an MCP tool handler that removes an artifact from the store.
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

// VFSPatchArgs defines the input for patching an artifact via MCP.
type VFSPatchArgs struct {
	ID        string `json:"id"`                  // ID or virtual path
	Content   string `json:"content"`             // Text to insert/append
	LineStart int    `json:"line_start,omitempty"` // Optional start line
	LineEnd   int    `json:"line_end,omitempty"`   // Optional end line
	Append    bool   `json:"append,omitempty"`     // If true, appends to end
	UserID    string `json:"user_id,omitempty"`    // User scope
}

// VFSPatch is an MCP tool handler that modifies an artifact's content.
func VFSPatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args VFSPatchArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	newSize, err := store.Patch(args.ID, args.UserID, []byte(args.Content), args.LineStart, args.LineEnd, args.Append)
	if err != nil {
		return mcp.NewToolResultText("error patching artifact: " + err.Error()), nil
	}

	res := map[string]interface{}{
		"success":  true,
		"new_size": newSize,
	}
	resBytes, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(resBytes)), nil
}

// VFSListArgs defines the input for listing a virtual directory.
type VFSListArgs struct {
	Path   string `json:"path"`               // The virtual directory path (e.g. "/docs")
	UserID string `json:"user_id,omitempty"` // User scope
}

// VFSList is an MCP tool handler that lists a virtual directory.
func VFSList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args VFSListArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	items, err := store.List(args.UserID, int(MCPListLimit), 0, args.Path)
	if err != nil {
		return mcp.NewToolResultText("error listing vfs: " + err.Error()), nil
	}

	resBytes, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(resBytes)), nil
}

// VFSFindArgs defines the input for searching artifacts.
type VFSFindArgs struct {
	Pattern string `json:"pattern"`           // Glob pattern (e.g. "*.txt")
	UserID  string `json:"user_id,omitempty"` // User scope
}

// VFSFind is an MCP tool handler that searches for artifacts.
func VFSFind(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args VFSFindArgs
	argBytes, _ := json.Marshal(req.Params.Arguments)
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return mcp.NewToolResultText(errInvalidArgs + err.Error()), nil
	}

	items, err := store.Find(args.UserID, args.Pattern)
	if err != nil {
		return mcp.NewToolResultText("error finding artifacts: " + err.Error()), nil
	}

	resBytes, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(resBytes)), nil
}

