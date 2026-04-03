/**
 * KademliaDHT unit tests
 *
 * Tests exercise the public API and the helper logic (routing table, lookup)
 * without requiring live internet connectivity. WebRTC is available via the
 * node-datachannel polyfill set up in jest.setup.ts.
 */

// TODO(testing): Add regression tests for recent DHT bug fixes
// Per AGENTS.md, these bugs were fixed but lack explicit regression tests:
// 1. Discovery algorithm zero-peers bug (fixed: proper bucket refresh)
// 2. ICE candidate ordering (fixed: candidates sent AFTER SDP answer)
// 3. Connection tie-breaking timeouts (fixed: timeout guards added)
// 4. Memory leak in relay offers (fixed: try/catch with cleanup)
// 5. Bootstrap timer lifecycle (fixed: bootstrapActive flag)
//
// Recommended test cases:
// describe('DHT Regressions', () => {
//   it('should discover peers even when routing table is empty', async () => {
//     // Test for zero-peers bug fix
//   });
//   
//   it('should send ICE candidates AFTER SDP answer', async () => {
//     // Test for ICE ordering fix - verify message sequence
//   });
//   
//   it('should handle simultaneous connection attempts with timeout', async () => {
//     // Test for tie-breaking fix
//   });
//   
//   it('should cleanup relay state on malformed offers', async () => {
//     // Test for memory leak fix - send malformed relay_offer
//   });
//   
//   it('should stop bootstrap timer after reaching DHT_K peers', async () => {
//     // Test for bootstrap lifecycle - verify bootstrapActive flag behavior
//   });
// });
//
// Priority: HIGH - Prevent regressions
// Impact: Confidence in DHT reliability, prevent bug reintroduction

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { KademliaDHT } from './dht';
import { Account } from '../account';
import { BrowserStorage } from '../storage/browser';

const TEST_DIR = 'test-data-dht';

describe('KademliaDHT', () => {
  let storage: BrowserStorage;
  let account: Account;
  let peerAccount: Account;

  beforeAll(async () => {
    storage = await BrowserStorage.create();

    account = await Account.create(storage, TEST_DIR + '/account', {
      name: 'DHT User',
      email: 'dht@test.com',
      passphrase: 'pass',
      algorithm: 'ed25519',
    });

    peerAccount = await Account.create(storage, TEST_DIR + '/peer', {
      name: 'DHT Peer',
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

  it('constructs with default ICE servers', () => {
    const dht = new KademliaDHT(account);
    expect(dht).toBeDefined();
    dht.stop();
  });

  it('constructs with custom ICE servers', () => {
    const dht = new KademliaDHT(account, [{ urls: 'stun:stun.example.com:3478' }]);
    expect(dht).toBeDefined();
    dht.stop();
  });

  it('stop() is idempotent', () => {
    const dht = new KademliaDHT(account);
    dht.stop();
    dht.stop(); // second call should not throw
  });

  it('resolver() returns a FingerprintResolver function', () => {
    const dht = new KademliaDHT(account);
    const resolver = dht.resolver();
    expect(typeof resolver).toBe('function');
    dht.stop();
  });

  it('findAddress returns null when routing table is empty', async () => {
    const dht = new KademliaDHT(account);
    const result = await dht.findAddress(peerAccount.getFingerprint());
    expect(result).toBeNull();
    dht.stop();
  });

  it('resolver() delegates to findAddress', async () => {
    const dht = new KademliaDHT(account);
    const resolver = dht.resolver();
    const result = await resolver(peerAccount.getFingerprint());
    expect(result).toBeNull();
    dht.stop();
  });

  it('join() with empty bootstrap list succeeds', async () => {
    const dht = new KademliaDHT(account);
    await expect(dht.join([])).resolves.toBeUndefined();
    dht.stop();
  });

  it('connectRelay throws when relay is not connected', async () => {
    const dht = new KademliaDHT(account);
    await expect(
      dht.connectRelay(
        { fingerprint: peerAccount.getFingerprint(), address: 'target.example.com:8080' },
        { fingerprint: account.getFingerprint(),     address: 'relay.example.com:8080' },
      )
    ).rejects.toThrow();
    dht.stop();
  });

});
