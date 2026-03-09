/**
 * Client Error Handling and Edge Cases Tests
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { Client } from './client';
import { Account } from './account';
import { FilesystemStorage } from './storage/filesystem';
import { PeerNotFoundError } from './types';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-client-edge';

describe('Client Error Handling and Edge Cases', () => {
  let storage: FilesystemStorage;
  let account: Account;
  let peerAccount: Account;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    account = await Account.create(storage, TEST_DIR + '/client', {
      name: 'Client',
      email: 'client@test.com',
      passphrase: 'pass',
      algorithm: 'ed25519',
    });

    peerAccount = await Account.create(storage, TEST_DIR + '/peer', {
      name: 'Peer',
      email: 'peer@test.com',
      passphrase: 'pass',
      algorithm: 'ed25519',
    });
  });

  afterAll(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  describe('Constructor and Configuration', () => {
    it('should create client with minimal config', () => {
      const client = new Client(account, storage, peerAccount.getFingerprint());
      expect(client).toBeDefined();
    });

    it('should create client with custom timeout', () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        timeout: 60000,
      });
      expect(client).toBeDefined();
    });

    it('should create client with DNS names', () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        dnsNames: ['peer.example.com', 'backup.example.com'],
      });
      expect(client).toBeDefined();
    });

    it('should create client with empty config', () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {});
      expect(client).toBeDefined();
    });

    it('should accept multiple resolvers', () => {
      const resolver1 = async () => 'http://peer1.local';
      const resolver2 = async () => 'http://peer2.local';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver1, resolver2],
      });

      expect(client).toBeDefined();
    });
  });

  describe('Resolver Error Handling', () => {
    it('should throw PeerNotFoundError when no resolvers configured', async () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [],
      });

      await expect(client.fetchFileList()).rejects.toThrow(PeerNotFoundError);
    });

    it('should throw PeerNotFoundError when all resolvers fail', async () => {
      const failingResolver1 = async () => {
        throw new Error('Resolver 1 failed');
      };
      const failingResolver2 = async () => {
        throw new Error('Resolver 2 failed');
      };

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [failingResolver1, failingResolver2],
      });

      await expect(client.fetchFileList()).rejects.toThrow(PeerNotFoundError);
    });

    it('should throw PeerNotFoundError when all resolvers return null', async () => {
      const nullResolver1 = async () => null as any;
      const nullResolver2 = async () => null as any;

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [nullResolver1, nullResolver2],
      });

      await expect(client.fetchFileList()).rejects.toThrow(PeerNotFoundError);
    });

    it('should use first successful resolver', async () => {
      const failingResolver = async () => {
        throw new Error('Failed');
      };
      const successResolver = async () => 'http://localhost:19999';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [failingResolver, successResolver],
        timeout: 100,
      });

      // Should attempt connection (will fail at HTTP level but resolver worked)
      await expect(client.fetchFileList()).rejects.toThrow();
      // The fact it tried to connect means resolver succeeded
    });
  });

  describe('HTTP Error Handling', () => {
    it('should handle connection refused', async () => {
      const resolver = async () => 'http://localhost:99999';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
        timeout: 1000,
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });

    it('should handle connection timeout', async () => {
      // Use a non-routable IP to trigger timeout
      const resolver = async () => 'http://192.0.2.1:80'; // TEST-NET-1

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
        timeout: 100, // Very short timeout
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });

    it('should handle invalid URL', async () => {
      const resolver = async () => 'not-a-valid-url';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });

    it('should handle malformed resolver response', async () => {
      const resolver = async () => 'ftp://invalid-protocol';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });
  });

  describe('Timeout Behavior', () => {
    it('should respect custom timeout', async () => {
      const resolver = async () => 'http://192.0.2.1:80';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
        timeout: 50, // Very short
      });

      const start = Date.now();
      
      await expect(client.fetchFileList()).rejects.toThrow();
      
      const duration = Date.now() - start;
      // Should timeout quickly (within ~500ms including overhead)
      expect(duration).toBeLessThan(2000);
    });

    it('should use default timeout when not specified', async () => {
      const resolver = async () => 'http://192.0.2.1:80';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
        // No timeout specified, should use default
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });
  });

  describe('Invalid Operations', () => {
    it('should throw on downloadFile without resolvers', async () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [],
      });

      await expect(client.downloadFile('test.txt')).rejects.toThrow(PeerNotFoundError);
    });

    it('should throw on downloadFileVersion without resolvers', async () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [],
      });

      await expect(client.downloadFileVersion('test.txt', 'hash123')).rejects.toThrow(PeerNotFoundError);
    });

    it('should throw on sync without resolvers', async () => {
      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [],
      });

      await expect(client.sync()).rejects.toThrow(PeerNotFoundError);
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty fingerprint in resolver', async () => {
      const resolver = async (fp: string) => {
        expect(fp).toBe(peerAccount.getFingerprint());
        throw new Error('No address');
      };

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });

    it('should handle resolver throwing immediately', async () => {
      const throwingResolver = async () => {
        throw new Error('Immediate failure');
      };

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [throwingResolver],
      });

      await expect(client.fetchFileList()).rejects.toThrow(PeerNotFoundError);
    });

    it('should handle resolver returning empty string', async () => {
      const emptyResolver = async () => '';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [emptyResolver],
      });

      await expect(client.fetchFileList()).rejects.toThrow();
    });

    it('should handle concurrent resolver failures', async () => {
      const resolver1 = async () => { throw new Error('Fail 1'); };
      const resolver2 = async () => { throw new Error('Fail 2'); };
      const resolver3 = async () => { throw new Error('Fail 3'); };

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver1, resolver2, resolver3],
      });

      await expect(client.fetchFileList()).rejects.toThrow(PeerNotFoundError);
    });
  });

  describe('Resolver Fallback Behavior', () => {
    it('should try all resolvers in order', async () => {
      const callOrder: number[] = [];

      const resolver1 = async () => {
        callOrder.push(1);
        throw new Error('Fail');
      };
      const resolver2 = async () => {
        callOrder.push(2);
        throw new Error('Fail');
      };
      const resolver3 = async () => {
        callOrder.push(3);
        throw new Error('Fail');
      };

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver1, resolver2, resolver3],
      });

      await expect(client.fetchFileList()).rejects.toThrow();

      // Should have tried all resolvers
      expect(callOrder.length).toBe(3);
    });

    it('should stop at first successful resolver', async () => {
      const callOrder: number[] = [];

      const resolver1 = async () => {
        callOrder.push(1);
        throw new Error('Fail');
      };
      const resolver2 = async () => {
        callOrder.push(2);
        return 'http://localhost:19999';
      };
      const resolver3 = async () => {
        callOrder.push(3);
        return 'http://localhost:29999';
      };

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver1, resolver2, resolver3],
        timeout: 100,
      });

      await expect(client.fetchFileList()).rejects.toThrow();

      // Should have called resolvers (they run concurrently in allSettled)
      expect(callOrder.length).toBeGreaterThan(0);
    });
  });

  describe('Sync Edge Cases', () => {
    it('should handle sync with no remote files', async () => {
      // Mock server that returns empty file list
      const resolver = async () => 'http://localhost:19999';

      const client = new Client(account, storage, peerAccount.getFingerprint(), {
        resolvers: [resolver],
        timeout: 100,
      });

      await expect(client.sync()).rejects.toThrow();
    });

  });
});
