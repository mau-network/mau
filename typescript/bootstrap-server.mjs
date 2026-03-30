#!/usr/bin/env node
/**
 * Mau WebRTC Bootstrap Server (Node.js)
 * 
 * This is a Node.js server that acts as a WebRTC peer and DHT bootstrap node.
 * It uses the TypeScript library to provide the same functionality as the Go server
 * but with WebRTC support for browser clients.
 * 
 * Usage:
 *   node bootstrap-server.mjs --data-dir ./.bootstrap-peer --port 8444
 */

// Set up WebRTC polyfill for Node.js FIRST
import './node-webrtc-polyfill.mjs';

import { Account, WebRTCServer, KademliaDHT } from './dist/index.js';
import { WebSocketServer } from 'ws';
import { createServer } from 'http';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import * as fs from 'fs/promises';
import { NodeFSStorage } from './node-fs-storage.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));

// Configuration
const DEFAULT_PORT = 8444;
const DEFAULT_DATA_DIR = './.bootstrap-peer-node';
const ACCOUNT_NAME = 'Bootstrap Peer';
const ACCOUNT_EMAIL = 'bootstrap@mau.local';
const ACCOUNT_PASSPHRASE = 'bootstrap-peer-secure-passphrase-change-me';

/**
 * Parse command-line arguments
 */
function parseArgs() {
  const args = process.argv.slice(2);
  const config = {
    port: DEFAULT_PORT,
    dataDir: DEFAULT_DATA_DIR,
  };

  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case '--port':
      case '-p':
        config.port = parseInt(args[++i], 10);
        break;
      case '--data-dir':
      case '-d':
        config.dataDir = args[++i];
        break;
      case '--help':
      case '-h':
        console.log(`
Usage: node bootstrap-server.mjs [options]

Options:
  --port, -p <port>       WebSocket signaling port (default: ${DEFAULT_PORT})
  --data-dir, -d <dir>    Data directory (default: ${DEFAULT_DATA_DIR})
  --help, -h              Show this help message

Example:
  node bootstrap-server.mjs --port 8444 --data-dir ./bootstrap-data
`);
        process.exit(0);
      default:
        console.error(`Unknown option: ${args[i]}`);
        process.exit(1);
    }
  }

  return config;
}

/**
 * Load or create bootstrap account
 */
async function getBootstrapAccount(dataDir) {
  // Create data directory if it doesn't exist
  await fs.mkdir(dataDir, { recursive: true });

  // Use NodeFSStorage for actual filesystem persistence
  const storage = new NodeFSStorage(dataDir);
  const accountPath = 'account'; // Relative to dataDir

  try {
    // Try to load existing account
    const account = await Account.load(
      storage,
      accountPath,
      ACCOUNT_PASSPHRASE
    );
    console.log('✅ Loaded existing account');
    return { account, storage };
  } catch (err) {
    // Create new account
    console.log('📝 Creating new account...');
    const account = await Account.create(storage, accountPath, {
      name: ACCOUNT_NAME,
      email: ACCOUNT_EMAIL,
      passphrase: ACCOUNT_PASSPHRASE,
    });
    // Account is automatically saved during creation
    console.log('✅ Created new account');
    return { account, storage };
  }
}

/**
 * WebSocket signaling server that integrates with WebRTC
 */
class SignalingServer {
  constructor(webrtcServer, dht) {
    this.webrtcServer = webrtcServer;
    this.dht = dht;
    this.clients = new Map(); // fingerprint -> ws connection
    this.pendingIceCandidates = new Map(); // connectionId -> candidate[]
  }

