// without real mcp rpc test for now..
import { describe, it, expect } from 'vitest';
import { createRouterTransport } from "@connectrpc/connect";
import { ArtifactClient } from './client.js';
import { ArtifactService } from "./gen/artifact_connect.js";
import { WriteResponse, ReadResponse } from "./gen/artifact_pb.js";

describe('ArtifactClient', () => {
  it('should write an artifact successfully through a mocked transport', async () => {
    // Create a mock implementation of the service
    const mockTransport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        async write(req) {
          return new WriteResponse({
            id: 'mock-id-123',
            filename: req.filename,
            uri: `artifact://${req.filename}`,
            expiresAt: new Date().toISOString(),
          });
        },
      });
    });

    const client = new ArtifactClient('http://localhost:9590', mockTransport);
    
    const resp = await client.write('test.txt', 'Hello Mock!');
    
    expect(resp.id).toBe('mock-id-123');
    expect(resp.uri).toBe('artifact://test.txt');
  });

  it('should read an artifact successfully through a mocked transport', async () => {
    const mockTransport = createRouterTransport(({ service }) => {
      service(ArtifactService, {
        async read(req) {
          return new ReadResponse({
            content: new TextEncoder().encode('Mocked content'),
            mimeType: 'text/plain',
            filename: 'mock.txt',
          });
        },
      });
    });

    const client = new ArtifactClient('http://localhost:9590', mockTransport);
    
    const resp = await client.read('some-id');
    
    expect(new TextDecoder().decode(resp.content)).toBe('Mocked content');
    expect(resp.mimeType).toBe('text/plain');
  });
});
