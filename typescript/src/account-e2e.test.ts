/**
 * Account E2E Tests - Friends, sync state, directories
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { Account } from './account';
import { FilesystemStorage } from './storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-account-e2e';

describe('Account E2E Tests', () => {
  let storage: FilesystemStorage;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });
  });

  afterAll(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  describe('Account Creation', () => {
    it('should create Ed25519 account', async () => {
      const account = await Account.create(storage, TEST_DIR + '/ed25519', {
        name: 'Test User',
        email: 'test@example.com',
        passphrase: 'test-passphrase',
        algorithm: 'ed25519',
      });

      expect(account).toBeDefined();
      expect(account.getName()).toBe('Test User');
      expect(account.getEmail()).toBe('test@example.com');
      expect(account.getFingerprint()).toBeDefined();
      expect(account.getFingerprint().length).toBeGreaterThan(0);
    });

    it('should create RSA account', async () => {
      const account = await Account.create(storage, TEST_DIR + '/rsa', {
        name: 'RSA User',
        email: 'rsa@example.com',
        passphrase: 'rsa-pass',
        algorithm: 'rsa',
        // bits: 2048,
      });

      expect(account).toBeDefined();
      expect(account.getFingerprint()).toBeDefined();
    }, 30000);

    it('should generate different fingerprints', async () => {
      const account1 = await Account.create(storage, TEST_DIR + '/user1', {
        name: 'User 1',
        email: 'user1@example.com',
        passphrase: 'pass1',
        algorithm: 'ed25519',
      });

      const account2 = await Account.create(storage, TEST_DIR + '/user2', {
        name: 'User 2',
        email: 'user2@example.com',
        passphrase: 'pass2',
        algorithm: 'ed25519',
      });

      expect(account1.getFingerprint()).not.toBe(account2.getFingerprint());
    });
  });

  describe('Account Loading', () => {
    it('should load existing account', async () => {
      const created = await Account.create(storage, TEST_DIR + '/persistent', {
        name: 'Persistent User',
        email: 'persist@example.com',
        passphrase: 'persist-pass',
        algorithm: 'ed25519',
      });

      const createdFingerprint = created.getFingerprint();

      const loaded = await Account.load(storage, TEST_DIR + '/persistent', 'persist-pass');

      expect(loaded.getFingerprint()).toBe(createdFingerprint);
      expect(loaded.getName()).toBe('Persistent User');
      expect(loaded.getEmail()).toBe('persist@example.com');
    });

    it('should fail with wrong passphrase', async () => {
      await Account.create(storage, TEST_DIR + '/protected', {
        name: 'Protected',
        email: 'protected@example.com',
        passphrase: 'correct-pass',
        algorithm: 'ed25519',
      });

      await expect(
        Account.load(storage, TEST_DIR + '/protected', 'wrong-pass')
      ).rejects.toThrow();
    });
  });

  describe('Friends Management', () => {
    let account: Account;
    let friend: Account;

    beforeAll(async () => {
      account = await Account.create(storage, TEST_DIR + '/main', {
        name: 'Main User',
        email: 'main@example.com',
        passphrase: 'main-pass',
        algorithm: 'ed25519',
      });

      friend = await Account.create(storage, TEST_DIR + '/friend', {
        name: 'Friend',
        email: 'friend@example.com',
        passphrase: 'friend-pass',
        algorithm: 'ed25519',
      });
    });

    it('should start with no friends', () => {
      const friends = account.getFriends();
      expect(friends).toEqual([]);
    });

    it('should add friend', async () => {
      const friendPublicKey = friend.getPublicKey();
      const fingerprint = await account.addFriend(friendPublicKey);

      expect(fingerprint).toBe(friend.getFingerprint());

      const friends = account.getFriends();
      expect(friends).toContain(friend.getFingerprint());
    });

    it('should check if is friend', async () => {
      const isFriend = account.isFriend(friend.getFingerprint());
      expect(isFriend).toBe(true);

      const isNotFriend = account.isFriend('unknown-fingerprint');
      expect(isNotFriend).toBe(false);
    });

    it('should get friend key', async () => {
      const friendKey = account.getFriendKey(friend.getFingerprint());
      expect(friendKey).toBeDefined();
    });

    it('should remove friend', async () => {
      await account.removeFriend(friend.getFingerprint());

      const friends = account.getFriends();
      expect(friends).not.toContain(friend.getFingerprint());

      const isFriend = account.isFriend(friend.getFingerprint());
      expect(isFriend).toBe(false);
    });

    it('should handle multiple friends', async () => {
      const friend2 = await Account.create(storage, TEST_DIR + '/friend2', {
        name: 'Friend 2',
        email: 'friend2@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      const friend3 = await Account.create(storage, TEST_DIR + '/friend3', {
        name: 'Friend 3',
        email: 'friend3@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      await account.addFriend(friend.getPublicKey());
      await account.addFriend(friend2.getPublicKey());
      await account.addFriend(friend3.getPublicKey());

      const friends = account.getFriends();
      expect(friends.length).toBe(3);
      expect(friends).toContain(friend.getFingerprint());
      expect(friends).toContain(friend2.getFingerprint());
      expect(friends).toContain(friend3.getFingerprint());
    });
  });

  describe('Sync State', () => {
    let account: Account;
    let friend: Account;

    beforeAll(async () => {
      account = await Account.create(storage, TEST_DIR + '/sync', {
        name: 'Sync User',
        email: 'sync@example.com',
        passphrase: 'sync-pass',
        algorithm: 'ed25519',
      });

      friend = await Account.create(storage, TEST_DIR + '/sync-friend', {
        name: 'Sync Friend',
        email: 'sync-friend@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      await account.addFriend(friend.getPublicKey());
    });

    it('should get initial sync state', async () => {
      const syncState = await account.getSyncState();
      expect(syncState).toBeDefined();
      expect(typeof syncState).toBe('object');
    });

    it('should update sync state', async () => {
      const timestamp = Date.now();
      await account.updateSyncState(friend.getFingerprint(), timestamp);

      const syncState = await account.getSyncState();
      expect(syncState[friend.getFingerprint()]).toBe(timestamp);
    });

    it('should handle multiple sync states', async () => {
      const friend2 = await Account.create(storage, TEST_DIR + '/sync-friend2', {
        name: 'Sync Friend 2',
        email: 'sync-friend2@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      await account.addFriend(friend2.getPublicKey());

      const timestamp1 = Date.now();
      const timestamp2 = timestamp1 + 1000;

      await account.updateSyncState(friend.getFingerprint(), timestamp1);
      await account.updateSyncState(friend2.getFingerprint(), timestamp2);

      const syncState = await account.getSyncState();
      expect(syncState[friend.getFingerprint()]).toBe(timestamp1);
      expect(syncState[friend2.getFingerprint()]).toBe(timestamp2);
    });
  });

  describe('Directories', () => {
    let account: Account;
    let friend: Account;

    beforeAll(async () => {
      account = await Account.create(storage, TEST_DIR + '/dirs', {
        name: 'Dir User',
        email: 'dirs@example.com',
        passphrase: 'dirs-pass',
        algorithm: 'ed25519',
      });

      friend = await Account.create(storage, TEST_DIR + '/dirs-friend', {
        name: 'Dir Friend',
        email: 'dirs-friend@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });
    });

    it('should get mau directory', () => {
      const mauDir = account.getMauDir();
      expect(mauDir).toBeDefined();
      expect(mauDir).toContain('dirs');
    });

    it('should get content directory', () => {
      const contentDir = account.getContentDir();
      expect(contentDir).toBeDefined();
      expect(contentDir).toContain('dirs');
    });

    it('should get friend content directory', () => {
      const friendDir = account.getFriendContentDir(friend.getFingerprint());
      expect(friendDir).toBeDefined();
      expect(friendDir).toContain(friend.getFingerprint());
    });

    it('should have different directories for different friends', async () => {
      const friend2 = await Account.create(storage, TEST_DIR + '/dirs-friend2', {
        name: 'Dir Friend 2',
        email: 'dirs-friend2@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      const friendDir1 = account.getFriendContentDir(friend.getFingerprint());
      const friendDir2 = account.getFriendContentDir(friend2.getFingerprint());

      expect(friendDir1).not.toBe(friendDir2);
    });
  });

  describe('Keys', () => {
    let account: Account;

    beforeAll(async () => {
      account = await Account.create(storage, TEST_DIR + '/keys', {
        name: 'Key User',
        email: 'keys@example.com',
        passphrase: 'keys-pass',
        algorithm: 'ed25519',
      });
    });

    it('should get public key', () => {
      const publicKey = account.getPublicKey();
      expect(publicKey).toBeDefined();
      expect(publicKey.length).toBeGreaterThan(0);
      expect(publicKey).toContain('BEGIN PGP PUBLIC KEY BLOCK');
    });

    it('should get consistent public key', () => {
      const key1 = account.getPublicKey();
      const key2 = account.getPublicKey();
      expect(key1).toBe(key2);
    });

    it('should get private key object', () => {
      const privateKey = account.getPrivateKey();
      expect(privateKey).toBeDefined();
    });

    it('should get public key object', () => {
      const publicKey = account.getPublicKeyObject();
      expect(publicKey).toBeDefined();
    });

    it('should get all public keys', () => {
      const allKeys = account.getAllPublicKeys();
      expect(Array.isArray(allKeys)).toBe(true);
      expect(allKeys.length).toBeGreaterThan(0);
    });
  });

  describe('Metadata', () => {
    it('should get name', async () => {
      const account = await Account.create(storage, TEST_DIR + '/meta-name', {
        name: 'Meta Name',
        email: 'meta@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      expect(account.getName()).toBe('Meta Name');
    });

    it('should get email', async () => {
      const account = await Account.create(storage, TEST_DIR + '/meta-email', {
        name: 'User',
        email: 'specific@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      expect(account.getEmail()).toBe('specific@example.com');
    });

    it('should get consistent fingerprint', async () => {
      const account = await Account.create(storage, TEST_DIR + '/meta-finger', {
        name: 'User',
        email: 'user@example.com',
        passphrase: 'pass',
        algorithm: 'ed25519',
      });

      const finger1 = account.getFingerprint();
      const finger2 = account.getFingerprint();

      expect(finger1).toBe(finger2);
    });
  });
});
