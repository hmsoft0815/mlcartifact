# @hmsoft0815/mlcartifact-client

A universal, clean, and fully typed TypeScript client for the mlcartifact service.

## Overview

The mlcartifact service provides a shared storage backend for AI agents and tools. This TypeScript client allows you to easily interact with the service from any environment (Browser, Node.js, Deno, Bun, or Edge functions).

It uses the [Connect](https://connectrpc.com/) protocol, which is a slim, type-safe alternative to traditional gRPC that works seamlessly over standard HTTP/1.1 or HTTP/2.

## Features

- **Universal:** Works everywhere fetch is available.
- **Fully Typed:** All requests and responses are strongly typed via Protobuf.
- **Lightweight:** Minimal dependencies, optimized for modern environments.
- **Connect Protocol:** Web-friendly, no need for complex gRPC-web proxies.

## Installation

The package is **not published on npm**. Install it from a checkout of the
repository (the build needs `../proto`, so the whole repo is required):

```bash
git clone https://github.com/hmsoft0815/mlcartifact.git
cd mlcartifact/client-ts
npm ci && npm run build        # generates src/gen via buf, compiles to dist/

# in your project: link the local build ...
npm install /path/to/mlcartifact/client-ts
# ... or create a tarball and install that
npm pack                       # -> hmsoft0815-mlcartifact-client-<version>.tgz
```

Requires Node.js >= 20 (or any runtime with `fetch`). Code generation uses the
`@bufbuild/buf` dev dependency; no system `protoc` is needed.

## Quick Start

```typescript
import { ArtifactClient } from '@hmsoft0815/mlcartifact-client';

async function example() {
  // baseUrl defaults to ARTIFACT_GRPC_ADDR or 'http://localhost:9590'
  const client = new ArtifactClient();

  // 1. Write an artifact
  // Supports string, Uint8Array or Blob
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

### Virtual file system (VFS)

```typescript
// write under a virtual path
await client.write('readme.md', '# Alpha', { virtualPath: '/projects/alpha/readme.md' });

// read / delete / patch accept the virtual path instead of the ID
await client.patch('/projects/alpha/readme.md', '\nmore text', { append: true });
await client.patch('/projects/alpha/readme.md', '# Alpha v2', { lineStart: 0, lineEnd: 1 });

// directory listing (sub-directories have isDirectory === true)
const dir = await client.list({ dirPath: '/projects' });

// glob / substring search over virtual paths
const found = await client.find('/projects/*/readme.md');
```

## API Reference

### new ArtifactClient(baseUrl?: string, transport?: Transport)

Creates a new client.
- baseUrl: The URL of the artifact server. Defaults to process.env.ARTIFACT_GRPC_ADDR or http://localhost:9590.
- transport: Optional custom Connect transport.

### write(filename: string, content: string | Uint8Array | Blob, options?: WriteOptions)

Saves an artifact to the store. Strings are UTF-8 encoded; `Blob` works in browsers and Node.js >= 18.
- options.virtualPath: Optional VFS path, e.g. `/projects/alpha/readme.md`.
- options.userId: Scope the artifact to a specific user.
- options.expiresHours: Number of hours until deletion (default: 24).
- options.mimeType: Explicitly set MIME type.
- options.source: Identify the creator of the artifact.

### read(idOrPath: string, options?: ReadOptions)

Retrieves an artifact by ID, original filename or virtual path (starting with `/`).

### list(options?: ListOptions)

Returns a list of artifacts.
- options.limit: Max items to return.
- options.offset: Pagination offset.
- options.userId: Filter by user.
- options.source: Filter by source.
- options.dirPath: VFS directory listing mode — returns the direct children of that directory.

### delete(idOrPath: string, options?: DeleteOptions)

Permanently removes an artifact (by ID, filename or virtual path).

### patch(idOrPath: string, content: string | Uint8Array | Blob, options?: PatchOptions)

Modifies an artifact in place.
- options.append: Append `content` to the end.
- options.lineStart / options.lineEnd: Otherwise replace the 0-based line range `[lineStart, lineEnd)` with `content` (defaults: 0 / lineStart, i.e. insert).
- options.userId: Scope to a user.

### find(pattern: string, options?: FindOptions)

Finds artifacts whose virtual path matches a glob pattern (plain substrings match case-insensitively). Returns a `ListResponse`.

## Environment Variables (Node.js)

The client automatically picks up these variables:

- ARTIFACT_GRPC_ADDR: Server URL (e.g., https://api.artifacts.local).
- ARTIFACT_USER_ID: Default user ID for all operations.
- ARTIFACT_SOURCE: Default source tag for writes.

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
