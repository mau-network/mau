/**
 * KademliaDHT unit tests
 *
 * Tests exercise the public API and the helper logic (routing table, lookup)
 * without requiring live internet connectivity. WebRTC is available via the
 * node-datachannel polyfill set up in jest.setup.ts.
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import * as fs from 'fs/promises';
import { KademliaDHT } from './dht';
import { Account } from '../account';
import { FilesystemStorage } from '../storage/filesystem';

const TEST_DIR = './test-data-dht';

describe('KademliaDHT', () => {
  let storage: FilesystemStorage;
  let account: Account;
  let peerAccount: Account;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

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
    await fs.rm(TEST_DIR, { recursive: true, force: true }).catch(() => {});
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
