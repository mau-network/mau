/**
 * WebRTC Server - Accept P2P connections over WebRTC Data Channels
 * 
 * Runs in browser, accepts incoming WebRTC connections and handles
 * HTTP-style requests over data channels with mTLS authentication.
 */

import type { Storage, Fingerprint } from '../types/index.js';
import type { Account } from '../account.js';
import type { ServerRequest, ServerResponse } from '../server.js';
import { Server } from '../server.js';
import { verify, getFingerprint, deserializePublicKey } from '../crypto/index.js';

export interface WebRTCServerConfig {
  iceServers?: RTCIceServer[];
  allowedPeers?: Fingerprint[]; // If set, only these peers can connect
}

export interface WebRTCConnection {
  peer: RTCPeerConnection;
  channel: RTCDataChannel;
  fingerprint: Fingerprint | null;
  authenticated: boolean;
}

/**
 * WebRTC Server accepts incoming peer connections
 */
export class WebRTCServer {
  private account: Account;
  private storage: Storage;
  private config: WebRTCServerConfig;
  private server: Server;
  private connections: Map<string, WebRTCConnection> = new Map();
  private signalingCallbacks: Map<string, (signal: any) => void> = new Map();

  constructor(account: Account, storage: Storage, config: WebRTCServerConfig = {}) {
    this.account = account;
    this.storage = storage;
    this.config = {
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
      ...config,
    };
    this.server = new Server(account, storage);
  }

  /**
   * Accept an incoming WebRTC offer and create answer
   */
  async acceptConnection(
    connectionId: string,
    offer: RTCSessionDescriptionInit
  ): Promise<RTCSessionDescriptionInit> {
    const peer = new RTCPeerConnection({ iceServers: this.config.iceServers });

    const connection: WebRTCConnection = {
      peer,
      channel: null as any,
      fingerprint: null,
      authenticated: false,
    };

    this.connections.set(connectionId, connection);

    // Wait for data channel from remote peer
    peer.ondatachannel = (event) => {
      connection.channel = event.channel;
      this.setupDataChannel(connectionId, connection.channel);
    };

    // Handle ICE candidates
    peer.onicecandidate = (event) => {
      if (event.candidate) {
        const callback = this.signalingCallbacks.get(connectionId);
        if (callback) {
          callback({
            type: 'ice-candidate',
            candidate: event.candidate,
          });
        }
      }
    };

    await peer.setRemoteDescription(offer);
    const answer = await peer.createAnswer();
    await peer.setLocalDescription(answer);

    return answer;
  }

  /**
   * Add ICE candidate to connection
   */
  async addIceCandidate(connectionId: string, candidate: RTCIceCandidateInit): Promise<void> {
    const connection = this.connections.get(connectionId);
    if (!connection) {
      throw new Error(`Connection ${connectionId} not found`);
    }
    await connection.peer.addIceCandidate(candidate);
  }

  /**
   * Set callback for signaling messages (ICE candidates, etc.)
   */
  onSignaling(connectionId: string, callback: (signal: any) => void): void {
    this.signalingCallbacks.set(connectionId, callback);
  }

  /**
   * Close a connection
   */
  closeConnection(connectionId: string): void {
    const connection = this.connections.get(connectionId);
    if (connection) {
      if (connection.channel) {
        connection.channel.close();
      }
      connection.peer.close();
      this.connections.delete(connectionId);
      this.signalingCallbacks.delete(connectionId);
    }
  }

  /**
   * Get all active connections
   */
  getConnections(): string[] {
    return Array.from(this.connections.keys());
  }

  /**
   * Setup data channel event handlers
   */
  private setupDataChannel(connectionId: string, channel: RTCDataChannel): void {
    channel.onopen = () => {
      console.log(`[WebRTCServer] Data channel opened: ${connectionId}`);
    };

    channel.onclose = () => {
      console.log(`[WebRTCServer] Data channel closed: ${connectionId}`);
      this.closeConnection(connectionId);
    };

    channel.onerror = (error) => {
      console.error(`[WebRTCServer] Data channel error: ${connectionId}`, error);
    };

    channel.onmessage = async (event) => {
      await this.handleMessage(connectionId, event.data);
    };
  }

