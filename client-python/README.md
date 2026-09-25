# mlcartifact Python Client

A light-weight, firewall-friendly Python client for the mlcartifact service using the Connect protocol.

## Features
- **Connect Protocol**: binary protobuf over plain HTTP POST. Works over HTTP/1.1; HTTP/2 is used on `https://` when the server offers it. No `grpcio` needed.
- **Full API**: write, read, list, delete, plus the virtual file system (VFS) operations patch and find.
- **Environment Aware**: Automatically respects ARTIFACT_GRPC_ADDR, ARTIFACT_SOURCE, and ARTIFACT_USER_ID.

## Installation

```bash
pip install .
```

Requires Python >= 3.10, `httpx[http2]` and `protobuf>=7.35.1,<8`.

## Quick Start

```python
from mlcartifact import ArtifactClient

# Initialize client (it will read ARTIFACT_GRPC_ADDR)
with ArtifactClient() as client:
    # Write an artifact
    res = client.write(
        filename="hello.md",
        content=b"# Hello World",
        description="My first artifact",
    )
    print(f"Artifact saved: {res.id}")

    # Read an artifact
    read_res = client.read(res.id)
    print(f"Content: {read_res.content.decode('utf-8')}")
```

## API

All methods return the protobuf response messages from `mlcartifact.gen.artifact_pb2`.
`user_id` defaults to `ARTIFACT_USER_ID`, `source` to `ARTIFACT_SOURCE`.

| Method | Description |
| :--- | :--- |
| `write(filename, content, description="", user_id=None, source=None, expires_in_hours=24, mime_type="", virtual_path="", metadata=None)` | Store an artifact. `virtual_path` places it in the VFS (e.g. `/projects/alpha/readme.md`); `metadata` is a `dict[str, str]`. |
| `read(id_or_filename, user_id=None)` | Read by ID, filename, or virtual path (starting with `/`). |
| `list(user_id=None, limit=0, offset=0, source="", dir_path="")` | List artifacts. With `dir_path`, lists the direct children of that VFS directory; sub-folders have `is_directory=True`. |
| `delete(id_or_filename, user_id=None)` | Delete by ID, filename, or virtual path. |
| `patch(id_or_path, content, user_id=None, line_start=0, line_end=0, append=False)` | Edit in place. `append=True` appends; otherwise lines `[line_start, line_end)` (0-based, end exclusive) are replaced by `content`. |
| `find(pattern, user_id=None)` | Search by virtual path glob, e.g. `/projects/*/readme.md`. |

### Virtual file system example

```python
with ArtifactClient() as client:
    client.write("readme.md", b"line 1\nline 2", virtual_path="/projects/alpha/readme.md")

    for item in client.list(dir_path="/projects").items:
        print(item.virtual_path, "(dir)" if item.is_directory else "")

    client.patch("/projects/alpha/readme.md", b"LINE 2", line_start=1, line_end=2)
    client.patch("/projects/alpha/readme.md", b"\nline 3", append=True)

    print([i.virtual_path for i in client.find("/projects/*/readme.md").items])
    client.delete("/projects/alpha/readme.md")
```

### Errors

Server errors raise `mlcartifact.ArtifactError` (a subclass of `httpx.HTTPStatusError`)
with the Connect error `code` (e.g. `"not_found"`) and `message`:

```python
from mlcartifact import ArtifactError

try:
    client.read("does-not-exist")
except ArtifactError as e:
    if e.code == "not_found":
        ...
```

## Configuration

The client automatically respects the following environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| ARTIFACT_GRPC_ADDR | localhost:9590 | The address of the server. |
| ARTIFACT_SOURCE | "" | Default source tag. |
| ARTIFACT_USER_ID | "" | Default user ID scoping. |

## Regenerating the protobuf stubs

Only the message classes are needed (the client speaks Connect, not gRPC):

```bash
pip install "grpcio-tools>=1.84.0"
python -m grpc_tools.protoc -I ../proto \
    --python_out=mlcartifact/gen --pyi_out=mlcartifact/gen ../proto/artifact.proto
```

If you regenerate with a newer `grpcio-tools`, raise the `protobuf` floor in
`pyproject.toml` to the "Protobuf Python Version" in the generated `artifact_pb2.py`.

## Example

`example.py` exercises every RPC against a running server:

```bash
ARTIFACT_GRPC_ADDR=localhost:9590 python example.py
```

## Versioning

The library version is available as:

```python
import mlcartifact
print(mlcartifact.__version__)
```

## License

MIT - Copyright (c) 2026 Michael Lechner
