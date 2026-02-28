# Artifact MCP-Server (wollmilchsau Ökosystem)

Sichere Speicherung und Abruf von Tool-generierten Inhalten (Markdown, Berichte, Logs), ohne das Projektverzeichnis zu überladen.
Copyright (c) 2026 Michael Lechner. Alle Rechte vorbehalten. Lizenziert unter der MIT-Lizenz.

## Features

- **Sicherer Speicher:** Speichert Textinhalte im verwalteten Verzeichnis `.artifacts/`.
- **Auto-Cleanup:** Entfernt abgelaufene Artefakte automatisch (Standard 24h).
- **LLM Referenz-Tags:** Liefert spezielle `<file id="...">` Tags für den direkten Zugriff durch den Nutzer.
- **Multi-Transport:** Unterstützt sowohl `stdio` als auch `SSE` (HTTP).
- **Strukturiertes Logging:** Nutzt `log/slog` für das Monitoring im Produktivbetrieb.

## Tools

### `write_artifact`
Speichert generierte Inhalte im Artefakt-Speicher.
- `filename`: Der gewünschte Dateiname (z.B. `bericht.md`).
- `content`: Der zu speichernde Textinhalt.
- `expires_in_hours`: (Optional) Stunden, nach denen die Datei gelöscht wird.

## Installation & Build

```bash
make build
# Ausgabe: build/artifact-server
```

## Betrieb

1. **stdio (Standard):**
   ```bash
   ./build/artifact-server
   ```
2. **SSE (HTTP):**
   ```bash
   ./build/artifact-server -addr :8081
   ```

## Tests

```bash
make test-client
```

## Claude Desktop Konfiguration

```json
{
  "mcpServers": {
    "artifact-server": {
      "command": "/absoluter/pfad/zu/artifact-server/build/artifact-server"
    }
  }
}
```
