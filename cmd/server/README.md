# Artifact MCP Server
**Author: Michael Lechner**

The Artifact MCP Server is a core component of the `wollmilchsau` ecosystem, providing a secure, persistent, and high-performance storage layer for LLM-generated assets.

---

## Why an Artifact Server?

In complex agentic workflows, LLMs often face three critical limitations that the Artifact Server solves:

1.  **Context Bloat:** LLMs shouldn't keep large generated reports, logs, or diagrams in their primary conversation context. The Artifact Server allows the LLM to "offload" data and reference it via a stable URI.
2.  **Cross-Tool State:** Different MCP tools (e.g., a "Researcher" and a "Chart Generator") often need to share data. The Artifact Server acts as the common clipboard for the entire agent suite.
3.  **Persistence:** Unlike internal memory, artifacts are saved to disk. This allows for session resumption and auditing of what the LLM actually produced.

## Features

- **Double-Interface:** Accessible via **MCP** (for LLMs) and **gRPC** (for high-speed internal service communication).
- **Auto-Cleanup:** Smart retention policy (default 24h) prevents storage exhaustion.
- **Universal Access:** Bridges the restricted V8 environment (`wollmilchsau`) to the host filesystem.
- **Rich Metadata:** Tracks creation time, source server, and expiration for every file.

---

## Tools for LLMs

### 📝 `write_artifact`
Saves content (text or binary data) to the storage.
- `filename`: Desired filename (e.g., `analysis_report.md`).
- `content`: The payload to store.
- `description`: Optional UTF-8 description of the artifact.
- `expires_in_hours`: Optional expiration (default: 24).

### 📖 `read_artifact`
Retrieves stored content.
- `id`: Unique artifact ID or filename.

### 📋 `list_artifacts`
Returns an overview of all active artifacts, including sizes and expiration dates.

### 🗑️ `delete_artifact`
Manual removal of a specific asset.

---

## Integration Example: Wollmilchsau (TypeScript)

The `artifact-server` provides a specialized bridge for the JavaScript/TypeScript runtime. Scripts can use the global `artifact` object:

```typescript
// Example: Storing a generated chart
const svg = "<svg>...</svg>";
const res = await artifact.write("growth_chart.svg", svg, "image/svg+xml", 24, "Monthly growth visualization");

console.log(`Diagram saved! Access URI: ${res.uri}`);
```

## Internal Architecture

The server is built in **Go** and utilizes `log/slog` for industrial-grade monitoring. It supports multiple transports:
- **stdio:** Standard MCP mode.
- **SSE (HTTP):** For remote or web-based integration.
- **gRPC:** High-performance binary transport for internal clients.

---

## Build & Run

```bash
# Build the binary
make build

# Start in stdio mode (default for proxy)
./build/artifact-server
```

Copyright (c) 2026 **Michael Lechner**. All rights reserved. Licensed under the MIT License.
