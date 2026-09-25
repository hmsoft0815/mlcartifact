// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package mcp provides the Model Context Protocol (MCP) tool implementations
// for the artifact service. These handlers allow LLMs to interact with the
// artifact store via the official MCP Go SDK.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var store = storage.NewStore(".artifacts")

// SetStore updates the global store instance used by all MCP handlers.
func SetStore(s *storage.Store) {
	store = s
}

// MCPListLimit defines the default maximum number of artifacts returned via MCP.
var MCPListLimit = 100

// SetMCPListLimit updates the default limit for listing artifacts.
func SetMCPListLimit(limit int) {
	MCPListLimit = limit
}

// Instructions is sent to the client on initialize and tells the model how the
// store is meant to be used.
const Instructions = `mlcartifact is a shared artifact store. Save large results with write_artifact and pass only the returned id (or virtual_path) on to the next tool instead of copying the content into the conversation. Read content back with read_artifact only when you really need to see it. Use virtual paths (starting with /) to organise files; see the vfs_usage prompt for details.`

// Tool annotations: the store is local, so no tool reaches an open world.
var (
	boolFalse = false
	boolTrue  = true
)

func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &boolFalse}
}

// Register adds all artifact tools and the vfs_usage prompt to the server.
func Register(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "write_artifact",
		Title: "Write artifact",
		Description: `Saves a file to the shared artifact store and returns its id, virtual_path and a <file> reference tag.
Supports an optional 'virtual_path' (e.g. '/projects/test/data.csv') to organise files hierarchically.
Pass the id on to other tools instead of copying the content.`,
		Annotations: &mcp.ToolAnnotations{Title: "Write artifact", DestructiveHint: &boolFalse, OpenWorldHint: &boolFalse},
		Icons: []mcp.Icon{{
			Source:   "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9ImN1cnJlbnRDb2xvciIgc3Ryb2tlLXdpZHRoPSIyIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1saW5lam9pbj0icm91bmQiPjxwYXRoIGQ9Ik0xNCAydkg2YTIgMiAwIDAgMC0yIDJ2MTZhMiAyIDAgMCAwIDIgMmgxMmEyIDIgMCAwIDAgMi0yVjhsLTYtNnoiLz48cG9seWxpbmUgcG9pbnRzPSIxNCAyIDE0IDggMjAgOCIvPjwvc3ZnPg==",
			MIMEType: "image/svg+xml",
		}},
	}, WriteArtifact)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "read_artifact",
		Title:       "Read artifact",
		Description: "Returns the content of a saved artifact as text. Supports lookup by unique ID or virtual path (if it starts with /).",
		Annotations: readOnly("Read artifact"),
	}, ReadArtifact)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "vfs_ls",
		Title:       "List directory",
		Description: "Lists files and virtual directories within a specific path.",
		Annotations: readOnly("List directory"),
	}, VFSList)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "vfs_patch",
		Title:       "Patch artifact",
		Description: "Replaces specific lines of an artifact or appends to it. Efficient for large files.",
		Annotations: &mcp.ToolAnnotations{Title: "Patch artifact", DestructiveHint: &boolTrue, OpenWorldHint: &boolFalse},
	}, VFSPatch)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "vfs_find",
		Title:       "Find artifacts",
		Description: "Searches for artifacts using glob patterns or keywords in their virtual paths.",
		Annotations: readOnly("Find artifacts"),
	}, VFSFind)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_artifacts",
		Title:       "List artifacts",
		Description: "Lists ALL saved artifacts for a user (flat list). Use vfs_ls for directory browsing.",
		Annotations: readOnly("List artifacts"),
	}, ListArtifacts)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_artifact",
		Title:       "Delete artifact",
		Description: "Permanently deletes an artifact. Supports ID or virtual path.",
		Annotations: &mcp.ToolAnnotations{Title: "Delete artifact", DestructiveHint: &boolTrue, IdempotentHint: true, OpenWorldHint: &boolFalse},
	}, DeleteArtifact)

	s.AddPrompt(&mcp.Prompt{
		Name:        "vfs_usage",
		Title:       "VFS usage guidelines",
		Description: "Guidelines on how to use the Virtual File System (VFS) for hierarchical artifact storage and surgical patching.",
	}, HandleVFSUsagePrompt)
}

