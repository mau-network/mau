/**
 * Tests for Client
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Client } from './client';
import { Account } from './account';
import { BrowserStorage } from './storage/browser';
import { sign, serializePublicKey } from './crypto/index';

/**
 * Creates a mock fetch that handles the mTLS auth handshake transparently.
 * Auth requests (URLs containing `/auth?challenge=`) are answered by signing
 * the challenge with `peerAccount`'s private key. All other requests return
 * `apiResponse`.
 */
async function createHandshakeFetch(
  peerAccount: Account,
  apiResponse: object,
): Promise<jest.Mock> {
  return jest.fn().mockImplementation(async (url: string) => {
    if (typeof url === 'string' && url.includes('/auth?challenge=')) {
      const challengeHex = new URL('http://x/' + url.split('/auth?challenge=')[1]).searchParams.get('challenge')
        ?? url.split('/auth?challenge=')[1];
      const challenge = new Uint8Array(
        (challengeHex.match(/.{2}/g) as string[]).map(b => parseInt(b, 16))
      );
      const signature = await sign(challenge, peerAccount.getPrivateKey());
      const publicKey = serializePublicKey(peerAccount.getPublicKeyObject());
      return { ok: true, json: async () => ({ publicKey, signature }) };
    }
    return { ok: true, ...apiResponse };
  });
}

const TEST_DIR = 'test-data-client';
const PEER_DIR = 'test-data-client-peer';

describe('Client', () => {
  let storage: BrowserStorage;
  let account: Account;
  let peerAccount: Account;

  beforeEach(async () => {
    storage = await BrowserStorage.create();

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
      await storage.remove(TEST_DIR);
      await storage.remove(PEER_DIR);
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should create client with static resolver', () => {
    const client = account.createClient(
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

    const mockFetch = await createHandshakeFetch(peerAccount, { json: async () => ({ files: [] }) });

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
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    await client.fetchFileList();

    expect(resolverCalled).toBe(true);
    expect(mockFetch).toHaveBeenCalled();
  });

  it('should fetch file list with after parameter', async () => {
    const afterDate = new Date('2024-01-01');

    const mockFetch = await createHandshakeFetch(peerAccount, { json: async () => ({ files: [] }) });

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    await client.fetchFileList(afterDate);

    expect(mockFetch).toHaveBeenCalled();
    // calls[0] is the auth handshake; calls[1] is the actual file list request
    const callOptions = mockFetch.mock.calls[1][1];
    expect(callOptions?.headers?.['If-Modified-Since']).toBe(afterDate.toUTCString());
  });

  it('should download file from peer', async () => {
    const mockData = new Uint8Array([1, 2, 3, 4]);
    const mockFetch = await createHandshakeFetch(peerAccount, { arrayBuffer: async () => mockData.buffer });

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    const data = await client.downloadFile('test.txt');

    expect(data).toEqual(mockData);
  });

  it('should download file version', async () => {
    const mockData = new Uint8Array([5, 6, 7, 8]);
    const mockFetch = await createHandshakeFetch(peerAccount, { arrayBuffer: async () => mockData.buffer });

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    const data = await client.downloadFileVersion('test.txt', 'abc123');

    expect(data).toEqual(mockData);
  });

  it('should throw on HTTP error', async () => {
    const mockFetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: 'Not Found',
    });

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        resolvers: [async () => 'localhost:8080'],
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    await expect(client.fetchFileList()).rejects.toThrow('HTTP 404');
  });

  it('should handle fetch timeout', async () => {
    const mockFetch = jest.fn().mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 200))
    );

    const client = new Client(
      account,
      storage,
      peerAccount.getFingerprint(),
      {
        timeout: 100,
        resolvers: [async () => 'localhost:8080'],
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    await expect(client.fetchFileList()).rejects.toThrow();
  }, 10000);

  it('should try all resolvers until one succeeds', async () => {
    const mockFetch = await createHandshakeFetch(peerAccount, { json: async () => ({ files: [] }) });

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
        fetchImpl: mockFetch as unknown as typeof fetch,
      }
    );

    await client.fetchFileList();

    expect(mockFetch).toHaveBeenCalled();
  });
});
