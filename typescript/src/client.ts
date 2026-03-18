/**
 * Client - P2P HTTP Client
 * 
 * Handles HTTP communication with peers for file synchronization.
 */

import pRetry from 'p-retry';
import type {
  Storage,
  Fingerprint,
  ClientConfig,
  FileListItem,
  FileListResponse,
} from './types/index.js';
import { HttpError, NetworkError, MauError } from './types/index.js';
import { HTTP_TIMEOUT_MS, URI_PROTOCOL_NAME, PeerNotFoundError } from './types/index.js';
import type { Account } from './account.js';
import { deserializePublicKey, getFingerprint, verify, normalizeFingerprint } from './crypto/index.js';

/**
 * P2P HTTP Client for file synchronization
 * 
 * Handles communication with peers over HTTP to download files and track sync state.
 * Supports peer discovery via resolvers and automatic retry with exponential backoff.
 * 
 * @example
 * ```typescript
 * const client = account.createClient(peer, [staticResolver(peers)]);
 * const stats = await client.sync();
 * console.log(`Downloaded ${stats.downloaded} files`);
 * ```
 */
export class Client {
  private account: Account;
  private storage: Storage;
  private peer: Fingerprint;
  private config: ClientConfig;
  private fetchImpl: typeof fetch;
  private resolvedAddress: string | null = null;
  private authenticated = false;

  constructor(
    account: Account,
    storage: Storage,
    peer: Fingerprint,
    config: ClientConfig = {}
  ) {
    this.account = account;
    this.storage = storage;
    this.peer = peer;
    this.config = {
      timeout: HTTP_TIMEOUT_MS,
      dnsNames: [],
      resolvers: [],
      ...config,
    };

    // Use provided fetch, global fetch, or node-fetch
    this.fetchImpl = config.fetchImpl ?? (typeof fetch !== 'undefined' ? fetch : this.getNodeFetch());
  }

  private getNodeFetch(): typeof fetch {
    try {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      return require('node-fetch');
    } catch {
      throw new NetworkError('fetch not available - install node-fetch for Node.js environments');
    }
  }

  /**
   * Resolve address once, pin the TLS certificate, run the mTLS handshake,
   * then cache all results.
   */
  private async ensureReady(): Promise<string> {
    if (!this.resolvedAddress) {
      this.resolvedAddress = await this.resolvePeerAddress();
    }
    if (!this.authenticated) {
      await this.performHandshake(this.resolvedAddress);
    }
    return this.resolvedAddress;
  }

