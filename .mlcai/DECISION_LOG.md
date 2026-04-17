# 📜 Decision Log (ADR)

Dieses Dokument listet fundamentale Entscheidungen auf, die im Projekt getroffen wurden.

## [2026-04-16]: Raw Bytes im Protobuf statt Base64
- **Kontext:** Wahl zwischen kodierter (Base64) und binärer Darstellung großer Dateien.
- **Entscheidung:** Wir nutzen `bytes content = 2` in `WriteRequest` und `ReadResponse`.
- **Grund:** 
  - Base64 erhöht die Payload-Größe um ~33%. Für große Dateien (CSV, PDF, Bilder) ein signifikanter Overhead.
  - Connect RPC und gRPC unterstützen binäre Payloads nativ ohne zusätzliche Serialisierung.
  - Vermeidung von Base64-Decoding im MCP-Client und -Server reduziert CPU-Last.
- **Konsequenz:** Clients müssen mit rohen Bytes umgehen. JSON-Payloads via MCP require base64-codierung beim Transport, aber das Protobuf-Interface bleibt binary-optimal.

## [2026-04-16]: Filesystem-basierter VFS ohne externe Datenbank
- **Kontext:** Persistente Speicherung von Artefakten (CSV, PDF, Bilder, etc.) mit Metadaten.
- **Entscheidung:** Nutzung des lokalen Dateisystems mit JSON-Sidecar-Metadaten statt PostgreSQL, SQLite oder anderer DBs.
- **Grund:**
  - Portabilität: Zero-Infrastructure-Setup — kein Docker-Compose mit DB-Hydratation nötig.
  - MCP-Use-Case: Daten müssen nicht durch den LLM-Kontext fließen sondern bleiben auf Disk.
  - JSON-Sidecars: `{filename}.meta.json` speichert Metadaten direkt neben den Artefakten für schnelle Indexierung.
- **Konsequenz:** VFS-Hierarchie wird als Verzeichnisstruktur simuliert. JSON-Sidecars ermöglichen schnelle Listen-Operationen. Keine ACID-Garantien über Artefakte hinweg. VFS ist jetzt ALPHA.

## [2026-04-16]: gRPC-First via Connect RPC
- **Kontext:** Wahl des RPC-Frameworks für Service-to-Service Kommunikation.
- **Entscheidung:** Connect RPC (statt reinem `google.golang.org/grpc`).
- **Grund:**
  - gRPC-kompatibel, aber HTTP/1.1-fähig (bessere Browser/Proxy-Interoperabilität).
  - Einfachere Client-Entwicklung in Multiple Languages (Go, TS, Python, Rust).
  - `protoc-gen-connect-go` generiert idiomatic Go-Clients.
  - SSE-Support für den MCP-Server (Web-Socket-Alternative).
- **Konsequenz:** Alle Clients (Go, TypeScript, Python) nutzen Connect RPC. MCP-Tools nutzen den gleichen Connect-Handler wie der native gRPC-Server.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-16
- **Aktualisiert von:** Qwen-3.5
- **Status:** Entwurf