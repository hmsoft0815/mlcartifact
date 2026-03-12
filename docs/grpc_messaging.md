# gRPC API Reference

This document describes the low-level gRPC communication layer of `mlcartifact`. 

> [!TIP]
> **Go Developers:** If you are using our Go library, please refer to the **[Go Client Library Guide](go_library.md)** for a more high-level and user-friendly documentation.

## Overview

The `mlcartifact` server provides a dual-interface:
1. **MCP (Model Context Protocol):** For LLMs and agents to interact with artifacts via `stdio` or `SSE`.
2. **gRPC:** For high-performance, internal service-to-service communication.

The Go library uses the gRPC interface to provide a type-safe and efficient way for other MCP servers or backend services to store and retrieve data.

---

## gRPC Service Definition

The service is defined in `proto/artifact.proto`.

### Service: `ArtifactService`

| Method | Request | Response | Description |
| :--- | :--- | :--- | :--- |
| `Write` | `WriteRequest` | `WriteResponse` | Persists a file/buffer to the store. |
| `Read` | `ReadRequest` | `ReadResponse` | Retrieves content and metadata by ID or filename. |
| `List` | `ListRequest` | `ListResponse` | Lists available artifacts, optionally filtered by user. |
| `Delete` | `DeleteRequest` | `DeleteResponse` | Removes an artifact from the store. |

### Important Messages

#### `WriteRequest`
- `filename` (string): The desired name (e.g., "report.pdf").
- `content` (bytes): Raw binary data. **Do not base64 encode.**
- `mime_type` (string, optional): Auto-detected if omitted.
- `expires_hours` (int32, optional): Default is 24 hours.
- `source` (string, optional): Identifier for the caller (e.g., "sql-mcp-server").
- `user_id` (string, optional): Scopes the artifact to a specific user.

---

## Go Client Library Usage

The Go library simplifies gRPC connection management and provides a clean API with functional options.

### Installation

```bash
go get github.com/hmsoft0815/mlcartifact
```

### Basic Setup

```go
import (
    "context"
    "log"
    "github.com/hmsoft0815/mlcartifact"
)

func main() {
    // NewClient reads ARTIFACT_GRPC_ADDR from env (default :9590)
    client, err := mlcartifact.NewClient()
    if err != nil {
        log.Fatalf("failed to create client: %v", err)
    }
    defer client.Close()

    ctx := context.Background()
    // ... use client
}
```

### Writing an Artifact

```go
resp, err := client.Write(ctx, "data.csv", []byte("id,name\n1,Alice"),
    mlcartifact.WithMimeType("text/csv"),
    mlcartifact.WithExpiresHours(48),
    mlcartifact.WithDescription("User export data"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Saved artifact: %s (URI: %s)\n", resp.Id, resp.Uri)
```

### Reading an Artifact

```go
// You can use the ID or the sanitized filename
resp, err := client.Read(ctx, "data.csv")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Content size: %d bytes, MIME: %s\n", len(resp.Content), resp.MimeType)
```

### Listing Artifacts

```go
list, err := client.List(ctx, "user-123", 
    mlcartifact.WithLimit(10),
)
if err != nil {
    log.Fatal(err)
}

for _, item := range list.Items {
    fmt.Printf("- %s (%s) created at %s\n", item.Filename, item.Id, item.CreatedAt)
}
```

---

## Configuration (Environment Variables)

The client library automatically respects the following environment variables if they are set:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `ARTIFACT_GRPC_ADDR` | `:9590` | The address of the `mlcartifact` gRPC server. |
| `ARTIFACT_SOURCE` | `""` | Default source tag for all `Write` operations. |
| `ARTIFACT_USER_ID` | `""` | Default user ID scoping for all operations. |

### Manual Override in Code

You can also specify these values directly when creating the client or per-request:

```go
// Direct address
client, _ := mlcartifact.NewClientWithAddr("localhost:9000")

// Per-request user override
client.Write(ctx, "file.txt", data, mlcartifact.WithUserID("special-user"))
```

---

## Error Handling

The library returns standard gRPC errors. You can use `google.golang.org/grpc/status` to inspect error codes (e.g., `codes.NotFound` if an artifact doesn't exist).
