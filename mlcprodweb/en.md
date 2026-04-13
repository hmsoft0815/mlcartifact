# MLC Artifact Service

**Persistent Memory for the Agentic Era**

MLC Artifact Service is more than just a storage backend. It provides a structured, hierarchical **Virtual File System (VFS)** designed specifically for AI agents, distributed tools, and complex automation workflows.

## Why use MLC Artifact Service?

In many agentic systems, data is passed back and forth as large strings or flat files. MLC Artifact Service introduces a stable, path-based workspace:

- **Hierarchical VFS**: Organize artifacts into directories like \`/projects/alpha/src/main.py\`.
- **Token Efficiency**: Instead of re-uploading a 100KB file for a 1-line change, agents use **VFSPatch** for surgical edits.
- **Protocol Agnostic**: Whether you use MCP, gRPC, or the CLI, your data is accessible and consistent.
- **Smart Lifecycles**: Automatic TTL management ensures your storage doesn't clutter while allowing critical data to persist.

## Key Tools

### vfs_ls & vfs_find
Navigate your virtual workspace with full glob support and directory-style listing.

### vfs_patch
Perform surgical updates. Append logs, replace specific code blocks, or update JSON metadata objects without high overhead.

### Integrated MCP Prompts
The service comes with its own usage guidelines to "train" agents on how to best manage their persistent memory.

## Getting Started

1.  **Download** the binary for your platform from the releases.
2.  **Start** the server: \`./artifact-server\`
3.  **Connect** via MCP to use it with your favorite LLM or via gRPC for your custom backend.

## Quick Setup (MCP)

### Claude Desktop
Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mlc-artifact": {
      "command": "artifact-server"
    }
  }
}
```

### Gemini-CLI
Add to your `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "mlc-artifact": {
      "command": "artifact-server"
    }
  }
}
```

### MCP-Tester
Add a new profile:

```bash
mcp-tester profile add artifact -c "artifact-server"
```
