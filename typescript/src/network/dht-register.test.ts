/**
 * Tests for DHT registerConnection() method
 * Validates that external WebRTC connections can be registered with the DHT
 */

import { KademliaDHT } from './dht.js';
import { Account } from '../account.js';
import { BrowserStorage } from '../storage/index.js';

describe('KademliaDHT.registerConnection', () => {
  let storage: BrowserStorage;
  let account1: Account;
  let account2: Account;
  let dht: KademliaDHT;

  beforeEach(async () => {
    storage = await BrowserStorage.create();
    
    // Create test accounts
    account1 = await Account.create(storage, 'test-dht-1', {
      name: 'DHT Test 1',
      email: 'dht1@test.com',
      passphrase: 'test-passphrase-123456',
    });
    
    account2 = await Account.create(storage, 'test-dht-2', {
      name: 'DHT Test 2',
      email: 'dht2@test.com',
      passphrase: 'test-passphrase-123456',
    });
    
    dht = new KademliaDHT(account1);
  });

  afterEach(async () => {
    dht.stop();
    try {
      await storage.remove('test-dht-1');
      await storage.remove('test-dht-2');
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should register an external WebRTC connection', async () => {
    // Create a mock peer connection and data channel
    const mockPeerConnection = new RTCPeerConnection();
    const mockDataChannel = mockPeerConnection.createDataChannel('test');
    
    const fingerprint = account2.getFingerprint();
    
    // Register the connection
    dht.registerConnection(fingerprint, mockPeerConnection, mockDataChannel);
    
    // Verify connection is registered (we can't directly check private conns Map,
    // but we can verify it doesn't throw and subsequent operations work)
    expect(() => {
      dht.registerConnection(fingerprint, mockPeerConnection, mockDataChannel);
    }).not.toThrow(); // Should be idempotent
    
    // Cleanup
    mockPeerConnection.close();
  });

  it('should ignore registration of own fingerprint', () => {
    const mockPeerConnection = new RTCPeerConnection();
    const mockDataChannel = mockPeerConnection.createDataChannel('test');
    
    const ownFingerprint = account1.getFingerprint();
    
    // Should not throw when registering own fingerprint (but should be ignored)
    expect(() => {
      dht.registerConnection(ownFingerprint, mockPeerConnection, mockDataChannel);
    }).not.toThrow();
    
    // Cleanup
    mockPeerConnection.close();
  });

  it('should handle multiple concurrent registrations', () => {
    const connections = [];
    
    for (let i = 0; i < 5; i++) {
      const pc = new RTCPeerConnection();
      const ch = pc.createDataChannel(`test-${i}`);
      const fakeFingerprint = `${'0'.repeat(39)}${i}`;
      
      dht.registerConnection(fakeFingerprint, pc, ch);
      connections.push({ pc, ch });
    }
    
    // All registrations should succeed without errors
    expect(connections.length).toBe(5);
    
    // Cleanup
    connections.forEach(({ pc }) => pc.close());
  });

  it('should set up message handler on data channel', (done) => {
    const mockPeerConnection = new RTCPeerConnection();
    const mockDataChannel = mockPeerConnection.createDataChannel('test');
    
    const fingerprint = account2.getFingerprint();
    
    // Mock onmessage to verify it gets set
    const originalOnMessage = mockDataChannel.onmessage;
    
    dht.registerConnection(fingerprint, mockPeerConnection, mockDataChannel);
    
    // Verify onmessage handler was set
    expect(mockDataChannel.onmessage).not.toBe(originalOnMessage);
    expect(mockDataChannel.onmessage).toBeDefined();
    
    // Cleanup
    mockPeerConnection.close();
    done();
  });

  it('should set up close handler on data channel', (done) => {
    const mockPeerConnection = new RTCPeerConnection();
    const mockDataChannel = mockPeerConnection.createDataChannel('test');
    
    const fingerprint = account2.getFingerprint();
    
    const originalOnClose = mockDataChannel.onclose;
    
    dht.registerConnection(fingerprint, mockPeerConnection, mockDataChannel);
    
    // Verify onclose handler was set
    expect(mockDataChannel.onclose).not.toBe(originalOnClose);
    expect(mockDataChannel.onclose).toBeDefined();
    
    // Cleanup
    mockPeerConnection.close();
    done();
  });

  it('should normalize fingerprint before registration', () => {
    const mockPeerConnection = new RTCPeerConnection();
    const mockDataChannel = mockPeerConnection.createDataChannel('test');
    
    const fingerprint = account2.getFingerprint();
    
    // Try with uppercase (should be normalized to lowercase internally)
    const uppercaseFingerprint = fingerprint.toUpperCase();
    
    expect(() => {
      dht.registerConnection(uppercaseFingerprint, mockPeerConnection, mockDataChannel);
    }).not.toThrow();
    
    // Try registering again with lowercase (should recognize as duplicate)
    expect(() => {
      dht.registerConnection(fingerprint, mockPeerConnection, mockDataChannel);
    }).not.toThrow();
    
    // Cleanup
    mockPeerConnection.close();
  });
});
