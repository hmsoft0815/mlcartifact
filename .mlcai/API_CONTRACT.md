# 🔌 API Contract: mlcartifact

## 1. Bereitgestellte Endpunkte (Exposed)

### gRPC / Connect RPC Service
- **Service:** `artifact.v1.ArtifactService`
- **Port:** `9590` (Standard) / `-grpc-addr` (CLI-Flag)
- **Transport:** gRPC über HTTP/2 oder HTTP/1.1 via Connect RPC (h2c cleartext)
- **Protobuf-Paket:** `artifact.v1`
- **Go-Package:** `github.com/hmsoft0815/mlcartifact/proto`

#### Connect RPC HTTP-Routen

```
POST /artifact.v1.ArtifactService/Write
POST /artifact.v1.ArtifactService/Read
POST /artifact.v1.ArtifactService/Delete
POST /artifact.v1.ArtifactService/List
POST /artifact.v1.ArtifactService/Patch
POST /artifact.v1.ArtifactService/Find
```

#### RPC-Signaturen

**Write(WriteRequest)** → WriteResponse
- Speichert ein Binär-Artefakt im lokalen Dateisystem.
- **WriteRequest-Fields:**
  - `filename` (string): Dateiname für das Artefakt.
  - `content` (bytes): Rohdaten (kein Base64!).
  - `mime_type` (string, optional): MIME-Typ (wird aus Dateiname abgeleitet).
  - `expires_hours` (int32, optional): Verfallszeit in Stunden (Standard: 24).
  - `source` (string, optional): Servername für Auditing.
  - `metadata` (map<string, string>, optional): Freie Metadaten.
  - `user_id` (string, optional): UUID für Multi-User-Isolation.
  - `description` (string, optional): Freitextbeschreibung.
  - `virtual_path` (string, optional): Virtueller Pfad für VFS-Hierarchie.
- **WriteResponse-Fields:**
  - `id` (string): Eindeutige ID des Artefakts.
  - `filename` (string): Bereinigter Dateiname.
  - `uri` (string): Referenz-URI im Format `artifact://<filename>`.
  - `expires_at` (string): Gültigkeitsende (ISO 8601).
  - `virtual_path` (string): Normalisierter VFS-Pfad.

**Read(ReadRequest)** → ReadResponse
- Holt ein Artefakt nach ID, Dateiname oder `virtual_path` (Pfade beginnen mit `/`).
- **ReadRequest-Fields:**
  - `id` (string): Artefakt-ID, Dateiname oder virtueller Pfad.
  - `user_id` (string, optional): Scopes die Suche auf einen Nutzer ein.
- **ReadResponse-Fields:**
  - `content` (bytes): Rohdaten des Artefakts.
  - `mime_type` (string): MIME-Typ.
  - `filename` (string): Dateiname.
  - `virtual_path` (string): Virtueller Pfad.

**Delete(DeleteRequest)** → DeleteResponse
- Löscht ein Artefakt.
- **DeleteRequest-Fields:**
  - `id` (string): Artefakt-ID, Dateiname oder virtueller Pfad.
  - `user_id` (string, optional): Scopes auf Nutzer.
- **DeleteResponse-Fields:** `deleted` (bool).

**List(ListRequest)** → ListResponse
- Listet Artefakte auf, optional gefiltert nach Quelle oder User.
- **ListRequest-Fields:**
  - `source` (string, optional): Filter nach Quelle.
  - `user_id` (string, optional): Scopes auf Nutzer.
  - `limit` (int32, optional): Obergrenze Ergebnisse.
  - `offset` (int32, optional): Versatz (Pagination).
  - `dir_path` (string, optional): Aktiviert VFS-Pfadliste (hierarchisch).
- **ListResponse-Fields:** `items` (repeated ArtifactInfo).

**ArtifactInfo-Fields:** `id`, `filename`, `mime_type`, `source`, `created_at`, `expires_at`, `size_bytes`, `user_id`, `description`, `virtual_path`, `is_directory`.

**Patch(PatchRequest)** → PatchResponse
- Ändert ein bestehendes Artefakt (z.B. Zeilenersetzung, Anhängen).
- **PatchRequest-Fields:**
  - `id` (string): ID oder virtueller Pfad.
  - `user_id` (string, optional): Scopes auf Nutzer.
  - `content` (bytes): Eingefügter/zu ersetzender Inhalt.
  - `line_start` (int32, optional): Startzeile (0-basiert).
  - `line_end` (int32, optional): Endzeile (0-basiert).
  - `append` (bool): Wenn `true`, wird an das Ende angehängt.
- **PatchResponse-Fields:** `success` (bool), `new_size` (int64), `updated_at` (string).

