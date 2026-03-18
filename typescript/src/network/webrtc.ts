/**
 * WebRTC Client - P2P over WebRTC Data Channels
 * 
 * Handles WebRTC connections with mTLS authentication over data channels.
 */

import pRetry from 'p-retry';
import type { Fingerprint, Storage } from '../types/index.js';
import type { Account } from '../account.js';
import {  } from '../crypto/index.js';

export interface WebRTCConfig {
  iceServers?: RTCIceServer[];
  timeout?: number;
}

/**
 * WebRTC-based client for browser P2P
 */
export class WebRTCClient {
  private account: Account;
  private storage: Storage;
  private peer: Fingerprint;
  private config: WebRTCConfig;
  private connection: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;

  constructor(
    account: Account,
    storage: Storage,
    peer: Fingerprint,
    config: WebRTCConfig = {}
  ) {
    this.account = account;
    this.storage = storage;
    this.peer = peer;
    this.config = {
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
      timeout: 30000,
      ...config,
    };
  }

  /**
   * Create WebRTC offer. Waits for ICE gathering to complete so the returned
   * SDP contains all candidates — no trickle-ICE signaling channel required.
   */
  async createOffer(): Promise<RTCSessionDescriptionInit> {
    this.connection = new RTCPeerConnection({ iceServers: this.config.iceServers });

    // Create data channel
    this.dataChannel = this.connection.createDataChannel('mau', {
      ordered: true,
    });

    this.setupDataChannel(this.dataChannel);

    const offer = await this.connection.createOffer();
    await this.connection.setLocalDescription(offer);
    await this.waitForICEGathering();

    return this.connection.localDescription as RTCSessionDescriptionInit;
  }