// WriteArtifactArgs defines the input for saving an artifact via MCP.
type WriteArtifactArgs struct {
	Filename       string         `json:"filename" jsonschema:"The desired name for the file (e.g. 'report.md', 'data.csv', 'script.py')."`
	Content        string         `json:"content" jsonschema:"The full text content to be saved."`
	VirtualPath    string         `json:"virtual_path,omitempty" jsonschema:"Optional hierarchical path (e.g. '/docs/manual.md'). Must start with /."`
	MimeType       string         `json:"mime_type,omitempty" jsonschema:"Optional MIME type. Detected from the filename if omitted."`
	Description    string         `json:"description,omitempty" jsonschema:"Optional human-readable description."`
	ExpiresInHours int            `json:"expires_in_hours,omitempty" jsonschema:"Optional: hours after which the file is deleted (default 24)."`
	Metadata       map[string]any `json:"metadata,omitempty" jsonschema:"Optional arbitrary key-value pairs."`
	UserID         string         `json:"user_id,omitempty" jsonschema:"Optional user ID that scopes the artifact."`
}

// WriteArtifactResult is the structured result of write_artifact.
type WriteArtifactResult struct {
	ID          string `json:"id" jsonschema:"Unique artifact ID"`
	Filename    string `json:"filename"`
	VirtualPath string `json:"virtual_path,omitempty"`
	MimeType    string `json:"mime_type"`
	ExpiresAt   string `json:"expires_at" jsonschema:"Scheduled deletion time (RFC 3339)"`
	Reference   string `json:"reference" jsonschema:"<file> tag to quote in the answer to the user"`
}

// WriteArtifact saves a file to the artifact store.
func WriteArtifact(ctx context.Context, req *mcp.CallToolRequest, args WriteArtifactArgs) (*mcp.CallToolResult, WriteArtifactResult, error) {
	store.Cleanup()

	if args.Filename == "" || args.Content == "" {
		return nil, WriteArtifactResult{}, errors.New("filename and content must not be empty")
	}

	meta, err := store.Write(
		args.Filename,
		[]byte(args.Content),
		args.MimeType,
		args.ExpiresInHours,
		"mcp-tool", // source
		args.UserID,
		args.Description,
		args.Metadata,
		args.VirtualPath,
	)
	if err != nil {
		return nil, WriteArtifactResult{}, fmt.Errorf("storage error: %w", err)
	}

	slog.Info("artifact saved via MCP", "id", meta.ID, "filename", meta.Filename, "vpath", meta.VirtualPath)

	out := WriteArtifactResult{
		ID:          meta.ID,
		Filename:    meta.Filename,
		VirtualPath: meta.VirtualPath,
		MimeType:    meta.MimeType,
		ExpiresAt:   meta.ExpiresAt.Format(time.RFC3339),
		Reference:   fmt.Sprintf("<file id=\"%s\" type=\"%s\">%s</file>", meta.ID, meta.MimeType, meta.Filename),
	}
	return textResult(out), out, nil
}

// textResult renders v as indented JSON without HTML escaping. The SDK's own
// fallback escapes '<' to \u003c, which garbles the <file> reference tag the
// model is supposed to copy into its answer.
func textResult(v any) *mcp.CallToolResult {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: strings.TrimSpace(buf.String())}}}
}

// ReadArtifactArgs defines the input for reading an artifact via MCP.
type ReadArtifactArgs struct {
	ID     string `json:"id" jsonschema:"The unique artifact ID or virtual path."`
	UserID string `json:"user_id,omitempty" jsonschema:"Optional user ID that scopes the lookup."`
}

