/**
 * WebRTC Mocks for Testing
 */

// Mock RTCPeerConnection
export class MockRTCPeerConnection {
  localDescription: RTCSessionDescription | null = null;
  remoteDescription: RTCSessionDescription | null = null;
  connectionState: RTCPeerConnectionState = 'new';
  iceConnectionState: RTCIceConnectionState = 'new';
  signalingState: RTCSignalingState = 'stable';
  
  private channels: Map<string, MockRTCDataChannel> = new Map();
  private eventListeners: Map<string, Function[]> = new Map();

  constructor(public config?: RTCConfiguration) {}

  createDataChannel(label: string, options?: RTCDataChannelInit): MockRTCDataChannel {
    const channel = new MockRTCDataChannel(label, options);
    this.channels.set(label, channel);
    
    // Simulate channel opening
    setTimeout(() => {
      channel.readyState = 'open';
      channel.dispatchEvent('open');
    }, 10);
    
    return channel;
  }

  async createOffer(options?: RTCOfferOptions): Promise<RTCSessionDescriptionInit> {
    return {
      type: 'offer',
      sdp: 'mock-offer-sdp'
    };
  }

  async createAnswer(options?: RTCAnswerOptions): Promise<RTCSessionDescriptionInit> {
    return {
      type: 'answer',
      sdp: 'mock-answer-sdp'
    };
  }

  async setLocalDescription(description: RTCSessionDescriptionInit): Promise<void> {
    this.localDescription = description as RTCSessionDescription;
    this.signalingState = description.type === 'offer' ? 'have-local-offer' : 'stable';
  }

  async setRemoteDescription(description: RTCSessionDescriptionInit): Promise<void> {
    this.remoteDescription = description as RTCSessionDescription;
    this.signalingState = description.type === 'offer' ? 'have-remote-offer' : 'stable';
    
    // Simulate connection
    setTimeout(() => {
      this.connectionState = 'connected';
      this.iceConnectionState = 'connected';
      this.dispatchEvent('connectionstatechange');
    }, 20);
  }

  async addIceCandidate(candidate?: RTCIceCandidateInit): Promise<void> {
    // Mock ICE candidate handling
  }

  addEventListener(event: string, callback: Function): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event)!.push(callback);
  }

  removeEventListener(event: string, callback: Function): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      const index = listeners.indexOf(callback);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  dispatchEvent(event: string): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach(callback => callback({ type: event }));
    }
  }

  close(): void {
    this.connectionState = 'closed';
    this.channels.forEach(channel => channel.close());
    this.channels.clear();
  }

  // Additional methods for testing
  getDataChannel(label: string): MockRTCDataChannel | undefined {
    return this.channels.get(label);
  }

  simulateDataChannelOpen(label: string): void {
    const channel = this.channels.get(label);
    if (channel) {
      channel.readyState = 'open';
      channel.dispatchEvent('open');
    }
  }
}

// Mock RTCDataChannel
export class MockRTCDataChannel {
  readyState: RTCDataChannelState = 'connecting';
  bufferedAmount = 0;
  
  private eventListeners: Map<string, Function[]> = new Map();
  private messageQueue: any[] = [];

  constructor(
    public label: string,
    public options?: RTCDataChannelInit
  ) {}

  send(data: string | Blob | ArrayBuffer | ArrayBufferView): void {
    if (this.readyState !== 'open') {
      throw new Error('DataChannel is not open');
    }
    this.messageQueue.push(data);
    
    // Simulate message delivery
    setTimeout(() => {
      this.dispatchEvent('message', { data });
    }, 5);
  }

  close(): void {
    this.readyState = 'closed';
    this.dispatchEvent('close');
  }

  addEventListener(event: string, callback: Function): void {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event)!.push(callback);
  }

  removeEventListener(event: string, callback: Function): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      const index = listeners.indexOf(callback);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  dispatchEvent(event: string, data?: any): void {
    const listeners = this.eventListeners.get(event);
    if (listeners) {
      listeners.forEach(callback => callback({ type: event, ...data }));
    }
  }

  // Helper for testing
  simulateMessage(data: any): void {
    this.dispatchEvent('message', { data });
  }
}

// Setup global mocks
export function setupWebRTCMocks(): void {
  (global as any).RTCPeerConnection = MockRTCPeerConnection;
  (global as any).RTCDataChannel = MockRTCDataChannel;
  (global as any).RTCSessionDescription = class {
    constructor(public type: string, public sdp: string) {}
  };
}

export function cleanupWebRTCMocks(): void {
  delete (global as any).RTCPeerConnection;
  delete (global as any).RTCDataChannel;
  delete (global as any).RTCSessionDescription;
}