**Find(FindRequest)** → ListResponse
- Glob-basierte Suche nach Artefakten.
- **FindRequest-Fields:**
  - `user_id` (string, optional): Scopes auf Nutzer.
  - `pattern` (string): Glob-Pattern (z.B. `*.txt`, `**/*.log`).

---

## 2. MCP-Schnittstelle (stdio / SSE)

Der Server exponiert MCP-Tools über stdio (Standard) oder SSE (optional, via `-addr` Flag).

### MCP-Tools (7 Tools)

| Tool | Beschreibung | Entspricht RPC |
|------|-------------|----------------|
| `write_artifact` | Erstellen eines Artefakts | `Write` |
| `read_artifact` | Lesen eines Artefakts | `Read` |
| `list_artifacts` | Flat-Liste aller Artefakte (max. `-mcp-list-limit`, Standard: 100) | `List` |
| `delete_artifact` | Löschen eines Artefakts | `Delete` |
| `vfs_ls` | VFS-Verzeichnisinhalt auflisten (Dateien + Unterverzeichnisse) | `List` (mit `dir_path`) |
| `vfs_patch` | Chirurgische Änderung (Zeilenersetzung / Anhängen) | `Patch` |
| `vfs_find` | Glob-Pattern-Suche im VFS | `Find` |

### MCP-Prompts

| Prompt | Beschreibung |
|--------|-------------|
| `vfs_usage` | Anleitung zur hierarchischen VFS-Nutzung, Patching und Discovery-Patterns |

### Transport-Modi

- **stdio** (Standard): Für Claude Desktop / lokale MCP-Integration. Kein `-addr` Flag nötig.
- **SSE** (optional): HTTP Server-Sent Events, aktiviert via `-addr :8082` (Taskfile-Standard) oder `-addr :8080` (Docker-Standard).

---

## 3. Server-Konfiguration

### CLI-Flags

| Flag | Standard | Beschreibung |
|------|----------|-------------|
| `-grpc-addr` | `:9590` | gRPC/Connect RPC Listen-Adresse |
| `-addr` | (leer) | SSE-Adresse; wenn leer, wird stdio verwendet |
| `-data-dir` | `~/mlcartifact/storage` | Basis-Verzeichnis für Artefakt-Speicherung |
| `-mcp-list-limit` | `100` | Obergrenze für `list_artifacts` Ergebnisse |
| `-dump` | `false` | MCP-Tool-Definitionen als JSON ausgeben |
| `-version` | `false` | Version ausgeben |

### Umgebungsvariablen

| Variable | Standard | Verwendet von | Beschreibung |
|----------|----------|--------------|-------------|
| `ARTIFACT_GRPC_ADDR` | `localhost:50051` | CLI, Go-Client | gRPC-Serveradresse |
| `ARTIFACT_SOURCE` | (leer) | Go-Client | Standard-Source-Tag |
| `ARTIFACT_USER_ID` | (leer) | CLI, Go-Client | Standard User-ID |

### Docker

- **Exponierte Ports:** 8080 (SSE), 9590 (gRPC)
- **Startbefehl:** `-addr :8080 -grpc-addr :9590 -data-dir /app/data`
- **Volume:** `/app/data` (Named Volume `artifacts-data`)
- **Base Image:** `alpine:3.21` (Non-Root User `mlc`)

---

## 4. Speicher-Layout

```
{data-dir}/
├── global/                    # Global zugängliche Artefakte
│   ├── {id}_{filename}        # Binär-Artefakt
│   └── {id}_{filename}.json   # Metadaten-Sidecar
└── users/
    └── {user_id}/             # User-Scoped Artefakte
        ├── {id}_{filename}
        └── {id}_{filename}.json
```

---

## 5. Konsumierte APIs (Consumed)

- Keine externen oder internen Fremddienste.
- Liest/Schreibt rein auf das lokale Dateisystem (`-data-dir`).

---

## 6. Globale Konventionen

- **Datumsformat:** Immer ISO 8601 (UTC).
- **Paginierung:** Über `limit` und `offset` (gRPC-Listen).
- **Isolation:** Multi-User-Isolation via `user_id` Strings (kein RBAC, geeignet für interne Tool-Orchestrierung).
- **Versionierung:** Protobuf-Definition in `proto/artifact.proto`; API unterliegt SemVer.
- **VFS-Pfade:** Beginnen immer mit `/`, werden normalisiert (`path.Clean`).
- **MIME-Erkennung:** Automatisch aus Dateiendung (Fallback: `application/octet-stream`).

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-16
- **Aktualisiert von:** Claude Opus 4.6
- **Status:** Aktuell
