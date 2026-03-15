/**
 * Tests for src/index.ts public API exports
 */

import { describe, it, expect, afterEach } from '@jest/globals';
import {
  createAccount,
  loadAccount,
  Account,
  Client,
  Server,
  File,
  createStorage,
  BrowserStorage,
  staticResolver,
  dhtResolver,
  combinedResolver,
  retryResolver,
  WebRTCClient,
  WebRTCServer,
  LocalSignalingServer,
  WebSocketSignaling,
  HTTPSignaling,
  SignaledConnection,
  KademliaDHT,
  MauError,
  PassphraseRequiredError,
  IncorrectPassphraseError,
  NoIdentityError,
  AccountAlreadyExistsError,
  InvalidFileNameError,
  FriendNotFollowedError,
  PeerNotFoundError,
  IncorrectPeerCertificateError,
} from './index';

const TEST_DIR = 'test-data-index';

afterEach(async () => {
  // Clean up IndexedDB
  const storage = await createStorage();
  try {
    await storage.remove(TEST_DIR);
  } catch {
    // Ignore cleanup errors
  }
});

describe('index exports', () => {
  describe('classes', () => {
    it('should export Account', () => {
      expect(Account).toBeDefined();
    });

    it('should export Client', () => {
      expect(Client).toBeDefined();
    });

    it('should export Server', () => {
      expect(Server).toBeDefined();
    });

    it('should export File', () => {
      expect(File).toBeDefined();
    });

    it('should export BrowserStorage', () => {
      expect(BrowserStorage).toBeDefined();
    });

    it('should export WebRTCClient', () => {
      expect(WebRTCClient).toBeDefined();
    });

    it('should export WebRTCServer', () => {
      expect(WebRTCServer).toBeDefined();
    });

    it('should export LocalSignalingServer', () => {
      expect(LocalSignalingServer).toBeDefined();
    });

    it('should export WebSocketSignaling', () => {
      expect(WebSocketSignaling).toBeDefined();
    });

    it('should export HTTPSignaling', () => {
      expect(HTTPSignaling).toBeDefined();
    });

    it('should export SignaledConnection', () => {
      expect(SignaledConnection).toBeDefined();
    });

    it('should export KademliaDHT', () => {
      expect(KademliaDHT).toBeDefined();
    });
  });

  describe('functions', () => {
    it('should export createAccount', () => {
      expect(createAccount).toBeDefined();
      expect(typeof createAccount).toBe('function');
    });

    it('should export loadAccount', () => {
      expect(loadAccount).toBeDefined();
      expect(typeof loadAccount).toBe('function');
    });

    it('should export createStorage', () => {
      expect(createStorage).toBeDefined();
      expect(typeof createStorage).toBe('function');
    });

    it('should export staticResolver', () => {
      expect(staticResolver).toBeDefined();
      expect(typeof staticResolver).toBe('function');
    });

    it('should export dhtResolver', () => {
      expect(dhtResolver).toBeDefined();
      expect(typeof dhtResolver).toBe('function');
    });

    it('should export combinedResolver', () => {
      expect(combinedResolver).toBeDefined();
      expect(typeof combinedResolver).toBe('function');
    });

    it('should export retryResolver', () => {
      expect(retryResolver).toBeDefined();
      expect(typeof retryResolver).toBe('function');
    });
  });

  describe('error classes', () => {
    it('should export MauError', () => {
      expect(MauError).toBeDefined();
      const err = new MauError('test', 'TEST_CODE');
      expect(err.code).toBe('TEST_CODE');
    });

    it('should export PassphraseRequiredError', () => {
      expect(PassphraseRequiredError).toBeDefined();
      const err = new PassphraseRequiredError();
      expect(err.code).toBe('PASSPHRASE_REQUIRED');
    });

    it('should export IncorrectPassphraseError', () => {
      expect(IncorrectPassphraseError).toBeDefined();
      const err = new IncorrectPassphraseError();
      expect(err.code).toBe('INCORRECT_PASSPHRASE');
    });

    it('should export NoIdentityError', () => {
      expect(NoIdentityError).toBeDefined();
    });

    it('should export AccountAlreadyExistsError', () => {
      expect(AccountAlreadyExistsError).toBeDefined();
    });

    it('should export InvalidFileNameError', () => {
      expect(InvalidFileNameError).toBeDefined();
    });

    it('should export FriendNotFollowedError', () => {
      expect(FriendNotFollowedError).toBeDefined();
    });

    it('should export PeerNotFoundError', () => {
      expect(PeerNotFoundError).toBeDefined();
      const err = new PeerNotFoundError();
      expect(err.code).toBe('PEER_NOT_FOUND');
    });

    it('should export IncorrectPeerCertificateError', () => {
      expect(IncorrectPeerCertificateError).toBeDefined();
    });
  });
});

describe('createAccount', () => {
  it('creates a new account', async () => {
    const account = await createAccount(TEST_DIR, 'Test User', 'test@example.com', 'passphrase');
    expect(account).toBeInstanceOf(Account);
    expect(account.getFingerprint()).toBeTruthy();
  });
});

describe('loadAccount', () => {
  it('loads an existing account', async () => {
    await createAccount(TEST_DIR, 'Test User', 'test@example.com', 'passphrase');
    const loaded = await loadAccount(TEST_DIR, 'passphrase');
    expect(loaded).toBeInstanceOf(Account);
  });

  it('throws with wrong passphrase', async () => {
    await createAccount(TEST_DIR, 'Test User', 'test@example.com', 'passphrase');
    await expect(loadAccount(TEST_DIR, 'wrong')).rejects.toThrow();
  });
});
