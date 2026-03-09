/**
 * Client - P2P HTTP Client
 * 
 * Handles HTTP communication with peers for file synchronization.
 */

import type {
  Storage,
  Fingerprint,
  ClientConfig,
  FileListItem,
  FileListResponse,
  Peer,
  FingerprintResolver,
} from './types/index.js';
import { HTTP_TIMEOUT_MS, URI_PROTOCOL_NAME, PeerNotFoundError } from './types/index.js';
import type { Account } from './account.js';
import { File } from './file.js';

export class Client {
  private account: Account;
  private storage: Storage;
  private peer: Fingerprint;
  private config: ClientConfig;
  private fetchImpl: typeof fetch;

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

    // Use global fetch or node-fetch
    this.fetchImpl = typeof fetch !== 'undefined' ? fetch : this.getNodeFetch();
  }

  private getNodeFetch(): typeof fetch {
    try {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      return require('node-fetch');
    } catch {
      throw new Error('fetch not available - install node-fetch for Node.js environments');
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
   * Fetch file list from peer
   */
  async fetchFileList(after?: Date): Promise<FileListItem[]> {
    const address = await this.resolvePeerAddress();
    const url = new URL(`${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}`);

    if (after) {
      url.searchParams.set('after', after.toISOString());
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchImpl(url.toString(), {
        signal: controller.signal,
        // Note: mTLS authentication would be configured here in a full implementation
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
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
    const address = await this.resolvePeerAddress();
    const url = `${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}/${encodeURIComponent(
      fileName
    )}`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchImpl(url, {
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
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
    const address = await this.resolvePeerAddress();
    const url = `${URI_PROTOCOL_NAME}://${address}/p2p/${this.peer}/${encodeURIComponent(
      fileName
    )}.versions/${encodeURIComponent(version)}`;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await this.fetchImpl(url, {
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
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
  async sync(): Promise<{ downloaded: number; updated: number; errors: number }> {
    const stats = { downloaded: 0, updated: 0, errors: 0 };

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
          const data = await this.downloadFile(fileInfo.path);

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
        }
      }

      // Update sync state
      await this.account.updateSyncState(this.peer, Date.now());
    } catch (err) {
      console.error(`Sync failed for ${this.peer}:`, err);
      throw err;
    }

    return stats;
  }

  private async calculateChecksum(data: Uint8Array): Promise<string> {
    const { sha256 } = await import('./crypto/index.js');
    return await sha256(data);
  }

  /**
   * Create a client for a specific peer
   */
  static create(
    account: Account,
    storage: Storage,
    peer: Peer,
    resolvers: FingerprintResolver[] = []
  ): Client {
    return new Client(account, storage, peer.fingerprint, {
      resolvers: [
        // Add static address resolver
        async (fingerprint) => {
          if (fingerprint === peer.fingerprint) {
            return peer.address;
          }
          return null;
        },
        ...resolvers,
      ],
    });
  }
}
