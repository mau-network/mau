/**
 * Server - P2P HTTP Server
 * 
 * Serves account files over HTTP for peer synchronization.
 */

import type { Storage, ServerConfig, FileListItem } from './types/index.js';
import { SERVER_RESULT_LIMIT } from './types/index.js';
import type { Account } from './account.js';
import { sign, serializePublicKey } from './crypto/index.js';
import type { KademliaDHT } from './network/dht.js';

export interface ServerRequest {
  method: string;
  url: string;
  path: string;
  query: Record<string, string>;
  headers: Record<string, string>;
  body?: string;
}

export interface ServerResponse {
  status: number;
  headers: Record<string, string>;
  body: Uint8Array | string;
}

/**
 * Server handles HTTP requests for file serving
 * 
 * This is a framework-agnostic implementation that can be
 * integrated with any HTTP server (Express, http.createServer, etc.)
 */
export class Server {
  private account: Account;
  private storage: Storage;
  private config: ServerConfig;
  private dht?: KademliaDHT;

  constructor(account: Account, storage: Storage, config: ServerConfig = {}, dht?: KademliaDHT) {
    this.account = account;
    this.storage = storage;
    this.config = {
      resultsLimit: SERVER_RESULT_LIMIT,
      bootstrapNodes: [],
      enableDHT: false,
      ...config,
    };
    this.dht = dht;
  }

  /**
   * Handle incoming HTTP request
   */
  async handleRequest(req: ServerRequest): Promise<ServerResponse> {
    // DHT WebRTC offer endpoint — must be checked before the fingerprint routes
    if (req.method === 'POST' && req.path === '/p2p/dht/offer') {
      return this.handleDHTOffer(req);
    }

    if (req.method !== 'GET') {
      return this.methodNotAllowed();
    }

    // Parse path
    const pathMatch = req.path.match(/^\/p2p\/([0-9a-f]+)(\/(.+))?$/);
    if (!pathMatch) {
      return this.notFound();
    }

    const fingerprint = pathMatch[1];
    const resource = pathMatch[3];

    // Verify fingerprint matches account
    if (fingerprint !== this.account.getFingerprint()) {
      return this.notFound();
    }

    // Route request
    if (!resource) {
      // List files: /p2p/<fingerprint>
      return await this.handleFileList(req);
    } else if (resource === 'auth') {
      // mTLS challenge: /p2p/<fingerprint>/auth?challenge=<hex>
      return await this.handleAuth(req);
    } else if (resource.includes('.versions/')) {
      // Get version: /p2p/<fingerprint>/<file>.versions/<hash>
      return await this.handleFileVersion(resource, req);
    } else {
      // Get file: /p2p/<fingerprint>/<file>
      return await this.handleFile(resource, req);
    }
  }

