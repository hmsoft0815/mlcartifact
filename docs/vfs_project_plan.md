# Project Plan: Virtual File System (VFS) Layer for mlcartifact

## Objective
Extend the `mlcartifact` storage to support a hierarchical **Virtual File System (VFS)**. This allows LLMs and tools to organize artifacts into "directories" (e.g., `/docs/reports/q1.md`) while keeping the underlying storage flat and ID-based.

## Core Features
1.  **Virtual Path Support**: Map arbitrary hierarchical paths to unique artifact IDs within a user's scope.
2.  **Hierarchical Navigation**: Support directory listing (`ls`) and recursive finding (`find`).
3.  **Partial Content Manipulation**: Add support for line-wise replacement and patching to avoid re-uploading large files for minor changes.
4.  **Implicit Directory Creation**: Like S3 or Git, directories exist if they contain files.
5.  **User Isolation**: Paths are scoped to a `UserID`. One user's `/docs` is separate from another's.

## Architecture

### 1. Metadata Extension
Update `ArtifactMetadata` to include:
- `VirtualPath`: The full path (e.g., `/projects/alpha/readme.md`).
- `FileName`: The base name of the file.
- `DirName`: The directory containing the file.

### 2. Storage Logic (Internal)
- Internally, files continue to be stored as `{id}_{filename}` to prevent collisions.
- The `Store.List` and `Store.Read` methods will be extended to support lookup by `VirtualPath`.

### 3. New MCP Tools (LLM Interface)
The following tools will be added to the server's MCP capabilities:
- `vfs_write`: Save content to a specific path (e.g., `/code/main.py`).
- `vfs_read`: Read content from a path.
- `vfs_ls`: List files and "subdirectories" in a path.
- `vfs_find`: Search for paths matching a pattern (e.g., `*.log`).
- `vfs_patch`: Replace specific lines or append to a file at a path.
- `vfs_delete`: Remove a file by path.

## Roadmap

### Phase 1: Storage Layer Enhancements (Core)
- [x] Update `ArtifactMetadata` struct.
- [x] Implement `Store.ReadByPath(userID, path)` and `Store.WriteByPath(userID, path, ...)`. (Implemented in general `Read`/`Write`)
- [x] Implement `Store.ListByPath(userID, dirPath)` (handles virtual directory logic).
- [x] Add `Store.Find(userID, pattern)`.

### Phase 2: Partial Edits (VFS Patch)
- [x] Implement `Store.Patch(userID, path, lineStart, lineEnd, newContent)`.
- [x] Implement `Store.Append(userID, path, content)`.

### Phase 3: API & MCP Integration
- [x] Add gRPC methods for VFS operations to `artifact.proto`.
- [x] Implement server-side handlers for new gRPC methods.
- [x] Register new MCP tools (`vfs_ls`, `vfs_write`, etc.).

### Phase 4: Validation & Tooling
- [x] Unit tests for VFS path resolution.
- [/] Update `artifact-cli` to support VFS commands (e.g., `artifact-cli vfs-ls /`).
- [/] Documentation update for VFS usage. (Prompt + mlcprodweb)

## Technical Design Considerations
- **Normalization**: Paths should always start with `/` and be sanitized (remove `..`, trailing slashes, etc.).
- **Atomic Operations**: Ensure metadata and file updates are as consistent as possible.
- **In-Memory Cache**: Implement an in-memory index (`UserID` -> `Path` -> `Metadata`) for $O(1)$ path resolution and fast `ls` operations. Rebuild index on server startup.
- **LLM Context Efficiency**: `vfs_ls` should return concise summaries to avoid bloating the LLM context.