// ReadArtifact returns an artifact's content as plain text. It has no output
// schema on purpose: a structured copy would send the content twice.
func ReadArtifact(ctx context.Context, req *mcp.CallToolRequest, args ReadArtifactArgs) (*mcp.CallToolResult, any, error) {
	if args.ID == "" {
		return nil, nil, errors.New("id must not be empty")
	}

	content, meta, err := store.Read(args.ID, args.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading artifact %q: %w", args.ID, err)
	}

	slog.Info("artifact read via MCP", "id", meta.ID, "filename", meta.Filename)

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(content)}}}, nil, nil
}

// ArtifactList is the structured result of all listing tools.
type ArtifactList struct {
	Count     int                         `json:"count" jsonschema:"Number of entries returned"`
	Artifacts []*storage.ArtifactMetadata `json:"artifacts"`
}

func newList(items []*storage.ArtifactMetadata) ArtifactList {
	if items == nil {
		items = []*storage.ArtifactMetadata{}
	}
	return ArtifactList{Count: len(items), Artifacts: items}
}

// ListArtifactsArgs defines the input for listing artifacts via MCP.
type ListArtifactsArgs struct {
	UserID string `json:"user_id,omitempty" jsonschema:"Optional user ID that scopes the listing."`
}

// ListArtifacts returns a flat list of available artifacts.
func ListArtifacts(ctx context.Context, req *mcp.CallToolRequest, args ListArtifactsArgs) (*mcp.CallToolResult, ArtifactList, error) {
	// We limit MCP results as LLMs don't need huge lists.
	items, err := store.List(args.UserID, MCPListLimit, 0, "")
	if err != nil {
		return nil, ArtifactList{}, fmt.Errorf("error listing artifacts: %w", err)
	}
	return nil, newList(items), nil
}

// DeleteArtifactArgs defines the input for deleting an artifact via MCP.
type DeleteArtifactArgs struct {
	ID     string `json:"id" jsonschema:"The unique artifact ID or virtual path."`
	UserID string `json:"user_id,omitempty" jsonschema:"Optional user ID that scopes the deletion."`
}

// DeleteArtifactResult is the structured result of delete_artifact.
type DeleteArtifactResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// DeleteArtifact removes an artifact from the store.
func DeleteArtifact(ctx context.Context, req *mcp.CallToolRequest, args DeleteArtifactArgs) (*mcp.CallToolResult, DeleteArtifactResult, error) {
	if args.ID == "" {
		return nil, DeleteArtifactResult{}, errors.New("id must not be empty")
	}

	deleted, err := store.Delete(args.ID, args.UserID)
	if err != nil {
		return nil, DeleteArtifactResult{}, fmt.Errorf("error deleting artifact %q: %w", args.ID, err)
	}
	if !deleted {
		return nil, DeleteArtifactResult{}, fmt.Errorf("artifact %q not found", args.ID)
	}

	slog.Info("artifact deleted via MCP", "id", args.ID)
	return nil, DeleteArtifactResult{ID: args.ID, Deleted: true}, nil
}

// VFSPatchArgs defines the input for patching an artifact via MCP.
type VFSPatchArgs struct {
	ID        string `json:"id" jsonschema:"The artifact ID or virtual path."`
	Content   string `json:"content" jsonschema:"The new content to insert or append."`
	LineStart int    `json:"line_start,omitempty" jsonschema:"Optional: first line to replace (0-based)."`
	LineEnd   int    `json:"line_end,omitempty" jsonschema:"Optional: line after the last one to replace (0-based, exclusive). line_start 2, line_end 4 replaces lines 2 and 3; line_end equal to line_start inserts without replacing."`
	Append    bool   `json:"append,omitempty" jsonschema:"If true, appends content to the end of the file (no newline is added)."`
	UserID    string `json:"user_id,omitempty" jsonschema:"Optional user ID scope."`
}

// VFSPatchResult is the structured result of vfs_patch.
type VFSPatchResult struct {
	Success bool  `json:"success"`
	NewSize int64 `json:"new_size" jsonschema:"Size of the artifact in bytes after the patch"`
}

