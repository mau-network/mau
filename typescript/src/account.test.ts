/**
 * Tests for Account
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Account } from './account';
import { BrowserStorage } from './storage/browser';

const TEST_DIR = 'test-data-account';

describe('Account', () => {
  let storage: BrowserStorage;

  beforeEach(async () => {
    storage = await BrowserStorage.create();
  });

  afterEach(async () => {
    try {
      await storage.remove(TEST_DIR);
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should create a new account', async () => {
    const account = await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
    });

    expect(account.getName()).toBe('Test User');
    expect(account.getEmail()).toBe('test@example.com');
    expect(account.getFingerprint()).toBeTruthy();
    expect(account.getFingerprint().length).toBeGreaterThan(0);
  });

  it('should load an existing account', async () => {
    // Create account
    const created = await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
    });

    const fingerprint = created.getFingerprint();

    // Load account
    const loaded = await Account.load(storage, TEST_DIR, 'test-passphrase');

    expect(loaded.getFingerprint()).toBe(fingerprint);
    expect(loaded.getName()).toBe('Test User');
  });

  it('should throw on duplicate account creation', async () => {
    await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
    });

    await expect(
      Account.create(storage, TEST_DIR, {
        name: 'Another User',
        email: 'another@example.com',
        passphrase: 'another-passphrase',
      })
    ).rejects.toThrow('Account already exists');
  });

  it('should add and list friends', async () => {
    const account = await Account.create(storage, TEST_DIR, {
      name: 'Alice',
      email: 'alice@example.com',
      passphrase: 'alice-pass',
    });

    // Create a friend account
    const friendDir = TEST_DIR + '/friend';
    const friend = await Account.create(storage, friendDir, {
      name: 'Bob',
      email: 'bob@example.com',
      passphrase: 'bob-pass',
    });

    const friendKey = friend.getPublicKey();
    const friendFingerprint = await account.addFriend(friendKey);

    expect(account.getFriends()).toContain(friendFingerprint);
    expect(account.isFriend(friendFingerprint)).toBe(true);
  });
});
