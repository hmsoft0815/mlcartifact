# 🔍 Detail-Dokumentation: mlcartifact

Referenzen auf technische Detail-Dokumente innerhalb des Projekts.
Diese Dateien enthalten tiefere Informationen zu Architektur, Workflows
und Implementierungsdetails — zu umfangreich für die .mlcai/-Docs,
aber wichtig als Nachschlagewerk.

**Hinweis:** Nicht den gesamten Inhalt in den Kontext laden — gezielt
die relevante Datei lesen wenn Details zu einem bestimmten Thema nötig sind.

## Projekt-Root

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [README.md](../README.md) | Englisch: Projekt-Ziele, Setup, Architektur (VFS, Connect RPC) | `README.md` |
| [README.de.md](../README.de.md) | Deutsch: Projekt-Ziele, Setup, Architektur (VFS, Connect RPC, MCP) | `README.de.md` |
| [TODO.md](../TODO.md) | Aktuelle Aufgaben & Roadmap | `TODO.md` |
| [Dockerfile](../Dockerfile) | Multi-stage Build: go build, Alpine-Image, Nicht-Root-User, Ports 8080|9590 | `Dockerfile` |
| [docker-compose.yml](../docker-compose.yml) | Startet server, client-go, client-ts, client-python | `docker-compose.yml` |
| [Taskfile.yml](../Taskfile.yml) | Automatisierung: build, test, docker-compose, generate | `Taskfile.yml` |
| [VERSION](../VERSION) | Server/CLI Version | `VERSION` |
| [VERSION_CLI](../VERSION_CLI) | CLI Version | `VERSION_CLI` |
| [.goreleaser.yml](../.goreleaser.yml) | GoReleaser-Konfiguration: .deb, .rpm, tar.gz, zip Builds | `goos/linux` → `.deb`, `.rpm`, GoReleaser-Targets |
| [CLIENT_GUIDE.md](../docs/CLIENT_GUIDE.md) | Installationshinweise für alle Clients (Go, TS, Python, macOS, Windows), MCP/Connect-RPC API | `docs/CLIENT_GUIDE.md` |
| [go.mod](../go.mod) | Go-Module: `github.com/hmsoft0815/mlcartifact`, Go 1.24.2, Connect RPC, gRPC, protobuf | `go.mod` |

## docs/ & Guides

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [vfs_project_plan.md](vfs_project_plan.md) | Virtueller Dateipfadplan, VFS-Dokumentenkonzept für MCP-Artefakte | `docs/vfs_project_plan.md` |
| [grpc_messaging.md](grpc_messaging.md) | Proto-Message-Definitionen, gRPC-Streaming-Datenfluss, MCP-Tool-Integration | `docs/grpc_messaging.md` |
| [go_library.md](go_library.md) | Detailiert die Go-Client-Library-Integration (embed) | `docs/go_library.md` |
| [how_it_works.png](how_it_works.png) | Architektur-Diagramm des VFS-Systems | `docs/how_it_works.png` |

## client-ts/

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [README.md](../client-ts/README.md) | TS-Client-Übersicht, Connect RPC, Universal Library (Node/Browser/Edge) | `client-ts/README.md` |
| [README.de.md](../client-ts/README.de.md) | TS-Client-Übersicht, Deutsch-Version | `client-ts/README.de.md` |
| [package.json](../client-ts/package.json) | @hmsoft0815/mlcartifact-client, v1.2.1, Vite+Rollup-Tooling | `client-ts/package.json` |
| [examples/](../client-ts/examples/) | TypeScript-Beispiel-Skript | `client-ts/examples/` |

## client-python/

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [README.md](../client-python/README.md) | Python-Client-Übersicht, Connect RPC | `client-python/README.md` |
| [pyproject.toml](../client-python/pyproject.toml) | mlcartifact 0.3.1, httpx, connectrpc, protobuf | `client-python/pyproject.toml` |
| [example.py](../client-python/example.py) | Python-Beispiel-Skript | `client-python/example.py` |

## client-rust/

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [README.md](../client-rust/README.md) | Rust-Client-Library (Roadmap) | `client-rust/README.md` |
| [Cargo.toml](../client-rust/Cargo.toml) | Rust-Crate-Definition | `client-rust/Cargo.toml` |

## proto/

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [artifact.proto](../proto/artifact.proto) | Protobuf-Schema: ArtifactService, Message Types (Write/Read/DELETE/PATCH/FIND) | `proto/artifact.proto` |
| artifact.connect.go | Generierter Connect RPC Service-Stub | `proto/protoconnect/artifact.connect.go` |

## cmd/ & internal/

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| artifact-cli/ | CLI-Implementation (Version Injection) | `cmd/artifact-cli/` |
| artifact-server/ | Server-Implementation (gRPC+SSE+MCP) | `cmd/artifact-server/` |
| internal/grpc/ | gRPC/Connect-RPC Service-Handler | `internal/grpc/` |
| internal/mcp/ | MCP-Server-Logik, Tool-Register | `internal/mcp/` |
| internal/storage/ | Virtual File System Repository mit JSON-Sidecars | `internal/storage/` |
| vfs_exhaustive_test.go | VFS-Randfall-Tests | `internal/storage/vfs_exhaustive_test.go` |

## mlcprodweb/ (Web-Dashboard)

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| [en.md](../mlcprodweb/en.md) | Web-Dashboard-Features (EN) (Roadmap) | `mlcprodweb/en.md` |
| [de.md](../mlcprodweb/de.md) | Web-Dashboard-Features (DE) (Roadmap) | `mlcprodweb/de.md` |
| [meta.yaml](../mlcprodweb/meta.yaml) | Dashboard-Konfiguration (Roadmap) | `mlcprodweb/meta.yaml` |

<!-- Hinweise für das LLM:
- NUR Dateien ausserhalb von .mlcai/ auflisten
- Nach Verzeichnis/Thema gruppieren (nicht eine flache Liste)
- Leere Kategorien entfernen
- Weitere Kategorien bei Bedarf ergänzen (z.B. Frontend, Tests, Config)
-->

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-16
- **Aktualisiert von:** Qwen-3.5
- **Status:** Entwurf