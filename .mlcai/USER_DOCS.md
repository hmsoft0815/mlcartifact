# 📚 Nutzer-Dokumentation: mlcartifact

Übersicht der Dokumentation für Endnutzer und Entwickler die das
Projekt verwenden (nicht entwickeln) wollen.

## Für Endnutzer

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| Quickstart | Erste Schritte, Installation | [README.md](../README.md) |
| Features | Was kann das Projekt, go, client libraries | [README.md](../README.md) |
| MCP Tools | write_artifact, read_artifact, list_artifacts, delete_artifact | [README.md](../README.md) |
| CLI Usage | artifact-cli create, download, list, delete | [README.md](../README.md) |
| Server Konfiguration | CLI-Flags, Umgebungsvariablen, Docker | [README.md](../README.md) |

## Für Entwickler

| Dokument | Beschreibung | Pfad |
|----------|-------------|------|
| Build-Anleitung | prerequisites, Build-Befehle, Dev-Setup | `make run-examples` Beispiel aufrufen |
| Task-Referenz | Verfügbare Build-Tasks (Taskfile.yml) | [Taskfile.yml](../Taskfile.yml) |
| Coding-Standards | Go-Style-Profile im .mlcai/ Style Guide | [MLC Style Guide](https://opencode.ai/docs/standards/go) |

## Plattform-Support

| Plattform | Format | Hinweise |
|-----------|--------|----------|
| Linux | Binary, .deb | Abhängigkeiten: gRPC-Server muss laufen |
| macOS | Binary | Abhängigkeiten: gRPC-Server muss laufen |
| Windows | Binary (via Git Bash/WSL) | Abhängigkeiten: gRPC-Server muss laufen |

## Was ist mlcartifact?

**mlcartifact** ist ein Shared Artifact Store für MCP-Ökosysteme. Es löst das Problem, dass große Datenmengen (SQL-Ergebnisse, PDFs, CSV-Files) nicht durch den LLM-Kontext fließen sollten.

Das **Pattern**: MCP Server A schreibt eine Datei → erhält eine ID → die LLM gibt die ID an Server B weiter → Server B liest die Datei. **Die Daten fließen nie durch den LLM.**

## Weitere Ressourcen

Technische Details, API-Referenz und Architektur sind in den
.mlcai/-Docs im Doc Hub dokumentiert — hier nicht wiederholen.

- **[INTEGRATION.md](/.mlcai/INTEGRATION.md)** - Deployment, gRPC, SSE, Docker
- **[TECH_STACK.md](/.mlcai/TECH_STACK.md)** - go, TypeScript, Python, Rust SDKs
- **[API_CONTRACT.md](/.mlcai/API_CONTRACT.md)** - gRPC-Protokoll, MCP-Prompts
- **[DETAIL_DOCS.md](/.mlcai/DETAIL_DOCS.md)** - Weitere Detail-Dokumente
- **[GO Client Library Guide](docs/go_library.md)** - Detaillierte Anleitung für Go-Entwickler
- **[Python Client](client-python/README.md)** - Python implementation and usage
- **[TypeScript Client](client-ts/README.md)** - Universal TypeScript client (Node, Browser, Edge)

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-17
- **Aktualisiert von:** qwen3.5:35b
- **Status:** Aktuell