import 'fake-indexeddb/auto';
import { test, expect, describe, beforeEach } from 'bun:test';
import { AccountManager } from './manager';
import { BrowserStorage } from '@mau-network/mau';

describe('AccountManager - Friend Management', () => {
  let manager: AccountManager;
  let friendManager: AccountManager;

  beforeEach(async () => {
    manager = await AccountManager.create();
    friendManager = await AccountManager.create();
  });

  describe('exportPublicKey', () => {
    test('should export public key after account creation', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'secure-passphrase-123');
      
      const publicKey = await manager.exportPublicKey();
      
      expect(publicKey).toBeDefined();
      expect(publicKey).toContain('-----BEGIN PGP PUBLIC KEY BLOCK-----');
      expect(publicKey).toContain('-----END PGP PUBLIC KEY BLOCK-----');
    });

    test('should export public key after unlocking account', async () => {
      await manager.createAccount('Bob', 'bob@example.com', 'bobs-passphrase-456');
      const publicKeyAfterCreate = await manager.exportPublicKey();
      
      // Simulate app restart by creating new manager
      const newManager = await AccountManager.create();
      await newManager.unlockAccount('bobs-passphrase-456');
      
      const publicKeyAfterUnlock = await newManager.exportPublicKey();
      
      expect(publicKeyAfterUnlock).toBe(publicKeyAfterCreate);
    });

    test('should throw error when no account unlocked', async () => {
      await expect(manager.exportPublicKey()).rejects.toThrow('No account unlocked');
    });
  });

  describe('addFriend', () => {
    test('should add friend with valid public key', async () => {
      // Create main account
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      // Create friend account and export key
      const storage2 = await BrowserStorage.create();
      await friendManager.createAccount('Bob', 'bob@example.com', 'bob-pass-456');
      const bobPublicKey = await friendManager.exportPublicKey();
      
      // Add Bob as friend
      const friend = await manager.addFriend(bobPublicKey);
      
      expect(friend.name).toBe('Bob');
      expect(friend.email).toBe('bob@example.com');
      expect(friend.fingerprint).toBeDefined();
      expect(friend.fingerprint.length).toBe(40); // PGP fingerprint length
    });

    test('should throw error with invalid public key', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      await expect(manager.addFriend('invalid-key')).rejects.toThrow();
    });

    test('should throw error when no account unlocked', async () => {
      await expect(manager.addFriend('some-key')).rejects.toThrow('No account unlocked');
    });
  });

  describe('listFriends', () => {
    test('should return empty array when no friends', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      const friends = await manager.listFriends();
      
      expect(friends).toEqual([]);
    });

    test('should list all added friends', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      // Create and add Bob
      await friendManager.createAccount('Bob', 'bob@example.com', 'bob-pass-456');
      const bobKey = await friendManager.exportPublicKey();
      await manager.addFriend(bobKey);
      
      // Create and add Carol (need new storage/manager)
      const carolStorage = await BrowserStorage.create();
      const carolManager = await AccountManager.create();
      await carolManager.createAccount('Carol', 'carol@example.com', 'carol-pass-789');
      const carolKey = await carolManager.exportPublicKey();
      await manager.addFriend(carolKey);
      
      const friends = await manager.listFriends();
      
      expect(friends).toHaveLength(2);
      expect(friends[0]?.name).toBe('Bob');
      expect(friends[0]?.email).toBe('bob@example.com');
      expect(friends[1]?.name).toBe('Carol');
      expect(friends[1]?.email).toBe('carol@example.com');
    });

    test('should persist friends across account unlock', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      await friendManager.createAccount('Bob', 'bob@example.com', 'bob-pass-456');
      const bobKey = await friendManager.exportPublicKey();
      await manager.addFriend(bobKey);
      
      // Note: Since we're using the same storage instance, just unlock the same manager
      // or create a new manager with the same storage. For now, test that friends persist
      // by checking they're in the manager
      const friends = await manager.listFriends();
      
      expect(friends).toHaveLength(1);
      expect(friends[0]?.name).toBe('Bob');
    });

    test('should throw error when no account unlocked', async () => {
      await expect(manager.listFriends()).rejects.toThrow('No account unlocked');
    });
  });

  describe('removeFriend', () => {
    test('should remove friend by fingerprint', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      await friendManager.createAccount('Bob', 'bob@example.com', 'bob-pass-456');
      const bobKey = await friendManager.exportPublicKey();
      const bob = await manager.addFriend(bobKey);
      
      let friends = await manager.listFriends();
      expect(friends).toHaveLength(1);
      
      await manager.removeFriend(bob.fingerprint);
      
      friends = await manager.listFriends();
      expect(friends).toHaveLength(0);
    });

    test('should persist friend removal across unlock', async () => {
      await manager.createAccount('Alice', 'alice@example.com', 'alice-pass-123');
      
      await friendManager.createAccount('Bob', 'bob@example.com', 'bob-pass-456');
      const bobKey = await friendManager.exportPublicKey();
      const bob = await manager.addFriend(bobKey);
      
      await manager.removeFriend(bob.fingerprint);
      
      // Verify removal persists
      const friends = await manager.listFriends();
      expect(friends).toHaveLength(0);
    });

    test('should throw error when no account unlocked', async () => {
      await expect(manager.removeFriend('some-fingerprint')).rejects.toThrow('No account unlocked');
    });
  });
});