  handleConnection(ws) {
    let clientFingerprint = null;

    ws.on('message', async (data) => {
      try {
        const msg = JSON.parse(data.toString());

        switch (msg.type) {
          case 'register':
            clientFingerprint = msg.fingerprint;
            this.clients.set(clientFingerprint, ws);
            console.log(`[Signaling] Client registered: ${clientFingerprint.slice(0, 16)}...`);
            break;

          case 'offer':
            await this.handleOffer(ws, msg);
            break;

          case 'ice':
            await this.handleICE(msg);
            break;

          default:
            console.warn(`[Signaling] Unknown message type: ${msg.type}`);
        }
      } catch (err) {
        console.error('[Signaling] Error handling message:', err);
        ws.send(JSON.stringify({
          type: 'error',
          error: err.message,
        }));
      }
    });

    ws.on('close', () => {
      if (clientFingerprint) {
        this.clients.delete(clientFingerprint);
        console.log(`[Signaling] Client disconnected: ${clientFingerprint.slice(0, 16)}...`);
      }
    });
  }

  async handleOffer(ws, msg) {
    const { from, to, offer } = msg;

    if (to === this.webrtcServer.account.getFingerprint()) {
      // Offer is for this server
      console.log(`[Signaling] Accepting connection from ${from.slice(0, 16)}...`);
      
      const connectionId = from;
      const answer = await this.webrtcServer.acceptConnection(connectionId, offer);

      // Send answer back
      ws.send(JSON.stringify({
        type: 'answer',
        answer: answer,
      }));

      // Set up ICE candidate callback
      this.webrtcServer.onSignaling(connectionId, (candidate) => {
        if (ws.readyState === 1) { // WebSocket.OPEN
          ws.send(JSON.stringify({
            type: 'ice',
            candidate: candidate,
          }));
        }
      });

      console.log(`[Signaling] Connection established with ${from.slice(0, 16)}...`);
      
      // Replay any queued ICE candidates that arrived early
      const queued = this.pendingIceCandidates.get(connectionId);
      if (queued && queued.length > 0) {
        console.log(`[Signaling] Replaying ${queued.length} queued ICE candidates for ${connectionId.slice(0, 16)}...`);
        for (const candidate of queued) {
          await this.webrtcServer.addIceCandidate(connectionId, candidate).catch(err => {
            console.warn(`[Signaling] Failed to replay ICE candidate: ${err.message}`);
          });
        }
        this.pendingIceCandidates.delete(connectionId);
      }
      
      // Add to DHT when data channel opens
      // This will be handled by WebRTCServer's data channel setup
    } else {
      // Forward to another peer
      const targetWs = this.clients.get(to);
      if (targetWs) {
        targetWs.send(JSON.stringify(msg));
      } else {
        ws.send(JSON.stringify({
          type: 'error',
          error: `Peer ${to} not connected`,
        }));
      }
    }
  }

  async handleICE(msg) {
    const { to, candidate } = msg;
    
    if (to === this.webrtcServer.account.getFingerprint()) {
      // ICE candidate for this server
      const connectionId = msg.from;
      
      // Try to add candidate, or queue it if connection not ready yet
      try {
        await this.webrtcServer.addIceCandidate(connectionId, candidate);
      } catch (err) {
        if (err.message.includes('not found')) {
          // Connection not established yet - queue the candidate
          if (!this.pendingIceCandidates.has(connectionId)) {
            this.pendingIceCandidates.set(connectionId, []);
          }
          this.pendingIceCandidates.get(connectionId).push(candidate);
          console.log(`[Signaling] Queued ICE candidate for pending connection ${connectionId.slice(0, 16)}...`);
        } else {
          throw err;
        }
      }
    } else {
      // Forward to another peer
      const targetWs = this.clients.get(to);
      if (targetWs) {
        targetWs.send(JSON.stringify(msg));
      }
    }
  }
}

/**
 * Update GUI .env file with server configuration (only if changed)
 */
