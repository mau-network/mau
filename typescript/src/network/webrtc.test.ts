/**
 * WebRTCClient unit tests — error paths that don't need a live connection
 *
 * Full connection tests (createOffer, acceptOffer, mTLS, sendRequest) are
 * covered by webrtc-advanced.test.ts which uses the @roamhq/wrtc polyfill.
 * This file covers the "not connected" guard clauses without touching native
 * WebRTC bindings (avoids conflicts between the two RTCPeerConnection impls).
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { WebRTCClient } from './webrtc';
import { Account } from '../account';
import { BrowserStorage } from '../storage/browser';

const TEST_DIR = 'test-data-webrtc-unit';

describe('WebRTCClient', () => {
  let storage: BrowserStorage;
  let account: Account;
  let peerAccount: Account;

  beforeAll(async () => {
    storage = await BrowserStorage.create();

    account = await Account.create(storage, TEST_DIR + '/account', {
      name: 'WRTCUser',
      email: 'wrtc@test.com',
      passphrase: 'pass',
      algorithm: 'ed25519',
    });

    peerAccount = await Account.create(storage, TEST_DIR + '/peer', {
      name: 'WRTCPeer',
      email: 'peer@test.com',
      passphrase: 'pass',
      algorithm: 'ed25519',
    });
  });

  afterAll(async () => {
    try {
      await storage.remove(TEST_DIR);
    } catch {
      // Ignore cleanup errors
    }
  });

  it('constructs with defaults', () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    expect(client).toBeDefined();
  });

  it('constructs with custom config', () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint(), {
      iceServers: [{ urls: 'stun:stun.example.com:3478' }],
      timeout: 5000,
    });
    expect(client).toBeDefined();
  });

  it('close() is safe before connecting', () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    expect(() => client.close()).not.toThrow();
  });

  it('completeConnection() throws when no connection exists', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(
      client.completeConnection({ type: 'answer', sdp: 'fake' })
    ).rejects.toThrow('No connection');
  });

  it('addIceCandidate() throws when no connection exists', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(
      client.addIceCandidate({ candidate: '' })
    ).rejects.toThrow('No connection');
  });

  it('performMTLS() throws when data channel not ready', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(client.performMTLS()).rejects.toThrow('Data channel not ready');
  });

  it('sendRequest() throws when data channel not ready', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(
      client.sendRequest({ method: 'GET', path: '/test' })
    ).rejects.toThrow('Data channel not ready');
  });

  it('fetchFileList() throws when data channel not ready', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(client.fetchFileList()).rejects.toThrow('Data channel not ready');
  });

  it('downloadFile() throws when data channel not ready', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(client.downloadFile('test.txt')).rejects.toThrow('Data channel not ready');
  });

  it('downloadFileVersion() throws when data channel not ready', async () => {
    const client = new WebRTCClient(account, storage, peerAccount.getFingerprint());
    await expect(
      client.downloadFileVersion('test.txt', 'v1')
    ).rejects.toThrow('Data channel not ready');
  });
});
