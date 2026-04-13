# MLC Artifact Service

**Persistenter Speicher für das Zeitalter der KI-Agenten**

Der MLC Artifact Service ist mehr als nur ein Storage-Backend. Er bietet ein strukturiertes, hierarchisches **Virtual File System (VFS)**, das speziell für KI-Agenten, verteilte Tools und komplexe Automatisierungs-Workflows entwickelt wurde.

## Warum MLC Artifact Service verwenden?

In vielen agentenbasierten Systemen werden Daten als große Strings oder flache Dateien hin- und hergeschoben. Der MLC Artifact Service führt einen stabilen, pfadbasierten Workspace ein:

- **Hierarchisches VFS**: Organisieren Sie Artefakte in Verzeichnissen wie \`/projekte/alpha/src/main.py\`.
- **Token-Effizienz**: Anstatt eine 100KB Datei für eine 1-zeilige Änderung erneut hochzuladen, nutzen Agenten **VFSPatch** für gezielte Edits.
- **Protokoll-agnostisch**: Egal ob Sie MCP, gRPC oder das CLI verwenden – Ihre Daten sind konsistent und überall zugänglich.
- **Intelligente Lebenszyklen**: Automatisches TTL-Management verhindert das Überlaufen des Speichers, während wichtige Daten dauerhaft erhalten bleiben.

## Kern-Tools

### vfs_ls & vfs_find
Navigieren Sie durch Ihren virtuellen Workspace mit voller Glob-Unterstützung und Verzeichnis-Auflistung.

### vfs_patch
Führen Sie gezielte Updates durch. Hängen Sie Log-Daten an, ersetzen Sie Code-Blöcke oder aktualisieren Sie JSON-Objekte ohne hohen Overhead.

### Integrierte MCP-Prompts
Der Service bringt eigene Nutzungsrichtlinien mit, um Agenten beizubringen, wie sie ihren persistenten Speicher am besten verwalten.

## Erste Schritte

1.  **Laden** Sie die Binärdatei für Ihre Plattform aus den Releases herunter.
2.  **Starten** Sie den Server: \`./artifact-server\`
3.  **Verbinden** Sie sich via MCP, um den Dienst mit Ihrem bevorzugten LLM zu nutzen, oder via gRPC für Ihr eigenes Backend.

## Schnellstart (MCP)

### Claude Desktop
Fügen Sie Folgendes zu Ihrer `claude_desktop_config.json` hinzu:

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
Fügen Sie den Server zu Ihrer `~/.gemini/settings.json` hinzu:

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
Ein neues Profil hinzufügen:

```bash
mcp-tester profile add artifact -c "artifact-server"
```
