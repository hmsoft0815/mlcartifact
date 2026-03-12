import { createClient, Client, Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ArtifactService } from "./gen/artifact_connect.js";
import { WriteRequest, ReadRequest, ListRequest, DeleteRequest, WriteResponse, ReadResponse, ListResponse, DeleteResponse } from "./gen/artifact_pb.js";
import { WriteOptions, ReadOptions, ListOptions, DeleteOptions } from "./options.js";

/**
 * Universal Client for the mlcartifact service.
 * Works seamlessly in Browser, Node.js, and Edge environments.
 * 
 * The client uses the Connect protocol to communicate with the artifact server.
 * It automatically handles environment variables in Node.js environments.
 */
export class ArtifactClient {
  private client: Client<typeof ArtifactService>;

  /**
   * Creates a new ArtifactClient.
   * 
   * @param baseUrl - The base URL of the artifact server (e.g. 'http://localhost:9590').
   *                  If omitted, it looks for ARTIFACT_GRPC_ADDR env var.
   * @param transport - Optional custom transport. If provided, baseUrl is ignored.
   *                    Useful for adding interceptors or mocking in tests.
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
   * 
   * @param filename - The name of the file (e.g. 'result.json').
   * @param content - The data to store. Can be a string or a Uint8Array.
   * @param opts - Optional configuration (expiresHours, mimeType, userId, etc.).
   * @returns A promise resolving to the WriteResponse (includes artifact ID and URI).
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
   * 
   * @param idOrFilename - The unique ID or the filename of the artifact to retrieve.
   * @param opts - Optional configuration (userId).
   * @returns A promise resolving to the ReadResponse (includes content and mimeType).
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
   * Lists artifacts available in the store.
   * 
   * @param opts - Optional filters and pagination (limit, offset, userId, source).
   * @returns A promise resolving to the ListResponse containing an array of ArtifactInfo.
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
   * Deletes an artifact from the store.
   * This is a permanent operation and cannot be undone.
   * 
   * @param idOrFilename - The unique ID or the filename of the artifact to delete.
   * @param opts - Optional configuration (userId).
   * @returns A promise resolving to the DeleteResponse (indicates success/failure).
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
