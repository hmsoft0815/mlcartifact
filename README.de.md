# mlcartifact

![mlcartifact Workflow und Logo](assets/mlcartifact2.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/hmsoft0815/mlcartifact.svg)](https://pkg.go.dev/github.com/hmsoft0815/mlcartifact)
[![Lizenz: MIT](https://img.shields.io/badge/Lizenz-MIT-yellow.svg)](LICENSE)

Eine Go-Bibliothek zur Kommunikation mit dem **Artifact-Storage-Dienst** über gRPC.
Enthält den Server (`artifact-server`) sowie einen Kommandozeilen-Client (`artifact-cli`).

Copyright (c) 2026 Michael Lechner. Alle Rechte vorbehalten.
Lizenziert unter der MIT-Lizenz.

> 🇬🇧 [English Version](README.md)

## Warum Model Context Protocol (MCP)?

KI-Agenten müssen oft Dateien (Daten, Berichte, Code) generieren oder bestehenden Kontext lesen, um Aufgaben zu erfüllen. Das **Model Context Protocol** bietet eine standardisierte Schnittstelle für die Interaktion zwischen Agenten und Tools.

`mlcartifact` löst das Problem des "flüchtigen Kontexts":
- **Persistenz**: Agenten können Statusinformationen oder generierte Dateien speichern, die über Sitzungen hinweg erhalten bleiben.
- **Kollaboration**: Mehrere Agenten (oder verschiedene MCP-Server wie `wollmilchsau`) können Daten über ein zentrales Hub austauschen.
- **Portabilität**: Dateien werden standardisiert gespeichert und sind via gRPC, HTTP/SSE oder Standard-I/O zugänglich.

---

## Überblick

`mlcartifact` stellt einen sauberen Go-Client bereit, um Artefakte (Dateien, Berichte, Code)
in einem gemeinsamen Speicherdienst zu lesen, zu schreiben, aufzulisten und zu löschen.
Konzipiert für KI-Agenten und MCP-Server, die Dateien über Tool-Grenzen hinweg austauschen müssen.

```
┌──────────────────────────────────────────────┐
│         Deine App / Dein MCP-Server          │
│                                              │
│   import "github.com/hmsoft0815/mlcartifact" │
│   client, _ := mlcartifact.NewClient()       │
│   client.Write(ctx, "bericht.md", daten)     │
└────────────────────┬─────────────────────────┘
                     │ gRPC (:9590)
           ┌─────────▼──────────┐
           │   artifact-server  │
           │  (MCP + gRPC API)  │
           └────────────────────┘
```

---

## Komponenten

| Pfad | Beschreibung |
|------|--------------|
| `.` | Go-Bibliothek — `import "github.com/hmsoft0815/mlcartifact"` |
| `cmd/server` | Eigenständiger Artifact-Storage-Server (gRPC + MCP stdio/SSE) |
| `cmd/cli` | Kommandozeilen-Client für den Server |

---

## Installation

### Bibliothek

```bash
go get github.com/hmsoft0815/mlcartifact
```

### Server & CLI (Vorkompilierte Binaries)

**Der einfachste Weg:** Lade die aktuellen Binaries für Windows, Linux oder macOS direkt von der **[GitHub Releases](https://github.com/hmsoft0815/mlcartifact/releases)** Seite herunter.

### Installation via Go
Wenn Go installiert ist:
```bash
# Server
go install github.com/hmsoft0815/mlcartifact/cmd/server@latest

# CLI
go install github.com/hmsoft0815/mlcartifact/cmd/cli@latest
```

---

## Schnellstart

### Server starten

```bash
# Über stdio (Standard, für MCP-Integration)
artifact-server

# Über SSE (für Netzwerkzugriff)
artifact-server -addr :8082 -grpc-addr :9590 -data-dir /var/artifacts
```

### Bibliothek verwenden

```go
package main

import (
    "context"
    "fmt"

    "github.com/hmsoft0815/mlcartifact"
)

func main() {
    // Verbindet sich mit ARTIFACT_GRPC_ADDR (Standard: :9590)
    client, err := mlcartifact.NewClient()
    if err != nil {
        panic(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Artefakt schreiben
    resp, err := client.Write(ctx, "bericht.md", []byte("# Hallo Welt"),
        mlcartifact.WithMimeType("text/markdown"),
        mlcartifact.WithExpiresHours(48),
    )
    if err != nil {
        panic(err)
    }
    fmt.Println("Artefakt-ID:", resp.Id)

    // Artefakt lesen
    data, err := client.Read(ctx, resp.Id)
    if err != nil {
        panic(err)
    }
    fmt.Println("Inhalt:", string(data.Content))
}
```

---

## CLI Nutzung

Das `artifact-cli` Tool ermöglicht die direkte Interaktion mit dem Speicher-Dienst über das Terminal.

### Verbindung
Der Client verbindet sich mit der gRPC-Schnittstelle des Servers. Die Adresse kann per Umgebungsvariable oder Flag gesetzt werden.

```bash
# Standard: localhost:9590
export ARTIFACT_GRPC_ADDR=localhost:9590

# Oder per Flag
artifact-cli -addr localhost:50051 <befehl>
```

### Beispiele

**Artefakte auflisten:**
```bash
# Alle Artefakte anzeigen (Global + eigene User-ID falls gesetzt)
artifact-cli list

# Mit Paginierung und Benutzer-Filter
artifact-cli list --limit 10 --offset 0 --user meine-id
```

**Datei hochladen:**
```bash
# Einfacher Upload
artifact-cli create ./daten.json

# Detaillierter Upload mit Metadaten
# Hinweis: 'expires' wird in Stunden angegeben (Standard: 24)
artifact-cli create ./bericht.pdf --name "Monatsbericht" --description "Analyse Q1" --user "analyst-1" --expires 72
```

**Artefakt herunterladen:**
```bash
# Per ID oder Dateiname an einen lokalen Zielpfad laden
artifact-cli download xyz123 ./mein-bericht.pdf
```

**Artefakt löschen:**
```bash
artifact-cli delete xyz123 --user "analyst-1"
```

---

## Claude Desktop Integration

Um `mlcartifact` als Tool in Claude Desktop zu nutzen, füge den Server zu deiner Konfigurationsdatei hinzu:

- **MacOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

### Standard-Konfiguration (via Stdio)
Dies ist der einfachste Weg. Claude startet den Server automatisch bei Bedarf.

```json
{
  "mcpServers": {
    "mlcartifact": {
      "command": "/absoluter/pfad/zu/artifact-server",
      "args": ["-data-dir", "/dein/absoluter/pfad/zu/artifacts"]
    }
  }
}
```

### Netzwerk-Konfiguration (via SSE)
Falls der Server bereits in deinem Netzwerk läuft:

```json
{
  "mcpServers": {
    "mlcartifact": {
      "sse": {
        "url": "http://localhost:8082/sse"
      }
    }
  }
}
```

---

## Konfiguration (Server)

| Flag | Standard | Beschreibung |
|------|----------|--------------|
| `-addr` | _(leer)_ | SSE-Adresse. Wenn leer, wird stdio verwendet. |
| `-grpc-addr` | `:9590` | gRPC-Adresse |
| `-data-dir` | `.artifacts` | Verzeichnis für Artifact-Speicherung |
| `-mcp-list-limit` | `100` | Max. Einträge bei `list_artifacts` |

### Umgebungsvariablen (Bibliothek)

| Variable | Beschreibung |
|----------|--------------|
| `ARTIFACT_GRPC_ADDR` | gRPC-Adresse (Standard: `:9590`) |
| `ARTIFACT_SOURCE` | Standard-Quelle für geschriebene Artefakte |
| `ARTIFACT_USER_ID` | Standard-Benutzer-ID für Artifact-Operationen |

---

## MCP-Tools

Als MCP-Server stellt `artifact-server` folgende Tools bereit:

| Tool | Beschreibung |
|------|--------------|
| `write_artifact` | Datei im Artifact-Store speichern |
| `read_artifact` | Datei per ID oder Dateiname abrufen |
| `list_artifacts` | Alle Artefakte eines Benutzers auflisten |
| `delete_artifact` | Artefakt dauerhaft löschen |

---

## Speicherstruktur

```
.artifacts/
├── global/              # Artefakte ohne Benutzer-ID
│   ├── {id}_{dateiname}
│   └── {id}_{dateiname}.json  # Metadaten-Sidecar
└── users/
    └── {user_id}/
        ├── {id}_{dateiname}
        └── {id}_{dateiname}.json
```

---

## Entwicklung

```bash
# Alle Tests ausführen
task test

# Alle Binaries bauen
task build

# Nur den Server bauen
task build-server
```

Alle verfügbaren Befehle sind in der [Taskfile](Taskfile.yml) dokumentiert.

---

## Roadmap

- [ ] **TypeScript / Node.js SDK**: Für Node-basierte MCP-Server und Web-Integrationen.
- [ ] **Python SDK**: Zur nahtlosen Integration in das KI/ML-Ecosystem (LangChain, AutoGen).
- [ ] **Docker Image**: Vorkonfigurierter `artifact-server` für einfaches Deployment.
- [ ] **Visual Dashboard**: Ein Web-Interface zum Durchsuchen und Verwalten gespeicherter Artefakte.

---

## Lizenz

MIT-Lizenz — Copyright (c) 2026 [Michael Lechner](https://github.com/hmsoft0815)
