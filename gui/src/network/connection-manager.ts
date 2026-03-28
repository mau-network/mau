/**
 * WebRTC Connection Manager
 * 
 * Manages WebRTC server lifecycle and DHT connectivity for the GUI application.
 * Starts WebRTC server when account is created/unlocked and joins the Kademlia DHT
 * network with HTTP bootstrap peers. DHT uses relay signaling for subsequent connections.
 */

import type { Account } from '@mau-network/mau';
import { WebRTCServer, KademliaDHT } from '@mau-network/mau';
import type { Peer } from '@mau-network/mau';

export interface ConnectionManagerConfig {
  /**
   * Bootstrap peers for DHT network (with HTTPS addresses exposing /p2p/dht/offer)
   * @default []
   */
  bootstrapPeers?: Peer[];
  
  /**
   * ICE servers for WebRTC STUN/TURN
   * @default [{ urls: 'stun:stun.l.google.com:19302' }]
   */
  iceServers?: RTCIceServer[];
}

/**
 * Manages WebRTC connections and DHT network for P2P communication
 */
export class ConnectionManager {
  private server: WebRTCServer | null = null;
  private dht: KademliaDHT | null = null;
  private config: Required<ConnectionManagerConfig>;

  constructor(config: ConnectionManagerConfig = {}) {
    this.config = {
      bootstrapPeers: config.bootstrapPeers ?? [],
      iceServers: config.iceServers ?? [{ urls: 'stun:stun.l.google.com:19302' }],
    };
  }

  /**
   * Start WebRTC server and join DHT network
   */
  async start(account: Account): Promise<void> {
    if (this.server) {
      console.warn('[ConnectionManager] Already started');
      return;
    }

    const storage = account.getStorage();
    const fingerprint = account.getFingerprint();

    console.log(`[ConnectionManager] Starting for ${fingerprint.slice(0, 16)}...`);

    // Start WebRTC server to accept incoming connections
    this.server = new WebRTCServer(account, storage, {
      iceServers: this.config.iceServers,
    });
    console.log('[ConnectionManager] WebRTC server started');

    // Initialize Kademlia DHT
    this.dht = new KademliaDHT(account, this.config.iceServers);
    console.log('[ConnectionManager] Kademlia DHT initialized');

    // Join DHT network with bootstrap peers if available
    if (this.config.bootstrapPeers.length > 0) {
      console.log(`[ConnectionManager] Joining DHT with ${this.config.bootstrapPeers.length} bootstrap peer(s)...`);
      console.log('[ConnectionManager] Bootstrap uses WebSocket signaling (browser) or HTTPS with mTLS (Go)');
      console.log('[ConnectionManager] Subsequent connections use DHT relay signaling');
      try {
        await this.dht.join(this.config.bootstrapPeers);
        console.log('[ConnectionManager] Successfully joined DHT network');
      } catch (error) {
        console.warn('[ConnectionManager] Failed to join DHT network:', error);
      }
    } else {
      console.log('[ConnectionManager] No bootstrap peers configured, skipping DHT join');
      console.log('[ConnectionManager] Hint: Add bootstrap peers with HTTPS addresses exposing /p2p/dht/offer');
    }

    console.log('[ConnectionManager] Startup complete');
  }

  /**
   * Stop WebRTC server and disconnect from network
   */
  async stop(): Promise<void> {
    if (!this.server) {
      console.warn('[ConnectionManager] Not started');
      return;
    }

    console.log('[ConnectionManager] Stopping...');

    // Stop DHT
    if (this.dht) {
      this.dht.stop();
      this.dht = null;
    }

    // Stop WebRTC server
    this.server.stop();
    this.server = null;

    console.log('[ConnectionManager] Stopped');
  }

  /**
   * Check if connection manager is running
   */
  isRunning(): boolean {
    return this.server !== null;
  }

  /**
   * Get WebRTC server instance
   */
  getServer(): WebRTCServer | null {
    return this.server;
  }

  /**
   * Get DHT instance
   */
  getDHT(): KademliaDHT | null {
    return this.dht;
  }
}

