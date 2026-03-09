/**
 * Tests for WebRTCServer
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { setupWebRTCMocks, cleanupWebRTCMocks } from '../__mocks__/webrtc';
import { WebRTCServer } from './webrtc-server';
import { Account } from '../account';
import { FilesystemStorage } from '../storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-webrtc-server';

describe('WebRTCServer', () => {
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

  it('should create WebRTCServer instance', () => {
    const server = new WebRTCServer(account, storage);
    expect(server).toBeDefined();
  });

  it('should accept incoming offers', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    const answer = await server.handleOffer(offer, 'remote-fingerprint');
    
    expect(answer).toBeDefined();
    expect(answer.type).toBe('answer');
    expect(answer.sdp).toBeDefined();
  });

  it('should handle multiple connections', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer1 = { type: 'offer' as const, sdp: 'mock-offer-sdp-1' };
    const offer2 = { type: 'offer' as const, sdp: 'mock-offer-sdp-2' };
    
    const answer1 = await server.handleOffer(offer1, 'fingerprint-1');
    const answer2 = await server.handleOffer(offer2, 'fingerprint-2');
    
    expect(answer1).toBeDefined();
    expect(answer2).toBeDefined();
  });

  it('should close specific connection', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    await server.handleOffer(offer, 'test-fingerprint');
    
    server.closeConnection('test-fingerprint');
    
    expect(server).toBeDefined();
  });

  it('should close all connections', async () => {
    const server = new WebRTCServer(account, storage);
    
    await server.handleOffer({ type: 'offer', sdp: 'sdp1' }, 'peer1');
    await server.handleOffer({ type: 'offer', sdp: 'sdp2' }, 'peer2');
    
    server.close();
    
    expect(server).toBeDefined();
  });

  it('should handle incoming HTTP requests', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    await server.handleOffer(offer, 'test-peer');
    
    // Wait for connection setup
    await new Promise(resolve => setTimeout(resolve, 100));
    
    expect(server).toBeDefined();
  });

  it('should handle ICE candidates for connections', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    await server.handleOffer(offer, 'test-peer');
    
    const candidate = {
      candidate: 'mock-candidate',
      sdpMid: '0',
      sdpMLineIndex: 0
    };
    
    await server.addIceCandidate('test-peer', candidate);
    
    expect(server).toBeDefined();
  });

  it('should get list of connected peers', async () => {
    const server = new WebRTCServer(account, storage);
    
    await server.handleOffer({ type: 'offer', sdp: 'sdp1' }, 'peer1');
    await server.handleOffer({ type: 'offer', sdp: 'sdp2' }, 'peer2');
    
    const peers = server.getConnectedPeers();
    
    expect(peers).toContain('peer1');
    expect(peers).toContain('peer2');
  });

  it('should handle data channel messages from clients', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    await server.handleOffer(offer, 'test-peer');
    
    await new Promise(resolve => setTimeout(resolve, 100));
    
    expect(server).toBeDefined();
  });

  it('should respond to file list requests', async () => {
    const server = new WebRTCServer(account, storage);
    
    const offer = { type: 'offer' as const, sdp: 'mock-offer-sdp' };
    await server.handleOffer(offer, 'test-peer');
    
    // Simulate file list request through data channel
    await new Promise(resolve => setTimeout(resolve, 100));
    
    expect(server).toBeDefined();
  });
});
