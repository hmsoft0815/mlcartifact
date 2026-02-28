# Artifact MCP Server
**Autor: Michael Lechner**

Der Artifact MCP Server ist eine Kernkomponente des `wollmilchsau`-Ökosystems. Er bietet eine sichere, persistente und hochperformante Speicherschicht für Assets, die von LLMs (Large Language Models) generiert wurden.

---

## Warum ein Artifact Server?

In komplexen agentenbasierten Workflows stoßen LLMs oft an drei kritische Grenzen, die der Artifact Server löst:

1.  **Kontext-Überlastung:** LLMs sollten große generierte Berichte, Logs oder Diagramme nicht ständig in ihrem primären Gesprächskontext halten. Der Artifact Server erlaubt es dem LLM, Daten "auszulagern" und über eine stabile URI darauf zu verweisen.
2.  **Tool-übergreifender Status:** Verschiedene MCP-Tools (z. B. ein "Researcher" und ein "Chart Generator") müssen oft Daten teilen. Der Artifact Server fungiert als gemeinsame Zwischenablage für die gesamte Agenten-Suite.
3.  **Persistenz:** Im Gegensatz zum flüchtigen Arbeitsspeicher werden Artefakte auf der Festplatte gespeichert. Dies ermöglicht die Fortsetzung von Sitzungen und eine spätere Überprüfung der Ergebnisse.

## Features

- **Doppel-Schnittstelle:** Erreichbar über **MCP** (für LLMs) und **gRPC** (für schnelle interne Kommunikation zwischen Diensten).
- **Auto-Cleanup:** Eine intelligente Speicherrichtlinie (Standard 24h) verhindert, dass der Speicherplatz erschöpft wird.
- **Universeller Zugriff:** Schlägt die Brücke von der restriktiven V8-Umgebung (`wollmilchsau`) zum Dateisystem des Hosts.
- **Umfangreiche Metadaten:** Trackt Erstellungszeit, Quell-Server und Ablaufdatum für jede Datei.

---

## Tools für LLMs

### 📝 `write_artifact`
Speichert Inhalte (Text oder binäre Daten) im Speicher ab.
- `filename`: Gewünschter Dateiname (z. B. `analyse_bericht.md`).
- `content`: Die zu speichernden Daten.
- `description`: Optionale UTF-8 Beschreibung des Artefakts.
- `expires_in_hours`: Optionales Ablaufdatum in Stunden (Standard: 24).

### 📖 `read_artifact`
Ruft gespeicherte Inhalte ab.
- `id`: Eindeutige Artefakt-ID oder Dateiname.

### 📋 `list_artifacts`
Gibt eine Übersicht aller aktiven Artefakte zurück, inklusive Größen und Ablaufdaten.

### 🗑️ `delete_artifact`
Manuelles Löschen eines spezifischen Assets.

---

## Integrationsbeispiel: Wollmilchsau (TypeScript)

Der `artifact-server` bietet eine spezialisierte Bridge für die JavaScript/TypeScript-Runtime. Skripte können das globale `artifact`-Objekt nutzen:

```typescript
// Beispiel: Speichern eines generierten Diagramms
const svg = "<svg>...</svg>";
const res = await artifact.write("wachstums_chart.svg", svg, "image/svg+xml", 24, "Visualisierung des monatlichen Wachstums");

console.log(`Diagramm gespeichert! Zugriff über URI: ${res.uri}`);
```

## Interne Architektur

Der Server ist in **Go** geschrieben und nutzt `log/slog` für professionelles Monitoring. Er unterstützt mehrere Protokolle:
- **stdio:** Standard-MCP-Modus.
- **SSE (HTTP):** Für Remote- oder Web-Integration.
- **gRPC:** Binärer Hochleistungs-Transport für interne Clients.

---

## Build & Run

```bash
# Binary kompilieren
make build

# Im stdio-Modus starten (Standard für den Proxy)
./build/artifact-server
```

Copyright (c) 2026 **Michael Lechner**. Alle Rechte vorbehalten. Lizenziert unter der MIT-Lizenz.
