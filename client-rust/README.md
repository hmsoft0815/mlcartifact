# mlcartifact Rust SDK

A Rust client for the mlcartifact service using [Tonic](https://github.com/hyperium/tonic) 0.14 / prost 0.14.

## Installation

The crate lives in the `client-rust/` subdirectory of the repository. Cargo has no
`subdirectory` key; a git dependency finds the crate by its package name
(`mlcartifact`) anywhere in the repo:

```toml
[dependencies]
mlcartifact = { git = "https://github.com/hmsoft0815/mlcartifact.git" }
tokio = { version = "1", features = ["full"] }
tonic = "0.14"   # only needed if you use tonic types (Status, Code) directly
```

Or, with a local checkout:

```toml
mlcartifact = { path = "../mlcartifact/client-rust" }
```

Building requires `protoc` (the proto is compiled from `../proto/artifact.proto`).

## Usage

```rust
use mlcartifact::{ArtifactClient, ListOptions, WriteOptions};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = ArtifactClient::connect("http://localhost:9590".into()).await?;

    // Write, optionally into the virtual file system
    let w = client
        .write_file("notes.txt", b"line0\nline1".to_vec(), WriteOptions {
            virtual_path: Some("/projects/alpha/notes.txt".into()),
            ..Default::default()
        })
        .await?;

    // Read by ID, filename or virtual path (starts with "/")
    let r = client.read("/projects/alpha/notes.txt".into(), None).await?;

    // Flat list (legacy signature) or VFS directory listing
    let all = client.list(None, Some(50), None).await?;
    let dir = client
        .list_with(ListOptions { dir_path: Some("/projects/alpha".into()), ..Default::default() })
        .await?;
    for item in dir.items {
        println!("{} dir={}", item.filename, item.is_directory);
    }

    // Patch: replace lines [1, 2) (0-based), or append
    client.patch(w.id.clone(), b"LINE1".to_vec(), Some(1), Some(2), false, None).await?;
    client.patch(w.id.clone(), b"\nmore".to_vec(), None, None, true, None).await?;

    // Find by glob / substring on the virtual path
    let hits = client.find("/projects/*/notes.txt", None).await?;

    client.delete(w.id, None).await?;
    Ok(())
}
```

### API overview (`ArtifactClient`)

| Method | Purpose |
|---|---|
| `connect(dst)` | Connect to `http://host:port` (gRPC over h2c) |
| `write(WriteRequest)` | Raw write |
| `write_file(filename, content, WriteOptions)` | Write with optional `virtual_path`, `mime_type`, `expires_hours`, `source`, `metadata`, `user_id`, `description` |
| `read(id_or_path, user_id)` | Read content and metadata |
| `list(user_id, limit, offset)` | Flat listing |
| `list_with(ListOptions)` | Listing with `source` filter and VFS `dir_path` mode |
| `patch(id_or_path, content, line_start, line_end, append, user_id)` | Line-range replace or append |
| `find(pattern, user_id)` | Search virtual paths |
| `delete(id_or_path, user_id)` | Delete an artifact |

The generated types and raw `ArtifactServiceClient` are available under `mlcartifact::gen`.

## Running the example and tests

```bash
ARTIFACT_GRPC_ADDR=localhost:9590 cargo run --example basic

# e2e test (skipped unless the variable is set)
MLCARTIFACT_E2E_ADDR=http://localhost:9590 cargo test
```

## License

MIT - Copyright (c) 2026 Michael Lechner
