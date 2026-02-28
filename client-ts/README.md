# @hmsoft0815/mlcartifact-client

A universal, clean, and fully typed TypeScript client for the `mlcartifact` service.

## Features

- **Universal:** Works in Browser, Node.js, Deno, Bun, and Edge functions (Cloudflare Workers, etc.).
- **Lightweight:** Uses the modern Connect RPC protocol and the standard `fetch` API.
- **Fully Typed:** Auto-generated types from Protobuf for maximum safety.
- **Dual Protocol:** The server supports both standard gRPC and the web-friendly Connect protocol.

## Installation

If the package is published to npm (not yet !):

```bash
npm install @hmsoft0815/mlcartifact-client
```

### Local Development & Usage


**Option A: npm link (Recommended for development)**
1. In `client-ts/`, run: `npm link`
2. In your target project, run: `npm link @hmsoft0815/mlcartifact-client`

**Option B: File Reference**
In your target project's `package.json`, add:
```json
"dependencies": {
  "@hmsoft0815/mlcartifact-client": "file:../path/to/mlcartifact/client-ts"
}
```

## Usage

```typescript
import { ArtifactClient } from '@hmsoft0815/mlcartifact-client';

async function main() {
  // baseUrl defaults to ARTIFACT_GRPC_ADDR env or 'http://localhost:9590'
  const client = new ArtifactClient();

  // Write an artifact (accepts string or Uint8Array)
  const resp = await client.write('test.txt', 'Hello from TypeScript!', {
    description: 'A sample text file',
    expiresHours: 24,
  });
  console.log(`Saved! ID: ${resp.id}, URI: ${resp.uri}`);

  // Read an artifact
  const data = await client.read(resp.id);
  // data.content is a Uint8Array
  console.log(`Content: ${new TextDecoder().decode(data.content)}`);

  // List artifacts
  const list = await client.list({ limit: 10 });
  console.log(`Found ${list.items.length} artifacts`);

  // Delete
  await client.delete(resp.id);
}
```

## Browser Usage

Since this library uses `fetch`, it works directly in the browser. If your server is on a different domain, ensure CORS is enabled on the server (the `mlcartifact` Go server supports Connect out of the box).

## Environment Variables (Node.js)

The client automatically respects these environment variables if available:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `ARTIFACT_GRPC_ADDR` | `http://localhost:9590` | Server base URL |
| `ARTIFACT_SOURCE` | - | Default source for writes |
| `ARTIFACT_USER_ID` | - | Default user ID for all operations |

## License

MIT
