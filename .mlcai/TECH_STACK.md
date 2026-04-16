# 🛠 Tech Stack & Constraints: mlcartifact

## Kern-Versionen
- **Sprache:** Go 1.24.2.
- **Framework:** 
  - [Connect RPC](https://connectrpc.com/) (gRPC-kompatibel, HTTP/1.1-fähig).
  - [mcp-go](https://github.com/mark3labs/mcp-go) (MCP SDK).
- **Datenbank:** Keine externe DB erforderlich (Local Filesystem VFS).

## Bibliotheken (Erlaubt/Fixiert)
- **Kommunikation:** `connectrpc.com/connect`, `google.golang.org/grpc`, `github.com/rs/cors`.
- **Protokoll:** `google.golang.org/protobuf`.
- **Testing:** `github.com/stretchr/testify` (Assertions), `internal/storage/vfs_exhaustive_test.go` für VFS-Validierung.
- **Tools:** [Task](https://taskfile.dev/) für Automatisierung, `golangci-lint` für Code-Qualität.

## Einschränkungen (Constraints)
- **Keine externe DB:** Das Projekt muss portabel bleiben und nutzt ausschließlich das lokale Dateisystem (mit JSON-Sidecars für Metadaten).
- **gRPC First:** Für die Service-zu-Service Kommunikation wird gRPC (via Connect RPC) gegenüber REST bevorzugt.
- **Small Files:** Trennung von Logik in `internal/grpc` (API), `internal/mcp` (Server-Logik) und `internal/storage` (VFS/Disk).
- **Cross-Platform:** Alle Pfadoperationen müssen via `path/filepath` gehandhabt werden (Linux/macOS/Windows Support).

## Styling-Regeln
- **Naming:** Variablen auf Englisch, Kommentare können Deutsch sein (bei interner Logik), API-Doku immer Englisch.
- **Architektur:** Repository-Pattern (in `internal/storage`) zur Abstraktion des Dateisystemzugriffs.
- **Fehlerbehandlung:** Explizite Go Error-Patterns (`if err != nil`). Wrapper-Typen für MCP-Kontext nutzen.
- **Versionierung:** Semantische Versionierung (SemVer) ist Pflicht für Releases.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
