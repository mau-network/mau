/**
 * E2E test for ICE candidate reuse
 * Verifies that pre-gathered ICE candidates are reused across connections
 */

import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import type { Account } from '../account.js';
import { KademliaDHT } from './dht.js';

// Mock Account
const mockAccount = {
  getFingerprint: () => 'A'.repeat(40),
  getPublicKey: () => '-----BEGIN PGP PUBLIC KEY BLOCK-----\n...',
} as Account;

// Mock ICE candidates
const mockICECandidates = [
  {
    candidate: 'candidate:1 1 UDP 2130706431 192.168.1.100 54321 typ host',
    sdpMLineIndex: 0,
    sdpMid: '0',
    toJSON: function() { return { candidate: this.candidate, sdpMLineIndex: this.sdpMLineIndex, sdpMid: this.sdpMid }; }
  },
  {
    candidate: 'candidate:2 1 UDP 1694498815 203.0.113.1 54321 typ srflx',
    sdpMLineIndex: 0,
    sdpMid: '0',
    toJSON: function() { return { candidate: this.candidate, sdpMLineIndex: this.sdpMLineIndex, sdpMid: this.sdpMid }; }
  },
] as RTCIceCandidate[];

describe('DHT ICE Candidate Reuse', () => {
  let dht: KademliaDHT;

  beforeEach(() => {
    dht = new KademliaDHT(mockAccount);
    
    // Manually populate pre-gathered ICE candidates (simulating gatherICECandidates)
    (dht as any).localICECandidates = mockICECandidates;
    (dht as any).iceGatheringComplete = true;
  });

  afterEach(() => {
    dht.stop();
  });

  it('should have pre-gathered ICE candidates available', () => {
    const candidateCount = (dht as any).localICECandidates.length;
    expect(candidateCount).toBe(2);
    expect((dht as any).iceGatheringComplete).toBe(true);
  });

  it('should reuse pre-gathered ICE candidates in relay connections', async () => {
    // Mock a relay peer connection (normalized fingerprints are lowercase)
    const relayFpr = 'b'.repeat(40);
    const mockChannel = {
      readyState: 'open',
      send: () => {},
      close: () => {},
    };
    
    // Register the relay peer
    (dht as any).conns.set(relayFpr, {
      pc: { close: () => {} } as any,
      ch: mockChannel,
      lastSeen: Date.now(),
    });
    (dht as any).addPeer({ 
      fingerprint: relayFpr, 
      address: 'http://relay:8080' 
    });
    
    // Mock the send method to capture relay_ice messages
    const sentICECandidates: any[] = [];
    (dht as any).send = (fpr: string, msg: any) => {
      if (msg.type === 'relay_ice') {
        sentICECandidates.push(msg.candidate);
      }
    };
    
    // Mock RTCPeerConnection to prevent actual WebRTC operations
    const mockPC = {
      localDescription: { type: 'offer', sdp: 'mock' },
      createDataChannel: () => ({
        readyState: 'connecting',
        onopen: null,
        onerror: null,
        onclose: null,
      }),
      setLocalDescription: async () => {},
      createOffer: async () => ({ type: 'offer', sdp: 'mock' }),
      onconnectionstatechange: null,
      onicegatheringstatechange: null,
      oniceconnectionstatechange: null,
      onicecandidate: null,
      close: () => {},
    };
    
    const originalRTC = global.RTCPeerConnection;
    (global as any).RTCPeerConnection = function() { return mockPC; };
    
    // Attempt a relay connection (don't await - it will timeout waiting for answer)
    const targetPeer = {
      fingerprint: 'c'.repeat(40),
      address: 'http://target:8080',
    };
    
    const _connectPromise = (dht as any).connectRelay(targetPeer, {
      fingerprint: relayFpr, 
      address: 'http://relay:8080' 
    });
    
    // Wait for ICE candidates to be sent (happens before timeout)
    await new Promise(resolve => setTimeout(resolve, 100));
    
    // Verify that pre-gathered ICE candidates were sent
    expect(sentICECandidates.length).toBe(2);
    expect(sentICECandidates[0]).toBeDefined();
    expect(sentICECandidates[1]).toBeDefined();
    
    // Restore
    (global as any).RTCPeerConnection = originalRTC;
    
    // Don't wait for the connection to complete (it will timeout after 30s)
  });

  it('should include pre-gathered candidates in HTTP offer', async () => {
    const candidateCount = (dht as any).localICECandidates.length;
    
    // Mock fetch to capture the HTTP request body
    let capturedBody: any = null;
    const originalFetch = global.fetch;
    global.fetch = (async (url: any, options: any) => {
      capturedBody = JSON.parse(options.body);
      // Return mock response
      return {
        ok: true,
        json: async () => ({ answer: { type: 'answer', sdp: 'mock' } }),
      };
    }) as any;
    
    // Mock RTCPeerConnection
    const mockChannel = {
      readyState: 'connecting',
      onopen: null as any,
      onerror: null as any,
      onclose: null as any,
    };
    
    const mockPC = {
      localDescription: { type: 'offer', sdp: 'mock' },
      createDataChannel: () => mockChannel,
      setLocalDescription: async () => {},
      createOffer: async () => ({ type: 'offer', sdp: 'mock' }),
      setRemoteDescription: async () => {},
      onicecandidate: null,
      close: () => {},
    };
    
    const originalRTC = global.RTCPeerConnection;
    (global as any).RTCPeerConnection = function() { return mockPC; };
    
    const peer = {
      fingerprint: 'D'.repeat(40),
      address: 'test.example.com:443',
    };
    
    // Start the connection attempt (don't await - it will timeout waiting for channel)
    const _connectPromise = (dht as any).connectHTTP(peer);
    
    // Wait for fetch to be called
    await new Promise(resolve => setTimeout(resolve, 100));
    
    // Verify the HTTP request included pre-gathered candidates (even though connection will timeout)
    expect(capturedBody).toBeDefined();
    expect(capturedBody.candidates).toBeDefined();
    expect(capturedBody.candidates.length).toBe(candidateCount);
    expect(capturedBody.candidates[0].candidate).toContain('192.168.1.100');
    expect(capturedBody.candidates[1].candidate).toContain('203.0.113.1');
    
    // Restore
    global.fetch = originalFetch;
    (global as any).RTCPeerConnection = originalRTC;
    
    // Don't wait for connection to complete (it will timeout)
  });
});