async function updateGUIEnv(fingerprint, port) {
  const guiEnvPath = join(__dirname, '../gui/.env');
  const envContent = `# Development Bootstrap Peer (Node.js)
# Auto-generated by bootstrap server
VITE_DEV_PEER_FINGERPRINT=${fingerprint}
VITE_DEV_PEER_WS_ADDRESS=localhost:${port}
`;

  try {
    // Check if file exists and has the same content
    try {
      const existing = await fs.readFile(guiEnvPath, 'utf-8');
      if (existing.trim() === envContent.trim()) {
        console.log(`✅ GUI .env file already up-to-date`);
        return;
      }
    } catch {
      // File doesn't exist or can't be read, write it
    }
    
    await fs.writeFile(guiEnvPath, envContent, 'utf-8');
    console.log(`✅ Updated GUI .env file`);
  } catch (err) {
    console.warn(`⚠️  Could not update GUI .env: ${err.message}`);
  }
}

/**
 * Main server
 */
async function main() {
  const config = parseArgs();

  console.log('🚀 Starting Mau WebRTC Bootstrap Server...\n');

  // Load account
  const { account, storage } = await getBootstrapAccount(config.dataDir);
  const fingerprint = account.getFingerprint();
  
  console.log(`📌 Server Fingerprint: ${fingerprint}`);
  console.log(`📂 Data Directory: ${config.dataDir}`);
  console.log(`🔌 WebSocket Port: ${config.port}\n`);

  // Update GUI .env file automatically
  await updateGUIEnv(fingerprint, config.port);
  console.log();

  // Create WebRTC server with connection callback
  const webrtcServer = new WebRTCServer(account, storage, {
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
    onConnectionEstablished: (connectionId, peerConnection, dataChannel) => {
      console.log(`[Bootstrap] Peer connected: ${connectionId.slice(0, 16)}...`);
      
      // Register this connection with the DHT so it can be used for relay signaling
      dht.registerConnection(connectionId, peerConnection, dataChannel);
      console.log(`[Bootstrap] Peer registered in DHT: ${connectionId.slice(0, 16)}...`);
      
      // Clean up when channel closes
      dataChannel.addEventListener('close', () => {
        console.log(`[Bootstrap] Peer disconnected: ${connectionId.slice(0, 16)}...`);
      });
    },
  });

  // Create DHT
  const dht = new KademliaDHT(account, [{ urls: 'stun:stun.l.google.com:19302' }]);

  // Start DHT (no bootstrap peers - this IS the bootstrap)
  await dht.join([]);
  console.log('✅ DHT initialized\n');
  
  // Register bootstrap server itself in routing table
  // This ensures the first client gets at least one peer (the bootstrap) in find_peer responses
  const wsAddress = `ws://localhost:${config.port}`;
  dht.registerSelf(wsAddress);
  console.log(`✅ Bootstrap self-registered at ${wsAddress}\n`);

  // Create HTTP server for WebSocket
  const httpServer = createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end(`Mau WebRTC Bootstrap Server\nFingerprint: ${fingerprint}\n`);
  });

  // Create WebSocket server
  const wss = new WebSocketServer({ server: httpServer });
  const signaling = new SignalingServer(webrtcServer, dht);

  wss.on('connection', (ws) => {
    signaling.handleConnection(ws);
  });

  // Start listening
  httpServer.listen(config.port, () => {
    console.log('💡 Server ready!');
    console.log(`📍 WebSocket: ws://localhost:${config.port}`);
    console.log(`🌐 HTTP: http://localhost:${config.port}`);
    console.log('\n🔗 Configure GUI with:');
    console.log(`   VITE_DEV_PEER_FINGERPRINT=${fingerprint}`);
    console.log(`   VITE_DEV_PEER_WS_ADDRESS=localhost:${config.port}`);
    console.log('\n✨ Waiting for connections...\n');
  });

  // Graceful shutdown
  process.on('SIGINT', () => {
    console.log('\n\n🛑 Shutting down...');
    dht.stop();
    wss.close();
    httpServer.close();
    process.exit(0);
  });
}

// Run
main().catch((err) => {
  console.error('❌ Fatal error:', err);
  process.exit(1);
});
