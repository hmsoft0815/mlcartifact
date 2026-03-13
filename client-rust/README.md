# mlcartifact Rust SDK

A high-performance Rust client for the mlcartifact service using [Tonic](https://github.com/hyperium/tonic).

## Installation

Add this to your Cargo.toml:

```toml
[dependencies]
mlcartifact = { git = "https://github.com/hmsoft0815/mlcartifact.git", subdirectory = "client-rust" }
tokio = { version = "1.0", features = ["full"] }
tonic = "0.12"
```

## Usage

```rust
use mlcartifact::gen::artifact_service_client::ArtifactServiceClient;
use mlcartifact::gen::WriteRequest;
use tonic::Request;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = ArtifactServiceClient::connect("http://localhost:9590").await?;

    let request = Request::new(WriteRequest {
        filename: "hello.txt".into(),
        content: b"Hello from Rust!".to_vec(),
        ..Default::default()
    });

    let response = client.write(request).await?;
    println!("Artifact ID: {}", response.into_inner().id);

    Ok(())
}
```

## Running the Example

```bash
cargo run --example basic
```

## License

MIT - Copyright (c) 2026 Michael Lechner
