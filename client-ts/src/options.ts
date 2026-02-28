/**
 * Options for writing an artifact.
 */
export interface WriteOptions {
  /**
   * The MIME type of the content (e.g., 'text/markdown', 'image/png').
   * If not provided, the server will detect it from the filename extension.
   */
  mimeType?: string;

  /**
   * Hours until the artifact is automatically deleted. Default is 24.
   */
  expiresHours?: number;

  /**
   * The source identifier (e.g., 'sql-mcp-server').
   */
  source?: string;

  /**
   * Scope the artifact to a specific user.
   */
  userId?: string;

  /**
   * Arbitrary key-value metadata.
   */
  metadata?: Record<string, string>;

  /**
   * Human-readable description. (not part of the metadata -
   *  this should be used for end user display)
   */
  description?: string;
}

/**
 * Options for reading an artifact.
 */
export interface ReadOptions {
  /**
   * Scope the read to a specific user.
   */
  userId?: string;
}

/**
 * Options for listing artifacts.
 */
export interface ListOptions {
  /**
   * Scope the list to a specific user.
   */
  userId?: string;

  /**
   * Max items to return.
   */
  limit?: number;

  /**
   * Offset for pagination.
   */
  offset?: number;

  /**
   * Filter by source.
   */
  source?: string;
}

/**
 * Options for deleting an artifact.
 */
export interface DeleteOptions {
  /**
   * Scope the delete to a specific user.
   */
  userId?: string;
}