  /**
   * PGP challenge-response handshake: verify the server controls the private key
   * that corresponds to the expected peer fingerprint.
   */
  private async performHandshake(address: string): Promise<void> {
    const challenge = crypto.getRandomValues(new Uint8Array(32));
    const challengeHex = Array.from(challenge)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');

    const url = `${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}/auth?challenge=${challengeHex}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchImpl(url, { signal: controller.signal });
      if (!response.ok) {
        throw new HttpError(response.status, response.statusText);
      }

      const data = await response.json() as { publicKey: string; signature: string };
      const peerKey = await deserializePublicKey(data.publicKey);
      const receivedFpr = normalizeFingerprint(getFingerprint(peerKey));
      const expectedFpr = normalizeFingerprint(this.peer);

      if (receivedFpr !== expectedFpr) {
        throw new MauError(
          `Peer fingerprint mismatch: expected ${expectedFpr}, got ${receivedFpr}`,
          'PEER_FINGERPRINT_MISMATCH'
        );
      }

      const valid = await verify(challenge, data.signature, peerKey);
      if (!valid) {
        throw new MauError('Peer signature verification failed during mTLS handshake', 'PEER_AUTH_FAILED');
      }

      this.authenticated = true;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Resolve peer address using configured resolvers
   */
  private async resolvePeerAddress(): Promise<string> {
    if (!this.config.resolvers || this.config.resolvers.length === 0) {
      throw new PeerNotFoundError();
    }

    const timeout = this.config.timeout || HTTP_TIMEOUT_MS;

    // Try all resolvers concurrently
    const results = await Promise.allSettled(
      this.config.resolvers.map((resolver) => resolver(this.peer, timeout))
    );

    for (const result of results) {
      if (result.status === 'fulfilled' && result.value) {
        return result.value;
      }
    }

    throw new PeerNotFoundError();
  }

  /**
   * Fetch with retry and exponential backoff
   * Retries on network errors and 5xx server errors
   */
  private async fetchWithRetry(
    url: string,
    options: RequestInit,
    maxRetries = 2
  ): Promise<Response> {
    return pRetry(
      async () => {
        const response = await this.fetchImpl(url, options);
        if (response.status >= 500) {throw new HttpError(response.status, response.statusText);}
        return response;
      },
      {
        retries: maxRetries,
        minTimeout: 100,
        factor: 2,
        onFailedAttempt: (error: Error): void => {
          if (error.name === 'AbortError') {throw error;}
        },
      }
    );
  }

  /**
   * Fetch file list from peer
   */
  async fetchFileList(after?: Date): Promise<FileListItem[]> {
    const address = await this.ensureReady();
    const url = new URL(`${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}`);

    if (after) {
      url.searchParams.set('after', after.toISOString());
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchWithRetry(url.toString(), {
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new HttpError(response.status, response.statusText);
      }

      const data: FileListResponse = await response.json();
      return data.files || [];
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Download a file from peer
   */
  async downloadFile(fileName: string): Promise<Uint8Array> {
    const address = await this.ensureReady();
    const url = `${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}/${encodeURIComponent(
      fileName
    )}`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchWithRetry(url, {
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new HttpError(response.status, response.statusText);
      }

      const buffer = await response.arrayBuffer();
      return new Uint8Array(buffer);
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Download a specific file version
   */
  async downloadFileVersion(fileName: string, version: string): Promise<Uint8Array> {
    const address = await this.ensureReady();
    const url = `${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}/${encodeURIComponent(
      fileName
    )}.versions/${encodeURIComponent(version)}`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchWithRetry(url, {
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new HttpError(response.status, response.statusText);
      }

      const buffer = await response.arrayBuffer();
      return new Uint8Array(buffer);
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Sync files from peer
   */
  async sync(): Promise<{ downloaded: number; updated: number; errors: number; failedFiles: string[] }> {
    const stats = { downloaded: 0, updated: 0, errors: 0, failedFiles: [] as string[] };

    // Get last sync time
    const syncState = await this.account.getSyncState();
    const lastSync = syncState[this.peer] || 0;
    const afterDate = new Date(lastSync);

    try {
      // Fetch file list
      const fileList = await this.fetchFileList(afterDate);

      // Download each file
      for (const fileInfo of fileList) {
        try {
          const data = await this.withRetry(() => this.downloadFile(fileInfo.path));

          // Save to friend's content directory
          const friendDir = this.account.getFriendContentDir(this.peer);
          const filePath = this.storage.join(friendDir, fileInfo.path);

          // Check if file exists and is different
          let isNew = true;
          if (await this.storage.exists(filePath)) {
            const existingData = await this.storage.readFile(filePath);
            const existingSum = await this.calculateChecksum(existingData);
            if (existingSum === fileInfo.sum) {
              continue; // File unchanged
            }
            isNew = false;
          }

          await this.storage.writeFile(filePath, data);

          if (isNew) {
            stats.downloaded++;
          } else {
            stats.updated++;
          }
        } catch (err) {
          console.error(`Failed to download ${fileInfo.path}:`, err);
          stats.errors++;
          stats.failedFiles.push(fileInfo.path);
        }
      }

      if (stats.failedFiles.length > 0) {
        console.warn(`Sync completed with ${stats.failedFiles.length} failed file(s):`, stats.failedFiles);
      }

      // Update sync state
      await this.account.updateSyncState(this.peer, Date.now());
    } catch (err) {
      console.error(`Sync failed for ${this.peer}:`, err);
      throw err;
    }

    return stats;
  }

  /**
   * Retry an async operation with exponential backoff.
   * Aborts immediately on AbortError or 4xx HTTP errors.
   */
  private async withRetry<T>(
    fn: () => Promise<T>,
    maxRetries = 2,
    initialDelayMs = 200
  ): Promise<T> {
    return pRetry(fn, {
      retries: maxRetries,
      minTimeout: initialDelayMs,
      factor: 2,
      onFailedAttempt: (error: Error): void => {
        if (error.name === 'AbortError' || (error instanceof HttpError && error.statusCode < 500)) {
          throw error;
        }
      },
    });
  }

  private async calculateChecksum(data: Uint8Array): Promise<string> {
    const { sha256 } = await import('./crypto/index.js');
    return await sha256(data);
  }
}
