/**
 * HTTP Signaling Server
 * 
 * Simple HTTP server for WebRTC signaling (offer/answer/ICE exchange)
 * Can be deployed as a standalone service or integrated into existing backend.
 */

import http from 'http';
import type { Fingerprint } from '../src/types/index.js';

interface SignalingMessage {
  from: Fingerprint;
  to: Fingerprint;
  type: 'offer' | 'answer' | 'ice-candidate';
  data: any;
  timestamp: number;
}

/**
 * In-memory message queue
 * In production, use Redis or similar
 */
class MessageQueue {
  private messages: Map<Fingerprint, SignalingMessage[]> = new Map();
  private maxAge = 60000; // 1 minute

  post(message: SignalingMessage): void {
    const key = message.to;
    const messages = this.messages.get(key) || [];
    messages.push({
      ...message,
      timestamp: Date.now(),
    });
    this.messages.set(key, messages);
  }

  poll(fingerprint: Fingerprint): SignalingMessage[] {
    const messages = this.messages.get(fingerprint) || [];
    this.messages.delete(fingerprint);

    // Filter out old messages
    const now = Date.now();
    return messages.filter((m) => now - m.timestamp < this.maxAge);
  }

  cleanup(): void {
    const now = Date.now();
    for (const [key, messages] of this.messages.entries()) {
      const filtered = messages.filter((m) => now - m.timestamp < this.maxAge);
      if (filtered.length === 0) {
        this.messages.delete(key);
      } else {
        this.messages.set(key, filtered);
      }
    }
  }
}

/**
 * HTTP Signaling Server
 */
export class HTTPSignalingServer {
  private queue: MessageQueue;
  private server: http.Server | null = null;
  private cleanupInterval: NodeJS.Timeout | null = null;

  constructor() {
    this.queue = new MessageQueue();
  }

  /**
   * Start HTTP server
   */
  start(port: number = 8080): Promise<void> {
    return new Promise((resolve) => {
      this.server = http.createServer(this.handleRequest.bind(this));

      this.server.listen(port, () => {
        console.log(`Signaling server listening on http://localhost:${port}`);
        resolve();
      });

      // Cleanup old messages every 30 seconds
      this.cleanupInterval = setInterval(() => {
        this.queue.cleanup();
      }, 30000);
    });
  }

  /**
   * Stop server
   */
  stop(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.cleanupInterval) {
        clearInterval(this.cleanupInterval);
      }

      if (!this.server) {
        resolve();
        return;
      }

      this.server.close((err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  /**
   * Handle HTTP request
   */
  private async handleRequest(req: http.IncomingMessage, res: http.ServerResponse) {
    // CORS headers
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

    if (req.method === 'OPTIONS') {
      res.writeHead(200);
      res.end();
      return;
    }

    const url = new URL(req.url!, `http://${req.headers.host}`);

    // POST /signal - Post signaling message
    if (req.method === 'POST' && url.pathname === '/signal') {
      try {
        const body = await this.readBody(req);
        const message: SignalingMessage = JSON.parse(body);

        // Validate message
        if (!message.from || !message.to || !message.type || !message.data) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: 'Invalid message format' }));
          return;
        }

        this.queue.post(message);

        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ success: true }));
      } catch (err) {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Invalid JSON' }));
      }
      return;
    }

    // GET /poll?fingerprint=xxx - Poll for messages
    if (req.method === 'GET' && url.pathname === '/poll') {
      const fingerprint = url.searchParams.get('fingerprint');

      if (!fingerprint) {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Missing fingerprint parameter' }));
        return;
      }

      const messages = this.queue.poll(fingerprint);

      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(messages));
      return;
    }

    // GET /health - Health check
    if (req.method === 'GET' && url.pathname === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ status: 'ok' }));
      return;
    }

    // 404
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Not found' }));
  }

  /**
   * Read request body
   */
  private readBody(req: http.IncomingMessage): Promise<string> {
    return new Promise((resolve, reject) => {
      let body = '';
      req.on('data', (chunk) => {
        body += chunk;
      });
      req.on('end', () => {
        resolve(body);
      });
      req.on('error', reject);
    });
  }
}

// CLI usage
if (import.meta.url === `file://${process.argv[1]}`) {
  const port = parseInt(process.argv[2] || '8080', 10);
  const server = new HTTPSignalingServer();

  server.start(port).then(() => {
    console.log('Signaling server started');
    console.log('Endpoints:');
    console.log(`  POST /signal - Post signaling message`);
    console.log(`  GET /poll?fingerprint=<fpr> - Poll for messages`);
    console.log(`  GET /health - Health check`);
  });

  process.on('SIGINT', async () => {
    console.log('\nShutting down...');
    await server.stop();
    process.exit(0);
  });
}
