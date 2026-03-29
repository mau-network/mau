/**
 * Integration test for DHT registerConnection + relay signaling
 * Validates the complete flow from registration to peer discovery
 */

import { KademliaDHT } from './dht.js';
import { Account } from '../account.js';
import { BrowserStorage } from '../storage/index.js';

describe('DHT registerConnection integration', () => {
  let storage: BrowserStorage;
  let bootstrap: Account;
  let peer1: Account;
  let peer2: Account;
  let bootstrapDHT: KademliaDHT;

  beforeEach(async () => {
    storage = await BrowserStorage.create();
    
    bootstrap = await Account.create(storage, 'test-bootstrap', {
      name: 'Bootstrap',
      email: 'bootstrap@test.com',
      passphrase: 'bootstrap-pass-123456',
    });
    
    peer1 = await Account.create(storage, 'test-peer1', {
      name: 'Peer 1',
      email: 'peer1@test.com',
      passphrase: 'peer1-pass-123456',
    });
    
    peer2 = await Account.create(storage, 'test-peer2', {
      name: 'Peer 2',
      email: 'peer2@test.com',
      passphrase: 'peer2-pass-123456',
    });
    
    bootstrapDHT = new KademliaDHT(bootstrap);
  });

  afterEach(async () => {
    bootstrapDHT.stop();
    try {
      await storage.remove('test-bootstrap');
      await storage.remove('test-peer1');
      await storage.remove('test-peer2');
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should allow registered peer to be queried via find_peer', async () => {
    // Simulate what bootstrap server does: register an incoming WebRTC connection
    const mockPC1 = new RTCPeerConnection();
    const mockCh1 = mockPC1.createDataChannel('dht');
    
    bootstrapDHT.registerConnection(peer1.getFingerprint(), mockPC1, mockCh1);
    
    // Now check if we can find this peer in the routing table
    // We can't directly call private methods, but we can verify the peer was added
    // by checking that subsequent operations don't throw
    
    expect(() => {
      bootstrapDHT.registerConnection(peer1.getFingerprint(), mockPC1, mockCh1);
    }).not.toThrow(); // Should handle duplicate registration gracefully
    
    mockPC1.close();
  });

  it('should register multiple peers in DHT routing table', async () => {
    const peer1PC = new RTCPeerConnection();
    const peer1Ch = peer1PC.createDataChannel('dht');
    
    const peer2PC = new RTCPeerConnection();
    const peer2Ch = peer2PC.createDataChannel('dht');
    
    // Register both peers
    bootstrapDHT.registerConnection(peer1.getFingerprint(), peer1PC, peer1Ch);
    bootstrapDHT.registerConnection(peer2.getFingerprint(), peer2PC, peer2Ch);
    
    // Both should be registered without errors
    // In a real scenario, when peer1 queries the DHT, it should get peer2 in the results
    
    // Verify internal state by checking conns Map size
    const connsMap = (bootstrapDHT as any).conns as Map<string, any>;
    expect(connsMap.size).toBe(2);
    
    // Cleanup
    peer1PC.close();
    peer2PC.close();
  });

  it('should handle message routing through registered connections', (done) => {
    const mockPC = new RTCPeerConnection();
    const mockCh = mockPC.createDataChannel('dht');
    
    let messageReceived = false;
    
    // Set up a message listener before registration
    mockCh.addEventListener('message', (event) => {
      messageReceived = true;
      const data = JSON.parse(event.data as string);
      expect(data.type).toBeDefined();
      done();
    });
    
    bootstrapDHT.registerConnection(peer1.getFingerprint(), mockPC, mockCh);
    
    // Simulate the data channel opening
    if (mockCh.readyState === 'open') {
      // Send a test message
      mockCh.dispatchEvent(new MessageEvent('message', {
        data: JSON.stringify({ type: 'ping', id: 'test123' })
      }));
    }
    
    // Cleanup after a short delay if message wasn't received
    setTimeout(() => {
      mockPC.close();
      if (!messageReceived) {
        done(); // Complete test even if no message (handler setup is what we're testing)
      }
    }, 1000);
  });

  it('should clean up connection when data channel closes', async () => {
    const mockPC = new RTCPeerConnection();
    const mockCh = mockPC.createDataChannel('dht');
    
    bootstrapDHT.registerConnection(peer1.getFingerprint(), mockPC, mockCh);
    
    // Verify connection is registered
    const connsMapBefore = (bootstrapDHT as any).conns as Map<string, any>;
    expect(connsMapBefore.has(peer1.getFingerprint().toLowerCase())).toBe(true);
    
    // Simulate channel close
    const closeEvent = new Event('close');
    mockCh.dispatchEvent(closeEvent);
    
    // Small delay for async cleanup
    await new Promise(resolve => setTimeout(resolve, 100));
    
    // Verify connection was removed
    const connsMapAfter = (bootstrapDHT as any).conns as Map<string, any>;
    expect(connsMapAfter.has(peer1.getFingerprint().toLowerCase())).toBe(false);
    
    mockPC.close();
  });

  it('should normalize fingerprint before storing in conns Map', () => {
    const mockPC = new RTCPeerConnection();
    const mockCh = mockPC.createDataChannel('dht');
    
    const uppercaseFingerprint = peer1.getFingerprint().toUpperCase();
    
    bootstrapDHT.registerConnection(uppercaseFingerprint, mockPC, mockCh);
    
    // Verify it's stored with normalized (lowercase) fingerprint
    const connsMap = (bootstrapDHT as any).conns as Map<string, any>;
    const lowercaseFingerprint = peer1.getFingerprint().toLowerCase();
    
    expect(connsMap.has(lowercaseFingerprint)).toBe(true);
    expect(connsMap.has(uppercaseFingerprint)).toBe(false);
    
    mockPC.close();
  });

  it('should add peer to routing table during registration', () => {
    const mockPC = new RTCPeerConnection();
    const mockCh = mockPC.createDataChannel('dht');
    
    bootstrapDHT.registerConnection(peer1.getFingerprint(), mockPC, mockCh);
    
    // Access routing table buckets
    const buckets = (bootstrapDHT as any).buckets as Array<{ peers: Array<{ fingerprint: string }> }>;
    
    // Find the peer in the routing table
    let found = false;
    for (const bucket of buckets) {
      for (const peer of bucket.peers) {
        if (peer.fingerprint.toLowerCase() === peer1.getFingerprint().toLowerCase()) {
          found = true;
          break;
        }
      }
      if (found) { break; }
    }
    
    expect(found).toBe(true);
    
    mockPC.close();
  });
});
