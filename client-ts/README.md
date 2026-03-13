# @hmsoft0815/mlcartifact-client

A universal, clean, and fully typed TypeScript client for the `mlcartifact` service.

## Overview

The `mlcartifact` service provides a shared storage backend for AI agents and tools. This TypeScript client allows you to easily interact with the service from any environment (Browser, Node.js, Deno, Bun, or Edge functions).

It uses the [Connect](https://connectrpc.com/) protocol, which is a slim, type-safe alternative to traditional gRPC that works seamlessly over standard HTTP/1.1 or HTTP/2.

## Features

- **🚀 Universal:** Works everywhere `fetch` is available.
- **🛡️ Fully Typed:** All requests and responses are strongly typed via Protobuf.
- **🪶 Lightweight:** Minimal dependencies, optimized for modern environments.
- **🔌 Connect Protocol:** Web-friendly, no need for complex gRPC-web proxies.

## Installation

```bash
npm install @hmsoft0815/mlcartifact-client
```

## Quick Start

```typescript
import { ArtifactClient } from '@hmsoft0815/mlcartifact-client';

async function example() {
  // baseUrl defaults to ARTIFACT_GRPC_ADDR or 'http://localhost:9590'
  const client = new ArtifactClient();

  // 1. Write an artifact
  // Supports string, Uint8Array, or Blob
  const writeResp = await client.write('hello.md', '# Hello World', {
    description: 'My first artifact',
    mimeType: 'text/markdown',
    expiresHours: 48,
    metadata: {
      category: 'testing',
      importance: 'high'
    }
  });

  console.log(`Artifact created with ID: ${writeResp.id}`);

  // 2. Read an artifact
  const readResp = await client.read(writeResp.id);
  const text = new TextDecoder().decode(readResp.content);
  console.log(`Content: ${text}`);

  // 3. List artifacts
  const listResp = await client.list({ 
    limit: 5,
    offset: 0
  });
  
  for (const item of listResp.items) {
    console.log(`- ${item.filename} (ID: ${item.id})`);
  }

  // 4. Delete an artifact
  await client.delete(writeResp.id);
}
```

## API Reference

### `new ArtifactClient(baseUrl?: string, transport?: Transport)`

Creates a new client.
- `baseUrl`: The URL of the artifact server. Defaults to `process.env.ARTIFACT_GRPC_ADDR` or `http://localhost:9590`.
- `transport`: Optional custom Connect transport.

### `write(filename: string, content: string | Uint8Array, options?: WriteOptions)`

Saves an artifact to the store.
- `options.userId`: Scope the artifact to a specific user.
- `options.expiresHours`: Number of hours until deletion (default: 24).
- `options.mimeType`: Explicitly set MIME type.
- `options.source`: Identify the creator of the artifact.

### `read(idOrFilename: string, options?: ReadOptions)`

Retrieves an artifact by ID or original filename.

### `list(options?: ListOptions)`

Returns a list of artifacts.
- `options.limit`: Max items to return.
- `options.offset`: Pagination offset.
- `options.userId`: Filter by user.

### `delete(idOrFilename: string, options?: DeleteOptions)`

Permanently removes an artifact.

## Environment Variables (Node.js)

The client automatically picks up these variables:

- `ARTIFACT_GRPC_ADDR`: Server URL (e.g., `https://api.artifacts.local`).
- `ARTIFACT_USER_ID`: Default user ID for all operations.
- `ARTIFACT_SOURCE`: Default source tag for writes.

## Versioning

The library version is exported as a constant:

```typescript
import { version } from '@hmsoft0815/mlcartifact-client';
console.log(version);
```

## Advanced: Custom Transport

If you need to add custom headers (like Auth tokens) to every request:

```typescript
import { createConnectTransport } from "@connectrpc/connect-web";
import { ArtifactClient } from "@hmsoft0815/mlcartifact-client";

const transport = createConnectTransport({
  baseUrl: "http://localhost:9590",
  interceptors: [
    (next) => async (req) => {
      req.header.set("Authorization", "Bearer my-token");
      return await next(req);
    },
  ],
});

const client = new ArtifactClient(undefined, transport);
```

## License

MIT - Copyright (c) 2026 Michael Lechner