  /**
   * Accept WebRTC offer and create answer. Waits for ICE gathering to complete
   * so the returned SDP contains all candidates.
   */
  async acceptOffer(offer: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit> {
    this.connection = new RTCPeerConnection({ iceServers: this.config.iceServers });

    // Wait for data channel from remote
    this.connection.ondatachannel = (event) => {
      this.dataChannel = event.channel;
      this.setupDataChannel(this.dataChannel);
    };

    await this.connection.setRemoteDescription(offer);
    const answer = await this.connection.createAnswer();
    await this.connection.setLocalDescription(answer);
    await this.waitForICEGathering();

    return this.connection.localDescription as RTCSessionDescriptionInit;
  }

  /**
   * Complete connection with answer
   */
  async completeConnection(answer: RTCSessionDescriptionInit): Promise<void> {
    if (!this.connection) {
      throw new Error('No connection to complete');
    }
    await this.connection.setRemoteDescription(answer);
  }

  /**
   * Add ICE candidate
   */
  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.connection) {
      throw new Error('No connection');
    }
    await this.connection.addIceCandidate(candidate);
  }

  /**
   * Perform mTLS handshake over data channel.
   * Retries up to maxRetries times with exponential backoff on timeout.
   */
  async performMTLS(maxRetries = 2): Promise<boolean> {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      throw new Error('Data channel not ready');
    }
    return pRetry(() => this.doPerformMTLS(), {
      retries: maxRetries,
      minTimeout: 500,
      factor: 2,
    });
  }

  private async doPerformMTLS(): Promise<boolean> {
    const publicKey = this.account.getPublicKey();
    const challenge = crypto.getRandomValues(new Uint8Array(32));

    // Register response handler BEFORE sending offer to eliminate the race condition
    const responsePromise = new Promise<boolean>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('mTLS timeout')), 5000);

      const handler = async (event: MessageEvent) => {
        try {
          const response = JSON.parse(event.data);
          if (response.type !== 'mtls_response') {return;}

          const { deserializePublicKey, getFingerprint, verify } = await import('../crypto/index.js');
          const peerKey = await deserializePublicKey(response.publicKey);
          const peerFingerprint = getFingerprint(peerKey);

          clearTimeout(timeout);
          this.dataChannel!.removeEventListener('message', handler);

          if (peerFingerprint !== this.peer) { resolve(false); return; }

          const challengeBytes = new Uint8Array(response.challenge);
          resolve(await verify(challengeBytes, response.signature, peerKey));
        } catch (err) {
          clearTimeout(timeout);
          this.dataChannel!.removeEventListener('message', handler);
          reject(err);
        }
      };

      this.dataChannel!.addEventListener('message', handler);
    });

    this.dataChannel!.send(JSON.stringify({
      type: 'mtls_offer',
      publicKey,
      challenge: Array.from(challenge),
    }));

    return responsePromise;
  }

  /**
   * Send HTTP-style request over data channel with retry.
   */
  async sendRequest(
    request: {
      method: string;
      path: string;
      query?: Record<string, string>;
      headers?: Record<string, string>;
    },
    maxRetries = 2,
    initialDelayMs = 200,
  ): Promise<{
    status: number;
    headers: Record<string, string>;
    body: Uint8Array | string;
  }> {
    return pRetry(() => this.doSendRequest(request), {
      retries: maxRetries,
      minTimeout: initialDelayMs,
      factor: 2,
    });
  }

  private doSendRequest(request: {
    method: string;
    path: string;
    query?: Record<string, string>;
    headers?: Record<string, string>;
  }): Promise<{
    status: number;
    headers: Record<string, string>;
    body: Uint8Array | string;
  }> {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      return Promise.reject(new Error('Data channel not ready'));
    }

    const message = JSON.stringify({
      type: 'request',
      method: request.method,
      path: request.path,
      query: request.query || {},
      headers: request.headers || {},
    });

    // Register response handler BEFORE sending to eliminate the race condition
    const responsePromise = new Promise<{ status: number; headers: Record<string, string>; body: Uint8Array | string }>(
      (resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('Request timeout')), 30000);

        const handler = (event: MessageEvent) => {
          if (!this.dataChannel) {return;}
          try {
            const response = JSON.parse(event.data);
            if (response.type !== 'response') {return;}

            clearTimeout(timeout);
            this.dataChannel.removeEventListener('message', handler);

            let body = response.body;
            if (Array.isArray(body)) {body = new Uint8Array(body);}

            resolve({ status: response.status, headers: response.headers || {}, body });
          } catch (err) {
            clearTimeout(timeout);
            this.dataChannel.removeEventListener('message', handler);
            reject(err);
          }
        };

        this.dataChannel!.addEventListener('message', handler);
      }
    );

    this.dataChannel.send(message);
    return responsePromise;
  }

  /**
   * Fetch file list from peer over WebRTC
   */
  async fetchFileList(after?: Date): Promise<import('../types/index.js').FileListResponse> {
    const query: Record<string, string> = {};
    if (after) {
      query.after = after.toISOString();
    }

    const response = await this.sendRequest({
      method: 'GET',
      path: `/p2p/${this.peer}`,
      query,
    });

    if (response.status !== 200) {
      throw new Error(`HTTP ${response.status}`);
    }

    const bodyText =
      typeof response.body === 'string'
        ? response.body
        : new TextDecoder().decode(response.body);

    return JSON.parse(bodyText);
  }

  /**
   * Download file from peer over WebRTC
   */
  async downloadFile(fileName: string): Promise<Uint8Array> {
    const response = await this.sendRequest({
      method: 'GET',
      path: `/p2p/${this.peer}/${encodeURIComponent(fileName)}`,
    });

    if (response.status !== 200) {
      throw new Error(`HTTP ${response.status}`);
    }

    if (response.body instanceof Uint8Array) {
      return response.body;
    }

    // Convert string to Uint8Array if needed
    return new TextEncoder().encode(response.body);
  }

  /**
   * Download file version from peer over WebRTC
   */
  async downloadFileVersion(fileName: string, version: string): Promise<Uint8Array> {
    const response = await this.sendRequest({
      method: 'GET',
      path: `/p2p/${this.peer}/${encodeURIComponent(fileName)}.versions/${encodeURIComponent(
        version
      )}`,
    });

    if (response.status !== 200) {
      throw new Error(`HTTP ${response.status}`);
    }

    if (response.body instanceof Uint8Array) {
      return response.body;
    }

    return new TextEncoder().encode(response.body);
  }

  /**
   * Wait for ICE gathering to complete (or timeout).
   * Call before close() to avoid "ICE candidate on closed connection" errors.
   */
  async waitForICEGathering(timeoutMs = 5000): Promise<void> {
    if (!this.connection || this.connection.iceGatheringState === 'complete') {return;}
    await new Promise<void>((resolve) => {
      const timeout = setTimeout(resolve, timeoutMs);
      this.connection!.onicegatheringstatechange = () => {
        if (this.connection?.iceGatheringState === 'complete') {
          clearTimeout(timeout);
          resolve();
        }
      };
    });
  }

  /**
   * Close connection
   */
  close(): void {
    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }
    if (this.connection) {
      this.connection.close();
      this.connection = null;
    }
  }

  private setupDataChannel(channel: RTCDataChannel): void {
    channel.onopen = () => {
    };

    channel.onclose = () => {
    };

    channel.onerror = (error) => {
      console.error('Data channel error:', error);
    };
  }
}
