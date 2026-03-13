# mlcartifact Python Client

A light-weight, firewall-friendly Python client for the mlcartifact service using the Connect protocol.

## Features
- **Connect Protocol**: Works over standard HTTP/1.1 (no strict HTTP/2 requirement).
- **Simple API**: Easy-to-use methods for write, read, list, and delete.
- **Environment Aware**: Automatically respects ARTIFACT_GRPC_ADDR, ARTIFACT_SOURCE, and ARTIFACT_USER_ID.

## Installation

```bash
pip install .
```

Requires httpx and protobuf.

## Quick Start

```python
from mlcartifact import ArtifactClient

# Initialize client (it will read ARTIFACT_GRPC_ADDR)
with ArtifactClient() as client:
    # Write an artifact
    res = client.write(
        filename="hello.md",
        content=b"# Hello World",
        description="My first artifact"
    )
    print(f"Artifact saved: {res.id}")

    # Read an artifact
    read_res = client.read(res.id)
    print(f"Content: {read_res.content.decode('utf-8')}")
```

## Configuration

The client automatically respects the following environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| ARTIFACT_GRPC_ADDR | localhost:9590 | The address of the server. |
| ARTIFACT_SOURCE | "" | Default source tag. |
| ARTIFACT_USER_ID | "" | Default user ID scoping. |

## Versioning

The library version is available as:

```python
import mlcartifact
print(mlcartifact.__version__)
```

## License

MIT - Copyright (c) 2026 Michael Lechner
