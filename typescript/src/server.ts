/**
 * Server - P2P HTTP Server
 * 
 * Serves account files over HTTP for peer synchronization.
 */

import type { Storage, ServerConfig, FileListItem } from './types/index.js';
import { SERVER_RESULT_LIMIT } from './types/index.js';
import type { Account } from './account.js';
import { File } from './file.js';
import { sign, serializePublicKey } from './crypto/index.js';

export interface ServerRequest {
  method: string;
  url: string;
  path: string;
  query: Record<string, string>;
  headers: Record<string, string>;
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

  constructor(account: Account, storage: Storage, config: ServerConfig = {}) {
    this.account = account;
    this.storage = storage;
    this.config = {
      resultsLimit: SERVER_RESULT_LIMIT,
      bootstrapNodes: [],
      enableMDNS: false,
      enableDHT: false,
      ...config,
    };
  }

  /**
   * Handle incoming HTTP request
   */
  async handleRequest(req: ServerRequest): Promise<ServerResponse> {
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
      return await this.handleFileVersion(resource);
    } else {
      // Get file: /p2p/<fingerprint>/<file>
      return await this.handleFile(resource);
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

    const files = await File.list(this.account, this.storage);
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
   * Handle file download request
   */
  private async handleFile(fileName: string): Promise<ServerResponse> {
    try {
      const file = File.create(this.account, this.storage, fileName);
      const data = await this.storage.readFile(file.getPath());

      return {
        status: 200,
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Length': data.length.toString(),
        },
        body: data,
      };
    } catch (err) {
      return this.notFound();
    }
  }

  /**
   * Handle file version download request
   */
  private async handleFileVersion(resource: string): Promise<ServerResponse> {
    const match = resource.match(/^(.+)\.versions\/([^/]+)$/);
    if (!match) {
      return this.notFound();
    }

    const [, fileName, versionHash] = match;

    try {
      const file = File.create(this.account, this.storage, fileName);
      const versionDir = `${file.getPath()}.versions`;
      const versionPath = this.storage.join(versionDir, versionHash);

      if (!(await this.storage.exists(versionPath))) {
        return this.notFound();
      }

      const data = await this.storage.readFile(versionPath);

      return {
        status: 200,
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Length': data.length.toString(),
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
   * Create Express.js middleware
   * 
   * Example:
   * ```
   * const app = express();
   * app.use(server.expressMiddleware());
   * ```
   */
  expressMiddleware() {
    return async (req: any, res: any, next: any) => {
      if (!req.path.startsWith('/p2p/')) {
        return next();
      }

      const request: ServerRequest = {
        method: req.method,
        url: req.url,
        path: req.path,
        query: req.query || {},
        headers: req.headers || {},
      };

      const response = await this.handleRequest(request);

      res.status(response.status);
      Object.entries(response.headers).forEach(([key, value]) => {
        res.setHeader(key, value);
      });

      if (response.body instanceof Uint8Array) {
        res.send(Buffer.from(response.body));
      } else {
        res.send(response.body);
      }
    };
  }

  /**
   * Create Node.js http.Server request handler
   * 
   * Example:
   * ```
   * const httpServer = http.createServer(server.nodeHandler());
   * ```
   */
  nodeHandler() {
    return async (req: any, res: any) => {
      const url = new URL(req.url!, `http://${req.headers.host}`);

      const query: Record<string, string> = {};
      url.searchParams.forEach((value, key) => {
        query[key] = value;
      });

      const request: ServerRequest = {
        method: req.method!,
        url: req.url!,
        path: url.pathname,
        query,
        headers: req.headers || {},
      };

      const response = await this.handleRequest(request);

      res.writeHead(response.status, response.headers);

      if (response.body instanceof Uint8Array) {
        res.end(Buffer.from(response.body));
      } else {
        res.end(response.body);
      }
    };
  }

  /**
   * Get server configuration
   */
  getConfig(): ServerConfig {
    return { ...this.config };
  }
}
