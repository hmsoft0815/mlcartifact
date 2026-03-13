# Go Client Library Guide

The `mlcartifact` Go library provides a high-level, type-safe API for interacting with the artifact storage service. It simplifies connection management, authentication (via environment), and common operations.

## Installation

```bash
go get github.com/hmsoft0815/mlcartifact
```

## Quick Start

The simplest way to use the library is to rely on environment variables for configuration.

```go
import (
    "context"
    "fmt"
    "log"

    "github.com/hmsoft0815/mlcartifact"
)

func main() {
    ctx := context.Background()

    // 1. Initialize client (reads ARTIFACT_GRPC_ADDR)
    // Automatically supports HTTP/2 (H2C) and falls back to HTTP/1.1
    client, err := mlcartifact.NewClient()
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // 2. Write an artifact
    res, err := client.Write(ctx, "hello.txt", []byte("Hello World!"),
        mlcartifact.WithDescription("A simple test file"),
        mlcartifact.WithExpiresHours(1),
    )
    if err != nil {
        log.Fatalf("Write failed: %v", err)
    }

    fmt.Printf("Created artifact ID: %s\n", res.Id)

    // 3. Read it back
    resRead, err := client.Read(ctx, res.Id)
    if err != nil {
        log.Fatalf("Read failed: %v", err)
    }

    fmt.Printf("Content: %s\n", string(resRead.Content))
}
```

## Configuration

The client automatically respects the following environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `ARTIFACT_GRPC_ADDR` | `:9590` | The address of the gRPC server. |
| `ARTIFACT_SOURCE` | `""` | Default source tag for all `Write` operations. |
| `ARTIFACT_USER_ID` | `""` | Default user ID scoping for all operations. |

### Manual Connection

If you need to connect to a specific address or provide a custom `http.Client`:

```go
client, err := mlcartifact.NewClientWithAddr("remote-host:9590")
```

### Firewall-Friendly Communication (Connect-Go)

The library uses the **Connect** protocol instead of raw gRPC. This provides several advantages for library users:

1. **Proxy Compatibility**: It works seamlessly through HTTP/1.1 proxies and load balancers that do not support HTTP/2.
2. **Firewall Friendly**: It uses standard HTTP POST requests, which are less likely to be blocked than raw gRPC streams.
3. **No TLS Required for Local H2C**: It supports HTTP/2 over cleartext (H2C) for high performance without the complexity of local certificate management.
4. **Browser Support**: The underlying protocol is compatible with `gRPC-Web`, making it easier to integrate with web frontends.

## Advanced Usage

### Working with Contexts & Timeouts

Always use bounded contexts for network operations to prevent hanging.

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

res, err := client.Read(ctx, "my-large-file.zip")
```

### Scoping to Users

By default, artifacts are saved in a "global" space unless a `UserID` is provided. You can set this globally via environment variables or per-request using functional options.

```go
// This artifact will only be visible when reading with the same user ID
client.Write(ctx, "private.txt", data, mlcartifact.WithUserID("user_123"))

// Reading requires the same ID
client.Read(ctx, "private.txt", mlcartifact.WithReadUserID("user_123"))
```

### Error Handling

The client returns standard gRPC errors. Use the `google.golang.org/grpc/status` package to check for specific error codes like `NotFound`.

```go
import "google.golang.org/grpc/status"
import "google.golang.org/grpc/codes"

res, err := client.Read(ctx, "non-existent")
if err != nil {
    if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
        fmt.Println("Artifact not found")
    } else {
        log.Fatal(err)
    }
}
```

## Testing & Mocking

The library is designed for testability. You can use `NewClientWithService` to wrap a mock implementation of the `ArtifactServiceClient` interface.

```go
// In your test file
type mockService struct {
    pb.UnimplementedArtifactServiceServer
    // ... add fields to track calls
}

func TestMyTool(t *testing.T) {
    mock := &mockService{}
    client := mlcartifact.NewClientWithService(mock)
    
    // Pass this client to your tool...
}
```

## Versioning

The library version is available as a constant:

```go
fmt.Println("mlcartifact version:", mlcartifact.Version)
```

## Thread Safety

The `mlcartifact.Client` is **thread-safe**. You should typically create one instance and share it across your entire application/server.
