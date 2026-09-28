# mlcartifact - Der gemeinsame Speicher für MCP-Ökosysteme

> **[mlcgo.eu](https://mlcgo.eu)** — Werkzeuge, Bibliotheken und Handbücher · [Produktseite](https://mlcgo.eu/products/mlcartifact/)


> **Große Daten gehören nicht in den LLM-Kontext.** Lass MCP-Server Dateien direkt in einen gemeinsamen Speicher schreiben und nur eine ID austauschen. Das LLM orchestriert - ohne die Rohdaten je zu sehen.

![mlcartifact Architecture](docs/how_it_works.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/hmsoft0815/mlcartifact.svg)](https://pkg.go.dev/github.com/hmsoft0815/mlcartifact)
[![Lizenz: MIT](https://img.shields.io/badge/Lizenz-MIT-yellow.svg)](LICENSE)

Copyright (c) 2026 Michael Lechner. Lizenziert unter der MIT-Lizenz.

> [English Version](README.md)

---

## Das Problem: Große Daten gehören nicht in den LLM-Kontext

Stell dir vor: Ein SQL-MCP-Server liefert 50.000 Zeilen zurück. Oder ein Report-Generator erzeugt ein 2MB-PDF. Fließen diese Ergebnisse durch das Kontext-Fenster des LLMs:

- **Werden Tokens verschwendet** - massiv
- **Wird das Kontextlimit gesprengt** - häufig
- **Wird alles langsamer** - unnötig

**mlcartifact** ist die Lösung: ein gemeinsamer Artefakt-Speicher. MCP-Server schreiben Ergebnisse direkt hinein und teilen dem LLM nur mit: *„Fertig. Artefakt-ID: `abc123`. Spalten: name, summe, datum."*

---

## Für wen ist mlcartifact gedacht?

**mlcartifact ist in erster Linie für den Betrieb mit lokalen LLMs gedacht** (etwa über Ollama, llama.cpp oder LM Studio) und für eigene Harnesses. Dort gibt es meist keinen eingebauten Weg, auf dem Werkzeuge Dateien untereinander weiterreichen: Jedes Zwischenergebnis müsste durch den Kontext des Modells, und genau der ist bei lokaler Hardware knapp und langsam.

Professionelle Plattformen wie Claude (Anthropic) oder Gemini (Google) bringen in der Regel einen eigenen, vergleichbaren Mechanismus mit, zum Beispiel eine Sandbox mit Dateisystem oder eine Datei-API. Große Dateien bleiben dort zwischen zwei Tool-Aufrufen außerhalb des Modellkontexts und müssen gar nicht erst über das Netzwerk zum Modell und zurück. Wer eine solche Plattform nutzt, sollte zuerst deren eingebaute Lösung prüfen. mlcartifact lohnt sich dort vor allem dann, wenn eigene MCP-Server über einen gemeinsamen Speicher direkt Daten austauschen sollen.

---

## Das Muster: MCP-Server tauschen Daten direkt aus

```
LLM: "Führe den SQL-Quartalsbericht aus und erzeuge daraus ein PDF."

  MCP-Server A (SQL)      mlcartifact         MCP-Server B (PDF)
       |                       |                       |
       |-- write_artifact() -->|                       |
       |   bericht.csv (2MB)   |                       |
       |<-- artifact ID: abc123|                       |
       |                       |                       |
       +-- sagt LLM: "Fertig." |                       |
                               |                       |
LLM: "PDF-Server: erstelle aus Artefakt abc123 ein PDF."
                               |                       |
                               |<-- read_artifact(id) -|
                               |    (liest 2MB CSV)    |
                               |---------------------->|
```

**Die großen Daten fließen nie durch das LLM.** Nur Artefakt-IDs werden ausgetauscht. Das LLM orchestriert - es trägt keine Daten.

---

## Warum gRPC & das Artefakt-Muster?

Über einfache lokale Dateispeicherung hinaus nutzt `mlcartifact` einen gRPC-zentrierten Ansatz, um die spezifischen Herausforderungen verteilter MCP-Ökosysteme zu lösen:

- **Nahtlose Portabilität**: Dienste können auf dem Host, in Docker-Containern oder auf entfernten Servern laufen. Alle verbinden sich via gRPC ohne gemeinsame Volumes oder komplexe Dateisystemrechte.
- **Erhöhte Sicherheit (Sandboxing)**: MCP-Server benötigen keinen Vollzugriff auf das Dateisystem des Hosts. Sie interagieren nur mit der Artefakt-API, was eine sichere Grenze zwischen deinen Daten und potenziell unsicheren Tools schafft.
- **Multi-Server-Datenaustausch**: Ermöglicht das „Shared Memory“-Muster, bei dem Server A Daten schreibt und Server B sie liest, orchestriert durch das LLM über IDs.
- **Metadaten & Lebenszyklus**: Automatische MIME-Typ-Erkennung, Herkunftsnachweise und ein **automatisches Ablaufdatum**.

### Vergleich: gRPC API vs. Lokales Dateisystem

| Feature | `mlcartifact` (gRPC) | Lokales Dateisystem (`/tmp`, etc.) |
| :--- | :--- | :--- |
| **Isolierung** | **Hoch** (API-Grenze) | **Niedrig** (OS-Berechtigungen nötig) |
| **Portabilität** | **Universell** (Netzwerkbasiert) | **Host-gebunden** (Shared Volumes nötig) |
| **Benutzer-Isolation**| Eingebaute Scoping-Logik | Manuelle Rechteverwaltung |
| **Bereinigung** | Automatisch (TTL-basiert) | Manuell oder via Cron-Job |
| **Performance** | Netzwerklatenz (ms) | Festplattengeschwindigkeit |
| **Komplexität** | Benötigt Server-Prozess | Kein zusätzlicher Prozess |

**Abwägung (Tradeoffs)**: Obwohl gRPC eine geringe Netzwerklatenz einführt und einen laufenden Server-Prozess erfordert, überwiegen in produktiven MCP-Umgebungen die Vorteile in Bezug auf Sicherheit, Multi-Server-Orchestrierung und vereinfachtes Deployment meist deutlich.

---

## Was ist in diesem Repository?

| Komponente | Beschreibung |
|---|---|
| **`artifact-server`** | MCP + gRPC Server. Speichert und liefert Artefakte. MCP über stdio, Streamable HTTP (`/mcp`) und das ältere SSE (`/sse`), Protokoll 2026-07-28, gebaut auf dem offiziellen [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). |
| **`artifact-cli`** | Kommandozeilen-Tool zum Hochladen, Herunterladen, Auflisten und Löschen. |
| **Go-Bibliothek** | `import "github.com/hmsoft0815/mlcartifact/client"` - direkt in jeden MCP-Server einbettbar. |
| **TypeScript-Client** | In `client-ts/` - universeller Client (Node, Browser, Edge) mittels Connect RPC. Installation: siehe [client-ts/README.de.md](client-ts/README.de.md). |
| **Python-Client** | In `client-python/` - Connect-Client auf Basis von httpx. |
| **Rust SDK** | In `client-rust/` - gRPC-Client mittels Tonic. |

Alle vier Clients decken die komplette API ab: Schreiben, Lesen, Auflisten, Löschen und die VFS-Funktionen (virtuelle Pfade, Verzeichnisse, Patch, Suche).

## Ökosystem & Verwandte Projekte

- **[wollmilchsau](https://github.com/hmsoft0815/wollmilchsau)** - Ein „Eierlegende-Wollmilchsau“-MCP-Server, der JavaScript- bzw. TypeScript-Skripte ausführen kann, die als Artefakte in `mlcartifact` gespeichert sind. Dies ermöglicht dynamische Tool-Ausführung, bei der das LLM ein Skript in den Artefakt-Speicher schreibt und `wollmilchsau` es in einer sicheren Umgebung ausführt.
- **[mcp-tester](https://github.com/hmsoft0815/mlc_mcptester)** - Ein CLI-Testwerkzeug und Skript-Runner für MCP-Server. Wird in `mlcartifact` eingesetzt, um jedes MCP-Tool und jeden Parameter mit deklarativen Assertions integrativ zu testen.

---

## Dokumentation

- **[Go-Client-Bibliothek Handbuch](docs/go_library.md)** - Umfassender Guide für Go-Entwickler.
- **[gRPC-API-Referenz](docs/grpc_messaging.md)** - Detaillierte technische Referenz für das gRPC-Protokoll.

---

## Schnellstart

### Installation

```bash
# via Installations-Script (Linux/macOS)
curl -sfL https://raw.githubusercontent.com/hmsoft0815/mlcartifact/main/scripts/install.sh | sh

# oder via Go
go install github.com/hmsoft0815/mlcartifact/cmd/artifact-server@latest
go install github.com/hmsoft0815/mlcartifact/cmd/artifact-cli@latest
```

Vorkompilierte `.deb`, `.rpm` und Binaries unter **[GitHub Releases](https://github.com/hmsoft0815/mlcartifact/releases)**.

### Server starten

```bash
# stdio-Modus (für Claude Desktop / MCP)
artifact-server -data-dir ~/mlcartifact/storage

# HTTP-Modus: Streamable HTTP auf /mcp, älteres SSE auf /sse
artifact-server -addr :8082 -grpc-addr :9590 -data-dir ~/mlcartifact/storage
```

### Go-Bibliothek in deinem MCP-Server nutzen

Siehe das **[Go-Client-Bibliothek Handbuch](docs/go_library.md)** für detaillierte Beispiele.

```go
import "github.com/hmsoft0815/mlcartifact/client"

// Verbinden (liest ARTIFACT_GRPC_ADDR, Standard: :9590)
c, _ := client.NewClient()
defer c.Close()

// Großes Ergebnis speichern - liefert eine ID, keine Daten
resp, _ := c.Write(ctx, "bericht.csv", csvDaten,
    client.WithMimeType("text/csv"),
    client.WithExpiresHours(24),
)

// Dem LLM mitteilen: "Fertig. ID: abc123. Spalten: name, summe, datum."
fmt.Println("artifact_id:", resp.Id)
```

---

## Claude Desktop Integration

```json
{
  "mcpServers": {
    "mlcartifact": {
      "command": "/pfad/zu/artifact-server",
      "args": ["-data-dir", "/dein/artifacts-pfad"]
    }
  }
}
```

Oder Verbindung zu einem laufenden Server über HTTP (der Server muss vorher gestartet sein). Aktuelle Clients nutzen Streamable HTTP; die genauen Schlüssel hängen vom Client ab:
```json
{
  "mcpServers": {
    "mlcartifact": {
      "type": "http",
      "url": "http://localhost:8082/mcp"
    }
  }
}
```

Ältere Clients, die nur SSE sprechen, nutzen weiterhin `http://localhost:8082/sse`.

---

## MCP-Tools

| Tool | Beschreibung |
|---|---|
| `write_artifact` | Datei speichern, optional unter einem `virtual_path` - liefert eine ID und einen `<file>`-Referenz-Tag |
| `read_artifact` | Datei per ID oder virtuellem Pfad abrufen |
| `list_artifacts` | Gespeicherte Artefakte auflisten (flach) |
| `vfs_ls` | Virtuelles Verzeichnis auflisten |
| `vfs_find` | Per Glob-Muster oder Stichwort suchen |
| `vfs_patch` | Zeilen ersetzen (`line_start` einschließlich, `line_end` ausschließlich) oder anhängen |
| `delete_artifact` | Dauerhaft löschen |

Alle Tools außer `read_artifact` liefern strukturierte Ergebnisse mit Output-Schema; Fehler kommen als Tool-Fehler (`isError`). Der Prompt `vfs_usage` erklärt dem Modell das virtuelle Dateisystem.

---

## CLI Nutzung

```bash
# Hochladen, Herunterladen, Auflisten, Löschen
artifact-cli create ./bericht.csv --name "Q1-Bericht" --expires 72
artifact-cli create -q ./script.sh             # Quiet-Modus: gibt nur die Artefakt-ID aus
artifact-cli download abc123 ./lokale-kopie.csv
artifact-cli list
artifact-cli delete abc123
```

Globale Optionen:
- `-addr`: gRPC-Serveradresse (Standard: `ARTIFACT_GRPC_ADDR` oder `localhost:9590`)
- `-token`: Authentifizierungs-Token für Remote-Zugriff (Standard: `ARTIFACT_GRPC_TOKEN` oder `ARTIFACT_TOKEN`)
- `-user`: Standard-Benutzer-ID (Standard: `ARTIFACT_USER_ID`)

---

## Server-Konfiguration

| Flag | Standard | Beschreibung |
|---|---|---|
| `-addr` | _(leer)_ | HTTP-Adresse (z. B. `127.0.0.1:8080` oder `:8080` für lokal, `0.0.0.0:8080` für alle): Streamable HTTP auf `/mcp`, SSE auf `/sse`. Nicht-Loopback erfordert Token. Leer = stdio-Modus. |
| `-grpc-addr` | `127.0.0.1:9590` | gRPC/Connect-Adresse (z. B. `127.0.0.1:9590` oder `:9590` für lokal, `0.0.0.0:9590` für alle Schnittstellen). Nicht-Loopback erfordert Token. |
| `-grpc-token` | _(leer)_ | Authentifizierungs-Token für Remote-Zugriff. Kann auch per `ARTIFACT_GRPC_TOKEN` gesetzt werden. |
| `-require-token-localhost` | `false` | Token-Pflicht auch für Localhost / Loopback-Verbindungen (Standard: false, Localhost verbindet ohne Token). |
| `-cors-origins` | _(leer)_ | Kommagetrennte Liste erlaubter Browser-CORS-Origins (Standard: keine / alle Cross-Origin-Browser-Anfragen abgewiesen). |
| `-data-dir` | `~/mlcartifact/storage` | Speicherverzeichnis |
| `-mcp-list-limit` | `100` | Max. Einträge bei `list_artifacts` |

**Umgebungsvariablen (Bibliothek & Server):**

| Variable | Beschreibung |
|---|---|
| `ARTIFACT_GRPC_ADDR` | gRPC-Adresse (Standard: `127.0.0.1:9590`) |
| `ARTIFACT_GRPC_TOKEN` | Authentifizierungs-Token für Remote-Zugriff (prüft auch `ARTIFACT_TOKEN`; für Localhost ignoriert, außer bei `-require-token-localhost`) |
| `ARTIFACT_REQUIRE_TOKEN_LOCALHOST` | Auf `true` oder `1` setzen, um Token auch für Localhost zu verlangen |
| `ARTIFACT_CORS_ORIGINS` | Kommagetrennte Liste erlaubter Browser-CORS-Origins |
| `ARTIFACT_SOURCE` | Standard-Quell-Tag |
| `ARTIFACT_USER_ID` | Standard-Benutzer-ID |

---

## Authentifizierung & Sicherheit

`mlcartifact` ist standardmäßig auf sicheren Betrieb ausgelegt:

- **Localhost ohne Token (Zero-Config)**: Verbindungen von Loopback-Adressen (`127.0.0.1`, `::1`) erfordern standardmäßig **kein** Authentifizierungs-Token. Der Server kann lokal ohne Zusatzaufwand gestartet und von CLI oder SDKs genutzt werden.
- **Remote-Zugriff (Token-Pflicht)**: Sobald der Server auf externen Schnittstellen lauscht (z. B. `0.0.0.0` oder eine Netzwerk-IP), wird unautorisierter Remote-Zugriff verweigert. Ein Token muss serverseitig via `-grpc-token <token>` oder `ARTIFACT_GRPC_TOKEN=<token>` hinterlegt werden.
- **Token-Pflicht für Localhost erzwingen**: Um ein Token auch bei Loopback-Verbindungen zu verlangen (z. B. auf geteilten Multi-User-Rechnern), wird `-require-token-localhost` übergeben oder `ARTIFACT_REQUIRE_TOKEN_LOCALHOST=true` gesetzt.
- **CORS-Schutz**: Browser-übergreifende Anfragen (Cross-Origin) sind standardmäßig gesperrt. Erlaubte Origins können mit `-cors-origins "https://example.com"` freigegeben werden.

### Token in den Clients verwenden

Alle Clients werten automatisch die Umgebungsvariable `ARTIFACT_GRPC_TOKEN` (oder `ARTIFACT_TOKEN`) aus und bieten zudem programmatische Optionen:

- **CLI**: `artifact-cli -token "<token>" ...` oder via `ARTIFACT_GRPC_TOKEN`
- **Go**: `client.NewClientWithAddr(addr, client.WithToken("<token>"))`
- **TypeScript**: `new ArtifactClient(addr, undefined, "<token>")`
- **Python**: `ArtifactClient(addr, token="<token>")`
- **Rust**: `ArtifactClient::connect_with_token(addr, "<token>")` oder via `ARTIFACT_GRPC_TOKEN`
- **HTTP / MCP**: Header `Authorization: Bearer <token>` mitliefern

---

## Speicherstruktur

```
~/mlcartifact/storage/
├── global/
│   ├── {id}_{dateiname}
│   └── {id}_{dateiname}.json   # Metadaten-Sidecar
└── users/
    └── {user_id}/
        ├── {id}_{dateiname}
        └── {id}_{dateiname}.json
```

---

## Entwicklung & Tests

```bash
# Unit-Tests für alle Komponenten (Go, TS, Python, Rust) ausführen
task test:all             # oder: make test-all

# Integrationstests mit mcp-tester ausführen
task test:integration     # oder: make test-integration

# Binaries kompilieren
task build                # oder: make build
```

### MCP-Server Integrationstests mit `mcp-tester`

Die MCP-Server-Implementierung wird kontinuierlich und vollständig mit **[mcp-tester](https://github.com/hmsoft0815/mlc_mcptester)** (getestet gegen **v1.6.2**) auf Protokoll- und Verhaltenskonformität geprüft:

- **Test-Skript**: [`tests/integration.mcp`](tests/integration.mcp)
- **Score / Ergebnis**: **54 von 54 Prüfungen bestanden (100% Erfolgsquote, 0 Fehler)**
- **Abdeckung**:
  - Alle 7 MCP-Tools: `write_artifact`, `read_artifact`, `list_artifacts`, `vfs_ls`, `vfs_find`, `vfs_patch`, `delete_artifact`
  - Sämtliche Pflicht- und optionalen Parameter (`virtual_path`, `mime_type`, `description`, `expires_in_hours`, `metadata`, `user_id`, `line_start`, `line_end`, `append`)
  - Mandanten- und Benutzerisolation (`user_id`-Trennung zwischen verschiedenen Benutzern)
  - VFS-Line-Patching (zeilenweises Ersetzen mit `line_start`/`line_end` und Anhängen mit `append`)
  - Fehlerbehandlung und Schema-Validierung (fehlende Pflichtfelder, leere Inhalte, unbekannte IDs/Pfade)

Integrationstests ausführen:
```bash
task test:integration
# oder
make test-integration
```

---

## Roadmap

- [x] **TypeScript / Node.js SDK**
- [x] **Python SDK** (httpx, Connect-Protokoll)
- [x] **Docker Image** - vorkonfigurierter Server
- [x] **Rust SDK** (Tonic-basiert)
- [ ] **Web Dashboard** - Artefakte im Browser verwalten

## Referenz

Das **[MCP-Handbuch](https://mlcgo.eu/books/mcp-handbuch/)** erklärt das Model Context Protocol von Grund auf —
Tools, Resources, Prompts, Transporte, Sicherheit und das Artifact-Pattern.
Auf Deutsch und Englisch.

---

## Lizenz

MIT-Lizenz - Copyright (c) 2026 [Michael Lechner](https://github.com/hmsoft0815)

<!-- mlcai-private -->
## Projektdokumentation (`.mlcai/`)

`.mlcai/` ist ein **privates Git-Submodul**: interne Planung, Backlog und Arbeitsnotizen, gepflegt mit dem MLC Doc Hub. Es ist nicht öffentlich zugänglich — **ohne** `--recurse-submodules` klonen; für den Build wird es nicht gebraucht. Links nach `.mlcai/` funktionieren nur mit Zugriff (`git submodule update --init .mlcai`).
