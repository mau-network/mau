/**
 * WebRTC Client - P2P over WebRTC Data Channels
 * 
 * Handles WebRTC connections with mTLS authentication over data channels.
 */

import type { Fingerprint, Storage } from '../types/index.js';
import type { Account } from '../account.js';
import { signAndEncrypt, decryptAndVerify } from '../crypto/index.js';

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
   * Create WebRTC offer
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

    return offer;
  }

  /**
   * Accept WebRTC offer and create answer
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

    return answer;
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
   * Perform mTLS handshake over data channel
   */
  async performMTLS(): Promise<boolean> {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      throw new Error('Data channel not ready');
    }

    // Send our public key
    const publicKey = this.account.getPublicKey();
    const challenge = crypto.getRandomValues(new Uint8Array(32));

    const handshakeOffer = JSON.stringify({
      type: 'mtls_offer',
      publicKey,
      challenge: Array.from(challenge),
    });

    this.dataChannel.send(handshakeOffer);

    // Wait for response
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('mTLS timeout')), 10000);

      const handler = async (event: MessageEvent) => {
        try {
          const response = JSON.parse(event.data);

          if (response.type === 'mtls_response') {
            // Verify peer's public key matches expected fingerprint
            const peerKey = await import('../crypto/index.js').then((m) =>
              m.deserializePublicKey(response.publicKey)
            );
            const peerFingerprint = await import('../crypto/index.js').then((m) =>
              m.getFingerprint(peerKey)
            );

            if (peerFingerprint !== this.peer) {
              clearTimeout(timeout);
              this.dataChannel!.removeEventListener('message', handler);
              resolve(false);
              return;
            }

            // Verify challenge signature
            const { verify } = await import('../crypto/index.js');
            const challengeBytes = new Uint8Array(response.challenge);
            const signatureValid = await verify(
              challengeBytes,
              response.signature,
              peerKey
            );

            clearTimeout(timeout);
            this.dataChannel!.removeEventListener('message', handler);
            resolve(signatureValid);
          }
        } catch (err) {
          clearTimeout(timeout);
          this.dataChannel!.removeEventListener('message', handler);
          reject(err);
        }
      };

      this.dataChannel!.addEventListener('message', handler);
    });
  }

  /**
   * Send request over data channel
   */
  async sendRequest(request: { method: string; path: string }): Promise<any> {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      throw new Error('Data channel not ready');
    }

    const message = JSON.stringify({
      type: 'request',
      ...request,
    });

    this.dataChannel.send(message);

    // Wait for response
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Request timeout')), 30000);

      const handler = (event: MessageEvent) => {
        try {
          const response = JSON.parse(event.data);
          if (response.type === 'response') {
            clearTimeout(timeout);
            this.dataChannel!.removeEventListener('message', handler);
            resolve(response);
          }
        } catch (err) {
          clearTimeout(timeout);
          this.dataChannel!.removeEventListener('message', handler);
          reject(err);
        }
      };

      this.dataChannel.addEventListener('message', handler);
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
      console.log('Data channel opened');
    };

    channel.onclose = () => {
      console.log('Data channel closed');
    };

    channel.onerror = (error) => {
      console.error('Data channel error:', error);
    };
  }
}
