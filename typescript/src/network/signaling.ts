/**
 * Signaling - WebRTC Signaling Coordination
 * 
 * Provides signaling mechanism for WebRTC peer connection establishment.
 * Can use WebSocket server or simple HTTP POST exchange.
 */

import type { Fingerprint } from '../types/index.js';

export interface SignalingMessage {
  from: Fingerprint;
  to: Fingerprint;
  type: 'offer' | 'answer' | 'ice-candidate';
  data: any;
}

/**
 * Simple in-memory signaling server for local/development use
 */
export class LocalSignalingServer {
  private pending: Map<string, SignalingMessage[]> = new Map();

  /**
   * Post a signaling message
   */
  async post(message: SignalingMessage): Promise<void> {
    const key = message.to;
    const messages = this.pending.get(key) || [];
    messages.push(message);
    this.pending.set(key, messages);
  }

  /**
   * Poll for messages addressed to a fingerprint
   */
  async poll(fingerprint: Fingerprint): Promise<SignalingMessage[]> {
    const messages = this.pending.get(fingerprint) || [];
    this.pending.delete(fingerprint);
    return messages;
  }

  /**
   * Clear all pending messages
   */
  clear(): void {
    this.pending.clear();
  }
}

/**
 * WebSocket-based signaling client
 */
export class WebSocketSignaling {
  private ws: WebSocket | null = null;
  private fingerprint: Fingerprint;
  private url: string;
  private messageHandlers: ((message: SignalingMessage) => void)[] = [];
  private connected: Promise<void>;

  constructor(url: string, fingerprint: Fingerprint) {
    this.url = url;
    this.fingerprint = fingerprint;
    this.connected = this.connect();
  }

  /**
   * Connect to signaling server
   */
  private connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        // Register with server
        this.ws!.send(
          JSON.stringify({
            type: 'register',
            fingerprint: this.fingerprint,
          })
        );
        resolve();
      };

      this.ws.onerror = (error) => {
        reject(error);
      };

      this.ws.onmessage = (event) => {
        try {
          const message: SignalingMessage = JSON.parse(event.data);
          this.messageHandlers.forEach((handler) => handler(message));
        } catch (err) {
          console.error('[Signaling] Error parsing message:', err);
        }
      };

      this.ws.onclose = () => {
        console.log('[Signaling] Disconnected');
      };
    });
  }

  /**
   * Send signaling message
   */
  async send(message: SignalingMessage): Promise<void> {
    await this.connected;
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket not connected');
    }
    this.ws.send(JSON.stringify(message));
  }

  /**
   * Register message handler
   */
  onMessage(handler: (message: SignalingMessage) => void): void {
    this.messageHandlers.push(handler);
  }

  /**
   * Close connection
   */
  close(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

/**
 * HTTP-based signaling client
 */
export class HTTPSignaling {
  private fingerprint: Fingerprint;
  private serverUrl: string;
  private pollInterval: number = 1000;
  private polling: boolean = false;
  private messageHandlers: ((message: SignalingMessage) => void)[] = [];

  constructor(serverUrl: string, fingerprint: Fingerprint) {
    this.serverUrl = serverUrl;
    this.fingerprint = fingerprint;
  }

  /**
   * Send signaling message
   */
  async send(message: SignalingMessage): Promise<void> {
    const response = await fetch(`${this.serverUrl}/signal`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(message),
    });

    if (!response.ok) {
      throw new Error(`Signaling failed: ${response.status}`);
    }
  }

  /**
   * Start polling for messages
   */
  startPolling(): void {
    if (this.polling) return;
    this.polling = true;
    this.poll();
  }

  /**
   * Stop polling
   */
  stopPolling(): void {
    this.polling = false;
  }

  /**
   * Register message handler
   */
  onMessage(handler: (message: SignalingMessage) => void): void {
    this.messageHandlers.push(handler);
  }

  private async poll(): Promise<void> {
    while (this.polling) {
      try {
        const response = await fetch(
          `${this.serverUrl}/poll?fingerprint=${this.fingerprint}`
        );

        if (response.ok) {
          const messages: SignalingMessage[] = await response.json();
          messages.forEach((message) => {
            this.messageHandlers.forEach((handler) => handler(message));
          });
        }
      } catch (err) {
        console.error('[Signaling] Poll error:', err);
      }

      await new Promise((resolve) => setTimeout(resolve, this.pollInterval));
    }
  }
}

/**
 * Helper to establish WebRTC connection with signaling
 */
export class SignaledConnection {
  private signaling: WebSocketSignaling | HTTPSignaling;
  private localFingerprint: Fingerprint;
  private remoteFingerprint: Fingerprint;
  private onOfferCallback?: (offer: RTCSessionDescriptionInit) => void;
  private onAnswerCallback?: (answer: RTCSessionDescriptionInit) => void;
  private onICECallback?: (candidate: RTCIceCandidateInit) => void;

  constructor(
    signaling: WebSocketSignaling | HTTPSignaling,
    localFingerprint: Fingerprint,
    remoteFingerprint: Fingerprint
  ) {
    this.signaling = signaling;
    this.localFingerprint = localFingerprint;
    this.remoteFingerprint = remoteFingerprint;

    // Setup message handler
    this.signaling.onMessage(this.handleMessage.bind(this));
  }

  /**
   * Send offer to remote peer
   */
  async sendOffer(offer: RTCSessionDescriptionInit): Promise<void> {
    await this.signaling.send({
      from: this.localFingerprint,
      to: this.remoteFingerprint,
      type: 'offer',
      data: offer,
    });
  }

  /**
   * Send answer to remote peer
   */
  async sendAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    await this.signaling.send({
      from: this.localFingerprint,
      to: this.remoteFingerprint,
      type: 'answer',
      data: answer,
    });
  }

  /**
   * Send ICE candidate to remote peer
   */
  async sendICECandidate(candidate: RTCIceCandidateInit): Promise<void> {
    await this.signaling.send({
      from: this.localFingerprint,
      to: this.remoteFingerprint,
      type: 'ice-candidate',
      data: candidate,
    });
  }

  /**
   * Register callback for offer
   */
  onOffer(callback: (offer: RTCSessionDescriptionInit) => void): void {
    this.onOfferCallback = callback;
  }

  /**
   * Register callback for answer
   */
  onAnswer(callback: (answer: RTCSessionDescriptionInit) => void): void {
    this.onAnswerCallback = callback;
  }

  /**
   * Register callback for ICE candidate
   */
  onICECandidate(callback: (candidate: RTCIceCandidateInit) => void): void {
    this.onICECallback = callback;
  }

  private handleMessage(message: SignalingMessage): void {
    // Only handle messages for us from the remote peer
    if (message.to !== this.localFingerprint || message.from !== this.remoteFingerprint) {
      return;
    }

    switch (message.type) {
      case 'offer':
        if (this.onOfferCallback) {
          this.onOfferCallback(message.data);
        }
        break;

      case 'answer':
        if (this.onAnswerCallback) {
          this.onAnswerCallback(message.data);
        }
        break;

      case 'ice-candidate':
        if (this.onICECallback) {
          this.onICECallback(message.data);
        }
        break;
    }
  }
}
