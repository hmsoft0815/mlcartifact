import { createClient, type Client, type Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  ArtifactService,
  type WriteResponse,
  type ReadResponse,
  type ListResponse,
  type DeleteResponse,
  type PatchResponse,
} from "./gen/artifact_pb.js";
import type {
  WriteOptions,
  ReadOptions,
  ListOptions,
  DeleteOptions,
  PatchOptions,
  FindOptions,
} from "./options.js";

/** Content accepted by write() and patch(). */
export type ArtifactContent = Uint8Array | string | Blob;

/** Reads an environment variable in Node.js; returns undefined elsewhere (browser, edge). */
function env(name: string): string | undefined {
  return typeof process === "undefined" ? undefined : process.env?.[name];
}

async function toBytes(content: ArtifactContent): Promise<Uint8Array> {
  if (typeof content === "string") {
    return new TextEncoder().encode(content);
  }
  if (typeof Blob !== "undefined" && content instanceof Blob) {
    return new Uint8Array(await content.arrayBuffer());
  }
  return content as Uint8Array;
}

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

    const url = baseUrl || env("ARTIFACT_GRPC_ADDR") || "http://localhost:9590";
    // Ensure URL has protocol
    const finalUrl = url.includes("://") ? url : `http://${url}`;

    this.client = createClient(ArtifactService, createConnectTransport({ baseUrl: finalUrl }));
  }

  /**
   * Writes an artifact to the store.
   *
   * @param filename - The name of the file (e.g. 'result.json').
   * @param content - The data to store: string (UTF-8 encoded), Uint8Array or Blob.
   * @param opts - Optional configuration (virtualPath, expiresHours, mimeType, userId, ...).
   * @returns A promise resolving to the WriteResponse (includes artifact ID, URI and virtual path).
   */
  async write(filename: string, content: ArtifactContent, opts: WriteOptions = {}): Promise<WriteResponse> {
    return await this.client.write({
      filename,
      content: await toBytes(content),
      mimeType: opts.mimeType,
      expiresHours: opts.expiresHours,
      source: opts.source || env("ARTIFACT_SOURCE"),
      metadata: opts.metadata,
      userId: opts.userId || env("ARTIFACT_USER_ID"),
      description: opts.description,
      virtualPath: opts.virtualPath,
    });
  }

  /**
   * Reads an artifact from the store.
   *
   * @param idOrPath - The artifact ID, filename, or virtual path (starting with '/').
   * @param opts - Optional configuration (userId).
   * @returns A promise resolving to the ReadResponse (includes content and mimeType).
   */
  async read(idOrPath: string, opts: ReadOptions = {}): Promise<ReadResponse> {
    return await this.client.read({
      id: idOrPath,
      userId: opts.userId || env("ARTIFACT_USER_ID"),
    });
  }

  /**
   * Lists artifacts available in the store.
   *
   * If `dirPath` is set, the server switches to VFS directory listing mode and
   * returns the direct children of that virtual directory (sub-directories have
   * `isDirectory === true`).
   *
   * @param opts - Optional filters and pagination (dirPath, limit, offset, userId, source).
   * @returns A promise resolving to the ListResponse containing an array of ArtifactInfo.
   */
  async list(opts: ListOptions = {}): Promise<ListResponse> {
    return await this.client.list({
      source: opts.source,
      userId: opts.userId || env("ARTIFACT_USER_ID"),
      limit: opts.limit,
      offset: opts.offset,
      dirPath: opts.dirPath,
    });
  }

  /**
   * Deletes an artifact from the store.
   * This is a permanent operation and cannot be undone.
   *
   * @param idOrPath - The artifact ID, filename, or virtual path (starting with '/').
   * @param opts - Optional configuration (userId).
   * @returns A promise resolving to the DeleteResponse (indicates success/failure).
   */
  async delete(idOrPath: string, opts: DeleteOptions = {}): Promise<DeleteResponse> {
    return await this.client.delete({
      id: idOrPath,
      userId: opts.userId || env("ARTIFACT_USER_ID"),
    });
  }

  /**
   * Modifies the content of an existing artifact in place.
   *
   * - `append: true` appends `content` to the end of the artifact.
   * - Otherwise the 0-based line range `[lineStart, lineEnd)` is replaced by
   *   `content` (both default to 0, i.e. insert at the beginning).
   *
   * @param idOrPath - The artifact ID or virtual path.
   * @param content - The content to insert or append.
   * @param opts - Patch mode (append / lineStart / lineEnd) and userId.
   * @returns A promise resolving to the PatchResponse (success, new size).
   */
  async patch(idOrPath: string, content: ArtifactContent, opts: PatchOptions = {}): Promise<PatchResponse> {
    return await this.client.patch({
      id: idOrPath,
      userId: opts.userId || env("ARTIFACT_USER_ID"),
      content: await toBytes(content),
      lineStart: opts.lineStart,
      lineEnd: opts.lineEnd,
      append: opts.append,
    });
  }

  /**
   * Finds artifacts whose virtual path matches a glob pattern
   * (e.g. '/projects/*.md'); plain substrings match case-insensitively.
   *
   * @param pattern - Glob pattern or substring.
   * @param opts - Optional configuration (userId).
   * @returns A promise resolving to a ListResponse with the matching artifacts.
   */
  async find(pattern: string, opts: FindOptions = {}): Promise<ListResponse> {
    return await this.client.find({
      pattern,
      userId: opts.userId || env("ARTIFACT_USER_ID"),
    });
  }
}