// VFSPatch modifies an artifact's content.
func VFSPatch(ctx context.Context, req *mcp.CallToolRequest, args VFSPatchArgs) (*mcp.CallToolResult, VFSPatchResult, error) {
	newSize, err := store.Patch(args.ID, args.UserID, []byte(args.Content), args.LineStart, args.LineEnd, args.Append)
	if err != nil {
		return nil, VFSPatchResult{}, fmt.Errorf("error patching artifact %q: %w", args.ID, err)
	}
	return nil, VFSPatchResult{Success: true, NewSize: newSize}, nil
}

// VFSListArgs defines the input for listing a virtual directory.
type VFSListArgs struct {
	Path   string `json:"path" jsonschema:"The virtual directory path to list (e.g. '/')."`
	UserID string `json:"user_id,omitempty" jsonschema:"Optional user ID scope."`
}

// VFSList lists a virtual directory.
func VFSList(ctx context.Context, req *mcp.CallToolRequest, args VFSListArgs) (*mcp.CallToolResult, ArtifactList, error) {
	items, err := store.List(args.UserID, MCPListLimit, 0, args.Path)
	if err != nil {
		return nil, ArtifactList{}, fmt.Errorf("error listing %q: %w", args.Path, err)
	}
	return nil, newList(items), nil
}

// VFSFindArgs defines the input for searching artifacts.
type VFSFindArgs struct {
	Pattern string `json:"pattern" jsonschema:"Search pattern (e.g. '*.log' or 'reports')."`
	UserID  string `json:"user_id,omitempty" jsonschema:"Optional user ID scope."`
}

// VFSFind searches for artifacts.
func VFSFind(ctx context.Context, req *mcp.CallToolRequest, args VFSFindArgs) (*mcp.CallToolResult, ArtifactList, error) {
	items, err := store.Find(args.UserID, args.Pattern)
	if err != nil {
		return nil, ArtifactList{}, fmt.Errorf("error finding artifacts: %w", err)
	}
	return nil, newList(items), nil
}

// HandleVFSUsagePrompt provides guidelines to the LLM on using the virtual file system.
func HandleVFSUsagePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	instructions := `# mlcartifact VFS Usage Guidelines

You are interacting with a persistent artifact storage service that supports a hierarchical **Virtual File System (VFS)**.

## 1. Organizing with Virtual Paths
Instead of relying on flat IDs, you can organize your work into logical directory structures using the ` + "`virtual_path`" + ` argument in ` + "`write_artifact`" + `.
- **Always** use absolute paths starting with ` + "`/`" + ` (e.g., ` + "`/projects/ai-analysis/report.md`" + `).
- Directories are created implicitly.

## 2. Surgical Edits with vfs_patch
For large artifacts (code files, data logs, long reports), prefer ` + "`vfs_patch`" + ` over ` + "`write_artifact`" + `.
- **Appending**: Use ` + "`append: true`" + ` to quickly add data to the end of a file.
- **Line Replacement**: Use ` + "`line_start`" + ` and ` + "`line_end`" + ` (0-indexed, end exclusive) to replace specific sections: ` + "`line_start: 2, line_end: 4`" + ` replaces lines 2 and 3.
- This is much faster and more token-efficient than re-uploading the entire file.

## 3. Discovery & Navigation
- Use ` + "`vfs_ls`" + ` to list contents of a virtual directory.
- Use ` + "`vfs_find`" + ` with glob patterns (e.g., ` + "`/**/*.go`" + `) to search across the entire user scope.
- When reading an artifact by path, ensure you include the leading ` + "`/`" + `.

## 4. Cross-Tool References
When you save an artifact, you receive a reference tag like ` + "`<file id=\"...\" type=\"...\">filename</file>`" + `.
- **Always** include this tag in your final response to the user so they can access the file.
- Other tools (like D2 renderer or Barcode generator) can also output these tags.
`

	return &mcp.GetPromptResult{
		Description: "Guidelines for using the mlcartifact VFS capabilities.",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "assistant",
				Content: &mcp.TextContent{Text: instructions},
			},
		},
	}, nil
}
