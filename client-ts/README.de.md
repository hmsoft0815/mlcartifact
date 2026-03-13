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

```bash
npm install @hmsoft0815/mlcartifact-client
```

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

## API-Referenz

### new ArtifactClient(baseUrl?: string, transport?: Transport)

Erstellt einen neuen Client.
- baseUrl: Die URL des Artefakt-Servers. Standardmäßig process.env.ARTIFACT_GRPC_ADDR oder http://localhost:9590.
- transport: Optionaler benutzerdefinierter Connect-Transport.

### write(filename: string, content: string | Uint8Array, options?: WriteOptions)

Speichert ein Artefakt im Speicher.
- options.userId: Beschränkt das Artefakt auf einen bestimmten Benutzer.
- options.expiresHours: Anzahl der Stunden bis zur automatischen Löschung (Standard: 24).
- options.mimeType: Explizite Angabe des MIME-Typs.
- options.source: Identifiziert den Ersteller des Artefakts.

### read(idOrFilename: string, options?: ReadOptions)

Ruft ein Artefakt anhand der ID oder des ursprünglichen Dateinamens ab.

### list(options?: ListOptions)

Gibt eine Liste von Artefakten zurück.
- options.limit: Maximale Anzahl an Einträgen.
- options.offset: Offset für die Paginierung.
- options.userId: Filter nach Benutzer.

### delete(idOrFilename: string, options?: DeleteOptions)

Löscht ein Artefakt dauerhaft.

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
