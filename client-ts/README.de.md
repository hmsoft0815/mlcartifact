# @hmsoft0815/mlcartifact-client

Ein universeller, sauberer und vollständig typisierter TypeScript-Client für den mlcartifact Dienst.

## Übersicht

Der mlcartifact-Dienst bietet ein gemeinsames Speicher-Backend für KI-Agenten und Tools. Mit diesem TypeScript-Client können Sie problemlos in jeder Umgebung (Browser, Node.js, Deno, Bun oder Edge-Funktionen) mit dem Dienst interagieren.

Er verwendet das [Connect](https://connectrpc.com/)-Protokoll, eine schlanke, typsichere Alternative zu traditionellem gRPC, die nahtlos über Standard-HTTP/1.1 oder HTTP/2 funktioniert.

## Features

- **Universell:** Funktioniert überall dort, wo fetch verfügbar ist.
- **Vollständig typisiert:** Alle Anfragen und Antworten sind über Protobuf streng typisiert.
- **Leichtgewichtig:** Minimale Abhängigkeiten, optimiert für moderne Umgebungen.
- **Connect-Protokoll:** Web-freundlich, keine komplexen gRPC-Web-Proxys erforderlich.

## Installation

Das Paket ist **nicht auf npm veröffentlicht**. Installation aus einem Checkout
des Repositories (der Build benötigt `../proto`, also das ganze Repo):

```bash
git clone https://github.com/hmsoft0815/mlcartifact.git
cd mlcartifact/client-ts
npm ci && npm run build        # erzeugt src/gen via buf, kompiliert nach dist/

# im eigenen Projekt: lokalen Build einbinden ...
npm install /pfad/zu/mlcartifact/client-ts
# ... oder ein Tarball erzeugen und dieses installieren
npm pack                       # -> hmsoft0815-mlcartifact-client-<version>.tgz
```

Benötigt Node.js >= 20 (oder eine Laufzeit mit `fetch`). Die Code-Generierung
nutzt die Dev-Abhängigkeit `@bufbuild/buf`; ein System-`protoc` ist nicht nötig.

## Schnellstart

```typescript
import { ArtifactClient } from '@hmsoft0815/mlcartifact-client';

async function example() {
  // baseUrl verwendet standardmäßig ARTIFACT_GRPC_ADDR oder 'http://localhost:9590'
  const client = new ArtifactClient();

  // 1. Artefakt schreiben
  // Unterstützt String, Uint8Array oder Blob
  const writeResp = await client.write('hello.md', '# Hello World', {
    description: 'Mein erstes Artefakt',
    mimeType: 'text/markdown',
    expiresHours: 48,
    metadata: {
      category: 'testing',
      importance: 'high'
    }
  });

  console.log(`Artefakt erstellt mit ID: ${writeResp.id}`);

  // 2. Artefakt lesen
  const readResp = await client.read(writeResp.id);
  const text = new TextDecoder().decode(readResp.content);
  console.log(`Inhalt: ${text}`);

  // 3. Artefakte auflisten
  const listResp = await client.list({ 
    limit: 5,
    offset: 0
  });
  
  for (const item of listResp.items) {
    console.log(`- ${item.filename} (ID: ${item.id})`);
  }

  // 4. Artefakt löschen
  await client.delete(writeResp.id);
}
```

### Virtuelles Dateisystem (VFS)

```typescript
// unter einem virtuellen Pfad schreiben
await client.write('readme.md', '# Alpha', { virtualPath: '/projects/alpha/readme.md' });

// read / delete / patch akzeptieren den virtuellen Pfad statt der ID
await client.patch('/projects/alpha/readme.md', '\nmore text', { append: true });
await client.patch('/projects/alpha/readme.md', '# Alpha v2', { lineStart: 0, lineEnd: 1 });

// Verzeichnisliste (Unterverzeichnisse haben isDirectory === true)
const dir = await client.list({ dirPath: '/projects' });

// Glob-/Teilstring-Suche über virtuelle Pfade
const found = await client.find('/projects/*/readme.md');
```

## API-Referenz

### new ArtifactClient(baseUrl?: string, transport?: Transport)

Erstellt einen neuen Client.
- baseUrl: Die URL des Artefakt-Servers. Standardmäßig process.env.ARTIFACT_GRPC_ADDR oder http://localhost:9590.
- transport: Optionaler benutzerdefinierter Connect-Transport.

### write(filename: string, content: string | Uint8Array | Blob, options?: WriteOptions)

Speichert ein Artefakt im Speicher. Strings werden UTF-8-kodiert; `Blob` funktioniert im Browser und ab Node.js 18.
- options.virtualPath: Optionaler VFS-Pfad, z. B. `/projects/alpha/readme.md`.
- options.userId: Beschränkt das Artefakt auf einen bestimmten Benutzer.
- options.expiresHours: Anzahl der Stunden bis zur automatischen Löschung (Standard: 24).
- options.mimeType: Explizite Angabe des MIME-Typs.
- options.source: Identifiziert den Ersteller des Artefakts.

### read(idOrPath: string, options?: ReadOptions)

Ruft ein Artefakt anhand der ID, des ursprünglichen Dateinamens oder des virtuellen Pfads (beginnt mit `/`) ab.

### list(options?: ListOptions)

Gibt eine Liste von Artefakten zurück.
- options.limit: Maximale Anzahl an Einträgen.
- options.offset: Offset für die Paginierung.
- options.userId: Filter nach Benutzer.
- options.source: Filter nach Quelle.
- options.dirPath: VFS-Verzeichnismodus — liefert die direkten Einträge dieses Verzeichnisses.

### delete(idOrPath: string, options?: DeleteOptions)

Löscht ein Artefakt dauerhaft (per ID, Dateiname oder virtuellem Pfad).

### patch(idOrPath: string, content: string | Uint8Array | Blob, options?: PatchOptions)

Ändert ein Artefakt direkt.
- options.append: `content` ans Ende anhängen.
- options.lineStart / options.lineEnd: Sonst wird der 0-basierte Zeilenbereich `[lineStart, lineEnd)` durch `content` ersetzt (Standard: 0 / lineStart, also Einfügen).
- options.userId: Auf einen Benutzer beschränken.

### find(pattern: string, options?: FindOptions)

Sucht Artefakte, deren virtueller Pfad auf ein Glob-Muster passt (einfache Teilstrings ohne Beachtung der Groß-/Kleinschreibung). Liefert eine `ListResponse`.

## Umgebungsvariablen (Node.js)

Der Client erkennt automatisch diese Variablen:

- ARTIFACT_GRPC_ADDR: Server-URL (z. B. https://api.artifacts.local).
- ARTIFACT_USER_ID: Standard-Benutzer-ID für alle Operationen.
- ARTIFACT_SOURCE: Standard-Quell-Tag für Schreibvorgänge.

## Fortgeschritten: Eigener Transport

Falls Sie benutzerdefinierte Header (wie Authentifizierungs-Token) zu jeder Anfrage hinzufügen müssen:

```typescript
import { createConnectTransport } from "@connectrpc/connect-web";
import { ArtifactClient } from "@hmsoft0815/mlcartifact-client";

const transport = createConnectTransport({
  baseUrl: "http://localhost:9590",
  interceptors: [
    (next) => async (req) => {
      req.header.set("Authorization", "Bearer my-token");
      return await next(req);
    },
  ],
});

const client = new ArtifactClient(undefined, transport);
```

## Lizenz

MIT - Copyright (c) 2026 Michael Lechner
