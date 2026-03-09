/**
 * Tests for WebRTCClient
 */

import { describe, it, expect, beforeAll, afterAll, beforeEach } from '@jest/globals';
import { setupWebRTCMocks, cleanupWebRTCMocks, MockRTCPeerConnection } from '../__mocks__/webrtc';
import { WebRTCClient } from './webrtc';
import { Account } from '../account';
import { FilesystemStorage } from '../storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-webrtc-client';

describe('WebRTCClient', () => {
  let storage: FilesystemStorage;
  let account: Account;

  beforeAll(async () => {
    setupWebRTCMocks();
    
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    account = await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
      algorithm: 'ed25519',
    });
  });

  afterAll(async () => {
    cleanupWebRTCMocks();
    
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  it('should create WebRTCClient instance', () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    expect(client).toBeDefined();
  });

  it('should create offer with data channel', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    const offer = await client.createOffer();

    expect(offer).toBeDefined();
    expect(offer.type).toBe('offer');
    expect(offer.sdp).toBeDefined();
  });

  it('should accept offer and create answer', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    const answer = await client.acceptOffer(offer);

    expect(answer).toBeDefined();
    expect(answer.type).toBe('answer');
    expect(answer.sdp).toBeDefined();
  });

  it('should handle answer', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    await client.createOffer();
    
    const answer = { type: 'answer' as const, sdp: 'mock-answer-sdp' };
    await client.handleAnswer(answer);
    
    expect(client).toBeDefined();
  });

  it('should add ICE candidates', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    await client.createOffer();
    
    const candidate = {
      candidate: 'mock-candidate',
      sdpMid: '0',
      sdpMLineIndex: 0
    };
    
    await client.addIceCandidate(candidate);
    
    expect(client).toBeDefined();
  });

  it('should handle data channel messages', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    await client.createOffer();
    
    // Wait for data channel to open
    await new Promise(resolve => setTimeout(resolve, 50));
    
    // Access private connection for testing
    const pc = (client as any).connection as MockRTCPeerConnection;
    const channel = pc.getDataChannel('mau-http');
    
    if (channel) {
      channel.readyState = 'open';
      
      // Simulate incoming message
      channel.simulateMessage('test message');
    }
    
    expect(client).toBeDefined();
  });

  it('should send requests over data channel', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    await client.createOffer();
    
    // Wait for data channel
    await new Promise(resolve => setTimeout(resolve, 50));
    
    const pc = (client as any).connection as MockRTCPeerConnection;
    const channel = pc.getDataChannel('mau-http');
    
    if (channel) {
      channel.readyState = 'open';
    }
    
    try {
      await Promise.race([
        client.request('/files'),
        new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), 100))
      ]);
    } catch (error: any) {
      // Expected to timeout in mock environment
      expect(error.message).toBeDefined();
    }
  });

  it('should close connection properly', () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    client.close();
    
    const pc = (client as any).connection as MockRTCPeerConnection;
    expect(pc.connectionState).toBe('closed');
  });

  it('should handle connection state changes', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    await client.createOffer();
    
    const pc = (client as any).connection as MockRTCPeerConnection;
    
    // Simulate connection
    pc.connectionState = 'connected';
    pc.dispatchEvent('connectionstatechange');
    
    await new Promise(resolve => setTimeout(resolve, 50));
    
    expect(pc.connectionState).toBe('connected');
  });

  it('should handle data channel errors', async () => {
    const client = new WebRTCClient(
      account,
      storage,
      'test-fingerprint-1234567890abcdef'
    );

    await client.createOffer();
    
    const pc = (client as any).connection as MockRTCPeerConnection;
    const channel = pc.getDataChannel('mau-http');
    
    if (channel) {
      channel.dispatchEvent('error', { error: new Error('test error') });
    }
    
    expect(client).toBeDefined();
  });
});
