# 🤖 AI & Integration Context: mlcartifact

## 1. Identität & Zweck
- **Kernaufgabe:** Zentraler Artifact-Store für das MCP-Ökosystem ("Shared Memory"). Verhindert, dass große Datenmengen (CSV, PDF, Bilder) durch den LLM-Kontext fließen müssen.
- **Technischer Stack:** Go, Connect RPC / gRPC, MCP (stdio/SSE).
- **Hoster/Infrastruktur:** Docker-Compose, Binary (via Taskfile), MCP-Client Integration (z.B. Claude Desktop).

## 2. Die "Nachbarschaft" (System-Kontext)
- **Upstream (Wovon hänge ich ab?):**
  - Keine externen Dienst-Abhängigkeiten. Nutzt das lokale Dateisystem zur Speicherung.
- **Downstream (Wer nutzt mich?):**
  - **[wollmilchsau](https://github.com/hmsoft0815/wollmilchsau):** Nutzt mlcartifact als Sandbox zum Speichern und Ausführen von Skripten (Python, Bash).
  - Beliebige MCP-Server, die große Datenmengen produzieren (z.B. SQL-Exporte).
- **Shared Resources:**
  - Nutzt ein konfigurierbares Datenverzeichnis (`-data-dir`) zur persistenten Speicherung der Artefakte und Metadaten (JSON-Sidecars).

## 3. Schnittstellen-Vertrag
- **Primäre API:** 
  - **gRPC / Connect RPC:** Port `9590` (Standard). Für High-Performance Service-to-Service Kommunikation.
  - **MCP (stdio/SSE):** Port `8080` (Standard für SSE). Für die direkte Interaktion mit LLMs.
- **Auth-Mechanismus:** Isolierung über optionale `user_id` Strings in den Requests. Aktuell kein komplexes RBAC-System.
- **Wichtige Datenmodelle:**
  - `Artifact`: { id: string, filename: string, content: bytes, mime_type: string, expires_at: string, user_id: string }
- **API-Doku-Link:** Details in `docs/grpc_messaging.md` und `proto/artifact.proto`.

## 4. Leitplanken & Regeln
- **Naming:** Go-Konventionen (PascalCase für Exporte, camelCase für interne Variablen). Protobuf nutzt snake_case für Felder.
- **Testing:** `task test` führt alle Unit- und Integrationstests aus. Neue Features erfordern entsprechende Tests in `internal/storage` oder `client/`.
- **Sicherheit:** MCP-Server benötigen keinen direkten Dateisystemzugriff, sondern kommunizieren nur über die API (Sandboxing-Vorteil).

## 5. Aktueller Fokus (Status)
- **Bekannte Probleme:** VFS (Virtual File System) Support ist in der Alpha-Phase.
- **Nächste Schritte:** Implementierung eines Web-Dashboards zur visuellen Verwaltung der Artefakte.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI
- **Status:** Aktuell
