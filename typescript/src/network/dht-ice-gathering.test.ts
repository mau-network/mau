/**
 * Tests for ICE candidate gathering optimization
 * 
 * Goal: Gather ICE candidates ONCE at startup, reuse for all connections
 */

import { describe, it, expect, beforeEach } from '@jest/globals';
import type { Account } from '../account.js';
import { KademliaDHT } from './dht.js';

// Mock Account
const mockAccount = {
  getFingerprint: () => '0123456789ABCDEF0123456789ABCDEF01234567',
  getPublicKey: () => '-----BEGIN PGP PUBLIC KEY BLOCK-----\n...',
} as Account;

// Mock RTCPeerConnection for testing
class MockRTCPeerConnection {
  localDescription: RTCSessionDescriptionInit | null = null;
  iceGatheringState: RTCIceGatheringState = 'new';
  onicegatheringstatechange: ((ev: Event) => void) | null = null;
  onicecandidate: ((ev: RTCPeerConnectionIceEvent) => void) | null = null;
  
  async createOffer(): Promise<RTCSessionDescriptionInit> {
    return { type: 'offer', sdp: 'mock-sdp' };
  }
  
  async setLocalDescription(desc: RTCSessionDescriptionInit): Promise<void> {
    this.localDescription = desc;
    // Simulate ICE gathering
    setTimeout(() => {
      this.iceGatheringState = 'gathering';
      this.onicegatheringstatechange?.(new Event('icegatheringstatechange'));
      
      // Emit mock ICE candidates
      this.onicecandidate?.({
        candidate: {
          candidate: 'candidate:1 1 UDP 2130706431 192.168.1.100 54321 typ host',
          sdpMLineIndex: 0,
          sdpMid: '0',
        },
      } as RTCPeerConnectionIceEvent);
      
      this.onicecandidate?.({
        candidate: {
          candidate: 'candidate:2 1 UDP 1694498815 203.0.113.1 54322 typ srflx',
          sdpMLineIndex: 0,
          sdpMid: '0',
        },
      } as RTCPeerConnectionIceEvent);
      
      // Signal gathering complete
      setTimeout(() => {
        this.iceGatheringState = 'complete';
        this.onicegatheringstatechange?.(new Event('icegatheringstatechange'));
        this.onicecandidate?.({ candidate: null } as RTCPeerConnectionIceEvent);
      }, 50);
    }, 10);
  }
  
  createDataChannel(_label: string): RTCDataChannel {
    return {
      readyState: 'connecting',
      close: () => {},
    } as RTCDataChannel;
  }
  
  close(): void {}
}

// Mock global RTCPeerConnection
(global as any).RTCPeerConnection = MockRTCPeerConnection;

describe('DHT ICE Gathering', () => {
  let dht: KademliaDHT;
  let pcCreationCount: number;
  let _originalRTCPC: any;

  beforeEach(() => {
    pcCreationCount = 0;
    _originalRTCPC = (global as any).RTCPeerConnection;

    // Wrap to count creations
    (global as any).RTCPeerConnection = class extends MockRTCPeerConnection {
      constructor(..._args: any[]) {
        super();
        pcCreationCount++;
      }
    };
    
    dht = new KademliaDHT(mockAccount);
  });

  it('should gather ICE candidates at startup', async () => {
    // Act
    await (dht as any).gatherICECandidates();
    
    // Assert
    const candidates = (dht as any).localICECandidates;
    expect(candidates).toBeDefined();
    expect(candidates.length).toBeGreaterThan(0);
    expect((dht as any).iceGatheringComplete).toBe(true);
  });

  it('should create only ONE peer connection for ICE gathering', async () => {
    // Act
    await (dht as any).gatherICECandidates();
    
    // Assert - only 1 PC created for gathering
    expect(pcCreationCount).toBe(1);
  });

  it('should timeout ICE gathering after 10 seconds', async () => {
    // Arrange - mock that never completes
    (global as any).RTCPeerConnection = class {
      localDescription = null;
      iceGatheringState = 'new';
      onicegatheringstatechange = null;
      onicecandidate = null;
      
      async createOffer() { return { type: 'offer', sdp: 'mock' }; }
      async setLocalDescription() {
        // Never emit candidates or complete
      }
      createDataChannel() { return { close: () => {} } as any; }
      close() {}
    };
    
    dht = new KademliaDHT(mockAccount);
    
    // Act
    const startTime = Date.now();
    await (dht as any).gatherICECandidates();
    const elapsed = Date.now() - startTime;
    
    // Assert - should timeout around 10s (allow some variance)
    expect(elapsed).toBeGreaterThanOrEqual(9_000);
    expect(elapsed).toBeLessThan(11_000);
    expect((dht as any).iceGatheringComplete).toBe(true); // Still marked complete
  }, 15000); // Increase test timeout to 15s

  it('should have host and srflx candidate types', async () => {
    // Act
    await (dht as any).gatherICECandidates();
    
    // Assert
    const candidates = (dht as any).localICECandidates;
    const hasHost = candidates.some((c: any) => c.candidate.includes('typ host'));
    const hasSrflx = candidates.some((c: any) => c.candidate.includes('typ srflx'));
    
    expect(hasHost).toBe(true);
    expect(hasSrflx).toBe(true);
  });

  it('should close temporary peer connection after gathering', async () => {
    // Arrange
    let closeCalled = false;
    (global as any).RTCPeerConnection = class extends MockRTCPeerConnection {
      close() {
        closeCalled = true;
      }
    };
    
    dht = new KademliaDHT(mockAccount);
    
    // Act
    await (dht as any).gatherICECandidates();
    
    // Assert
    expect(closeCalled).toBe(true);
  });

  it('should gather ICE before connecting to bootstrap peers', async () => {
    // Simplest test: verify ICE is gathered during join()
    
    // Mock lookup and bootstrap discovery to avoid hanging
    (dht as any).lookup = async () => undefined;
    (dht as any).bootstrapDiscovery = async () => {};
    
    // Act - join should gather ICE
    await dht.join([]).catch(() => {});
    
    // Stop timers
    dht.stop();
    
    // Assert - ICE gathering should be complete
    expect((dht as any).iceGatheringComplete).toBe(true);
    expect((dht as any).localICECandidates.length).toBeGreaterThan(0);
  });

  it('should have gathered ICE candidates available for reuse', async () => {
    // This test verifies ICE candidates are available after gathering
    // Later code can embed these in offers instead of gathering per-connection
    
    // Arrange & Act
    await (dht as any).gatherICECandidates();
    
    // Assert - candidates are available for embedding in offers
    const candidates = (dht as any).localICECandidates;
    expect(candidates).toBeInstanceOf(Array);
    expect(candidates.length).toBeGreaterThan(0);
    expect((dht as any).iceGatheringComplete).toBe(true);
    
    // Verify we can access candidate details
    const firstCandidate = candidates[0];
    expect(firstCandidate).toBeDefined();
    expect(firstCandidate.candidate).toBeDefined();
    expect(typeof firstCandidate.candidate).toBe('string');
  });
});
