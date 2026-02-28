import { createClient, Client, Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ArtifactService } from "./gen/artifact_connect.js";
import { WriteRequest, ReadRequest, ListRequest, DeleteRequest, WriteResponse, ReadResponse, ListResponse, DeleteResponse } from "./gen/artifact_pb.js";
import { WriteOptions, ReadOptions, ListOptions, DeleteOptions } from "./options.js";

/**
 * Universal Client for the mlcartifact service.
 * Works in Browser, Node.js, and Edge environments.
 */
export class ArtifactClient {
  private client: Client<typeof ArtifactService>;

  /**
   * Creates a new ArtifactClient.
   * @param baseUrl The base URL of the artifact server (e.g. 'http://localhost:9590').
   * @param transport Optional custom transport (useful for testing/mocking).
   */
  constructor(baseUrl?: string, transport?: Transport) {
    if (transport) {
      this.client = createClient(ArtifactService, transport);
      return;
    }

    // Graceful handling for non-Node environments
    const envAddr = typeof process === 'undefined' ? undefined : process.env?.ARTIFACT_GRPC_ADDR;
    const url = baseUrl || envAddr || 'http://localhost:9590';
    
    // Ensure URL has protocol
    const finalUrl = url.includes('://') ? url : `http://${url}`;

    const defaultTransport = createConnectTransport({
      baseUrl: finalUrl,
    });

    this.client = createClient(ArtifactService, defaultTransport);
  }

  /**
   * Writes an artifact to the store.
   */
  async write(filename: string, content: Uint8Array | string, opts: WriteOptions = {}): Promise<WriteResponse> {
    const envSource = typeof process === 'undefined' ? undefined : process.env?.ARTIFACT_SOURCE;
    const envUser = typeof process === 'undefined' ? undefined : process.env?.ARTIFACT_USER_ID;

    const req = new WriteRequest({
      filename,
      // have to use as Uint8Array<ArrayBuffer> because of the type
      //  definition in the generated code (UTF8String is not supported)
      content: (typeof content === 'string' ? 
        new TextEncoder().encode(content) 
        : content) as Uint8Array<ArrayBuffer>,
      mimeType: opts.mimeType,
      expiresHours: opts.expiresHours,
      source: opts.source || envSource,
      metadata: opts.metadata,
      userId: opts.userId || envUser,
      description: opts.description,
    });

    return await this.client.write(req);
  }

  /**
   * Reads an artifact from the store.
   */
  async read(idOrFilename: string, opts: ReadOptions = {}): Promise<ReadResponse> {
    const envUser = typeof process === 'undefined' ? undefined : process.env?.ARTIFACT_USER_ID;

    const req = new ReadRequest({
      id: idOrFilename,
      userId: opts.userId || envUser,
    });

    return await this.client.read(req);
  }

  /**
   * Lists artifacts.
   */
  async list(opts: ListOptions = {}): Promise<ListResponse> {
    const envUser = typeof process === 'undefined' ? undefined : process.env?.ARTIFACT_USER_ID;

    const req = new ListRequest({
      source: opts.source,
      userId: opts.userId || envUser,
      limit: opts.limit,
      offset: opts.offset,
    });

    return await this.client.list(req);
  }

  /**
   * Deletes an artifact.
   * This is a permanent operation.
   */
  async delete(idOrFilename: string, opts: DeleteOptions = {}): Promise<DeleteResponse> {
    const envUser = typeof process === 'undefined' ? undefined : process.env?.ARTIFACT_USER_ID;

    const req = new DeleteRequest({
      id: idOrFilename,
      userId: opts.userId || envUser,
    });

    return await this.client.delete(req);
  }
}