  /**
   * Handle file list request
   */
  private async handleFileList(req: ServerRequest): Promise<ServerResponse> {
    // Parse If-Modified-Since header for incremental sync
    let modifiedSince = 0;
    const ifModifiedSinceHeader = req.headers['if-modified-since'];
    if (ifModifiedSinceHeader) {
      const parsed = Date.parse(ifModifiedSinceHeader);
      if (!isNaN(parsed)) {
        modifiedSince = parsed;
      }
    }

    const files = await this.account.listFiles();
    const fileList: FileListItem[] = [];

    for (const file of files) {
      // Check modification time if filtering is requested
      if (modifiedSince > 0) {
        const stats = await this.storage.stat(file.getPath());
        if (stats.modifiedTime && stats.modifiedTime <= modifiedSince) {
          continue; // Skip files not modified since the specified time
        }
      }

      const [sum, size] = await Promise.all([file.getChecksum(), file.getSize()]);

      fileList.push({
        path: file.getName(),
        size,
        sum,
      });

      // Limit results
      if (fileList.length >= (this.config.resultsLimit || SERVER_RESULT_LIMIT)) {
        break;
      }
    }

    return {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ files: fileList }),
    };
  }

  /**
   * Handle DHT WebRTC offer: POST /p2p/dht/offer
   * Body: { from: Fingerprint, offer: RTCSessionDescriptionInit }
   * Response: { answer: RTCSessionDescriptionInit }
   */
  private async handleDHTOffer(req: ServerRequest): Promise<ServerResponse> {
    if (!this.dht) {
      return { status: 404, headers: { 'Content-Type': 'text/plain' }, body: 'DHT not enabled' };
    }
    try {
      const { from, offer } = JSON.parse(req.body ?? '{}') as {
        from: string;
        offer: RTCSessionDescriptionInit;
      };
      if (!from || !offer) {
        return { status: 400, headers: { 'Content-Type': 'text/plain' }, body: 'Bad Request' };
      }
      const remoteAddr = req.headers['x-forwarded-for'] ?? req.headers['host'] ?? '';
      const answer = await this.dht.handleHTTPOffer(from, offer, remoteAddr);
      return {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ answer }),
      };
    } catch {
      return { status: 500, headers: { 'Content-Type': 'text/plain' }, body: 'Internal Server Error' };
    }
  }

  /**
   * Handle mTLS challenge-response: GET /p2p/<fingerprint>/auth?challenge=<hex>
   *
   * Signs the caller's challenge with the account private key so the client can
   * verify this server controls the key for the advertised fingerprint.
   */
  private async handleAuth(req: ServerRequest): Promise<ServerResponse> {
    const challengeHex = req.query['challenge'];
    if (!challengeHex || !/^[0-9a-f]+$/i.test(challengeHex) || challengeHex.length % 2 !== 0) {
      return {
        status: 400,
        headers: { 'Content-Type': 'text/plain' },
        body: 'Bad Request: missing or invalid challenge',
      };
    }

    const challenge = new Uint8Array(
      (challengeHex.match(/.{2}/g) as string[]).map(b => parseInt(b, 16))
    );

    const signature = await sign(challenge, this.account.getPrivateKey());
    const publicKey = serializePublicKey(this.account.getPublicKeyObject());

    return {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ publicKey, signature }),
    };
  }

  /**
   * Handle file download request with Range support for resumable downloads
   */
  private async handleFile(fileName: string, req?: ServerRequest): Promise<ServerResponse> {
    try {
      const { validateFileName } = await import('./crypto/index.js');
      const { InvalidFileNameError } = await import('./types/index.js');
      
      if (!validateFileName(fileName)) {
        throw new InvalidFileNameError('contains invalid characters or path separators');
      }

      const contentDir = this.account.getContentDir();
      const filePath = this.storage.join(contentDir, fileName);
      const data = await this.storage.readFile(filePath);
      const totalSize = data.length;

      // Parse Range header
      const rangeHeader = req?.headers['range'];
      if (rangeHeader && rangeHeader.startsWith('bytes=')) {
        const rangeMatch = rangeHeader.match(/bytes=(\d+)-(\d*)/);
        if (rangeMatch) {
          const start = parseInt(rangeMatch[1], 10);
          const end = rangeMatch[2] ? parseInt(rangeMatch[2], 10) : totalSize - 1;
          
          if (start >= 0 && start < totalSize && end >= start && end < totalSize) {
            const rangeData = data.slice(start, end + 1);
            return {
              status: 206,
              headers: {
                'Content-Type': 'application/octet-stream',
                'Content-Length': rangeData.length.toString(),
                'Content-Range': `bytes ${start}-${end}/${totalSize}`,
                'Accept-Ranges': 'bytes',
              },
              body: rangeData,
            };
          }
        }
      }

      return {
        status: 200,
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Length': data.length.toString(),
          'Accept-Ranges': 'bytes',
        },
        body: data,
      };
    } catch (err) {
      return this.notFound();
    }
  }

  /**
   * Handle file version download request with Range support
   */
  private async handleFileVersion(resource: string, req?: ServerRequest): Promise<ServerResponse> {
    const match = resource.match(/^(.+)\.versions\/([^/]+)$/);
    if (!match) {
      return this.notFound();
    }

    const [, fileName, versionHash] = match;

    try {
      const { validateFileName } = await import('./crypto/index.js');
      const { InvalidFileNameError } = await import('./types/index.js');
      
      if (!validateFileName(fileName)) {
        throw new InvalidFileNameError('contains invalid characters or path separators');
      }

      const contentDir = this.account.getContentDir();
      const filePath = this.storage.join(contentDir, fileName);
      const versionDir = `${filePath}.versions`;
      const versionPath = this.storage.join(versionDir, `${versionHash}.pgp`);

      if (!(await this.storage.exists(versionPath))) {
        return this.notFound();
      }

      const data = await this.storage.readFile(versionPath);
      const totalSize = data.length;

      // Parse Range header
      const rangeHeader = req?.headers['range'];
      if (rangeHeader && rangeHeader.startsWith('bytes=')) {
        const rangeMatch = rangeHeader.match(/bytes=(\d+)-(\d*)/);
        if (rangeMatch) {
          const start = parseInt(rangeMatch[1], 10);
          const end = rangeMatch[2] ? parseInt(rangeMatch[2], 10) : totalSize - 1;
          
          if (start >= 0 && start < totalSize && end >= start && end < totalSize) {
            const rangeData = data.slice(start, end + 1);
            return {
              status: 206,
              headers: {
                'Content-Type': 'application/octet-stream',
                'Content-Length': rangeData.length.toString(),
                'Content-Range': `bytes ${start}-${end}/${totalSize}`,
                'Accept-Ranges': 'bytes',
              },
              body: rangeData,
            };
          }
        }
      }

      return {
        status: 200,
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Length': data.length.toString(),
          'Accept-Ranges': 'bytes',
        },
        body: data,
      };
    } catch (err) {
      return this.notFound();
    }
  }

  private methodNotAllowed(): ServerResponse {
    return {
      status: 405,
      headers: { 'Content-Type': 'text/plain' },
      body: 'Method Not Allowed',
    };
  }

  private notFound(): ServerResponse {
    return {
      status: 404,
      headers: { 'Content-Type': 'text/plain' },
      body: 'Not Found',
    };
  }

  /**
   * Get server configuration
   */
  getConfig(): ServerConfig {
    return { ...this.config };
  }
}
