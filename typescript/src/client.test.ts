/**
 * Tests for Client
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Client } from './client';
import { Account } from './account';
import { FilesystemStorage } from './storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-client';
const PEER_DIR = './test-data-client-peer';

describe('Client', () => {
  let storage: FilesystemStorage;
  let account: Account;
  let peerAccount: Account;

  beforeEach(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });
    await fs.mkdir(PEER_DIR, { recursive: true });

    account = await Account.create(storage, TEST_DIR, {
      name: 'Client User',
      email: 'client@example.com',
      passphrase: 'client-pass',
      algorithm: 'ed25519',
    });

    peerAccount = await Account.create(storage, PEER_DIR, {
      name: 'Peer User',
      email: 'peer@example.com',
      passphrase: 'peer-pass',
      algorithm: 'ed25519',
    });

    // Add peer as friend
    await account.addFriend(peerAccount.getPublicKey());
  });

  afterEach(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
      await fs.rm(PEER_DIR, { recursive: true, force: true });
    } catch (err) { /* cleanup error ignored */ }
  });

  it('should create client with static resolver', () => {
    const client = Client.create(
      account,
      storage,
      {
        fingerprint: peerAccount.getFingerprint(),
        address: 'localhost:8080',
      }
    );

    expect(client).toBeDefined();
  });

  it('should throw PeerNotFoundError when no resolvers', async () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      { resolvers: [] }
    );

    await expect(client.fetchFileList()).rejects.toThrow('Couldn\'t find peer');
  });

  it('should use custom timeout', () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      { timeout: 5000 }
    );

    expect(client).toBeDefined();
  });

  it('should resolve peer address using resolver', async () => {
    let resolverCalled = false;

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [
          async (fingerprint) => {
            resolverCalled = true;
            expect(fingerprint).toBe(peerAccount.getFingerprint());
            return 'localhost:8080';
          },
        ],
      }
    );

    // Mock fetch before creating client
    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ files: [] }),
    });

    global.fetch = mockFetch as any;

    // @ts-expect-error - Set fetchImpl to use our mock
    client['fetchImpl'] = mockFetch as any;

    await client.fetchFileList();

    expect(resolverCalled).toBe(true);
    expect(mockFetch).toHaveBeenCalled();
  });

  it('should fetch file list with after parameter', async () => {
    const afterDate = new Date('2024-01-01');

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
      }
    );

    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ files: [] }),
    });

    // @ts-expect-error - Testing error handling
    client['fetchImpl'] = mockFetch as any;

    await client.fetchFileList(afterDate);

    expect(mockFetch).toHaveBeenCalled();
    const callUrl = mockFetch.mock.calls[0][0];
    expect(callUrl).toContain('after=2024-01-01');
  });

  it('should download file from peer', async () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
      }
    );

    const mockData = new Uint8Array([1, 2, 3, 4]);
    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      arrayBuffer: async () => mockData.buffer,
    });

    // @ts-expect-error - Testing error handling
    client['fetchImpl'] = mockFetch as any;

    const data = await client.downloadFile('test.txt');

    expect(data).toEqual(mockData);
  });

  it('should download file version', async () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
      }
    );

    const mockData = new Uint8Array([5, 6, 7, 8]);
    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      arrayBuffer: async () => mockData.buffer,
    });

    // @ts-expect-error - Testing error handling
    client['fetchImpl'] = mockFetch as any;

    const data = await client.downloadFileVersion('test.txt', 'abc123');

    expect(data).toEqual(mockData);
  });

  it('should throw on HTTP error', async () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
      }
    );

    const mockFetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: 'Not Found',
    });

    // @ts-expect-error - Testing error handling
    client['fetchImpl'] = mockFetch as any;

    await expect(client.fetchFileList()).rejects.toThrow('HTTP 404');
  });

  it('should handle fetch timeout', async () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        timeout: 100,
        resolvers: [async () => 'localhost:8080'],
      }
    );

    const mockFetch = jest.fn().mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 200))
    );

    // @ts-expect-error - Testing error handling
    client['fetchImpl'] = mockFetch as any;

    await expect(client.fetchFileList()).rejects.toThrow();
  }, 10000);

  it('should try all resolvers until one succeeds', async () => {
    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [
          async () => null, // First fails
          async () => null, // Second fails
          async () => 'localhost:8080', // Third succeeds
        ],
      }
    );

    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ files: [] }),
    });

    // @ts-expect-error - Testing error handling
    client['fetchImpl'] = mockFetch as any;

    await client.fetchFileList();

    expect(mockFetch).toHaveBeenCalled();
  });
});