  /**
   * Handle incoming message on data channel
   */
  private async handleMessage(connectionId: string, data: string): Promise<void> {
    const connection = this.connections.get(connectionId);
    if (!connection || !connection.channel) {
      return;
    }

    try {
      const message = JSON.parse(data);

      // Handle mTLS handshake
      if (message.type === 'mtls_offer') {
        await this.handleMTLSOffer(connectionId, message);
        return;
      }

      // Require authentication for all other messages
      if (!connection.authenticated) {
        this.sendError(connection.channel, 401, 'Unauthorized - mTLS required');
        return;
      }

      // Handle HTTP-style request
      if (message.type === 'request') {
        await this.handleRequest(connectionId, message);
        return;
      }

      console.warn(`[WebRTCServer] Unknown message type: ${message.type}`);
    } catch (err) {
      console.error(`[WebRTCServer] Error handling message:`, err);
      if (connection.channel.readyState === 'open') {
        this.sendError(connection.channel, 400, 'Bad Request');
      }
    }
  }

  /**
   * Handle mTLS handshake offer
   */
  private async handleMTLSOffer(connectionId: string, message: any): Promise<void> {
    const connection = this.connections.get(connectionId);
    if (!connection || !connection.channel) {
      return;
    }

    try {
      // Deserialize peer's public key
      const peerKey = await deserializePublicKey(message.publicKey);
      const peerFingerprint = await getFingerprint(peerKey);

      // Check if peer is allowed
      if (this.config.allowedPeers && !this.config.allowedPeers.includes(peerFingerprint)) {
        this.sendError(connection.channel, 403, 'Forbidden - Peer not allowed');
        this.closeConnection(connectionId);
        return;
      }

      connection.fingerprint = peerFingerprint;

      // Sign the challenge
      const challenge = new Uint8Array(message.challenge);
      const { sign } = await import('../crypto/index.js');
      const privateKey = await this.account.getPrivateKey();
      const signature = await sign(challenge, privateKey);

      // Send response
      const response = {
        type: 'mtls_response',
        publicKey: this.account.getPublicKey(),
        challenge: Array.from(challenge),
        signature,
      };

      connection.channel.send(JSON.stringify(response));
      connection.authenticated = true;

      console.log(`[WebRTCServer] mTLS authenticated: ${peerFingerprint.slice(0, 8)}...`);
    } catch (err) {
      console.error(`[WebRTCServer] mTLS handshake failed:`, err);
      this.sendError(connection.channel, 403, 'mTLS authentication failed');
      this.closeConnection(connectionId);
    }
  }

  /**
   * Handle HTTP-style request
   */
  private async handleRequest(connectionId: string, message: any): Promise<void> {
    const connection = this.connections.get(connectionId);
    if (!connection || !connection.channel) {
      return;
    }

    try {
      // Convert to ServerRequest format
      const request: ServerRequest = {
        method: message.method || 'GET',
        url: message.url || message.path || '',
        path: message.path || '',
        query: message.query || {},
        headers: message.headers || {},
      };

      // Process through server handler
      const response = await this.server.handleRequest(request);

      // Send response over data channel
      const responseMessage = {
        type: 'response',
        status: response.status,
        headers: response.headers,
        body:
          response.body instanceof Uint8Array
            ? Array.from(response.body)
            : response.body,
      };

      connection.channel.send(JSON.stringify(responseMessage));
    } catch (err) {
      console.error(`[WebRTCServer] Error handling request:`, err);
      this.sendError(connection.channel, 500, 'Internal Server Error');
    }
  }

  /**
   * Send error response
   */
  private sendError(channel: RTCDataChannel, status: number, message: string): void {
    if (channel.readyState !== 'open') {
      return;
    }

    const response = {
      type: 'response',
      status,
      headers: { 'Content-Type': 'text/plain' },
      body: message,
    };

    channel.send(JSON.stringify(response));
  }

  /**
   * Stop server and close all connections
   */
  stop(): void {
    for (const connectionId of this.connections.keys()) {
      this.closeConnection(connectionId);
    }
    this.connections.clear();
    this.signalingCallbacks.clear();
  }
}
