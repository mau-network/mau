/**
 * Tests for src/index.ts convenience API
 */

import { describe, it, expect, afterEach } from '@jest/globals';
import * as fs from 'fs/promises';
import {
  createAccount,
  loadAccount,
  Account,
  Client,
  Server,
  File,
  FilesystemStorage,
  createStorage,
  MauError,
  PeerNotFoundError,
  DHT_B,
  DHT_K,
  DHT_ALPHA,
} from './index';

const TEST_DIR = './test-data-index';

afterEach(async () => {
  await fs.rm(TEST_DIR, { recursive: true, force: true }).catch(() => {});
});

describe('src/index re-exports', () => {
  it('exports core classes', () => {
    expect(Account).toBeDefined();
    expect(Client).toBeDefined();
    expect(Server).toBeDefined();
    expect(File).toBeDefined();
    expect(FilesystemStorage).toBeDefined();
    expect(createStorage).toBeDefined();
  });

  it('exports error classes', () => {
    expect(MauError).toBeDefined();
    expect(PeerNotFoundError).toBeDefined();
    const err = new PeerNotFoundError();
    expect(err.code).toBe('PEER_NOT_FOUND');
  });

  it('exports DHT constants', () => {
    expect(DHT_B).toBe(160);
    expect(DHT_K).toBe(20);
    expect(DHT_ALPHA).toBe(3);
  });
});

describe('createAccount', () => {
  it('creates a new account', async () => {
    await fs.mkdir(TEST_DIR, { recursive: true });
    const account = await createAccount(TEST_DIR, 'Test User', 'test@example.com', 'passphrase');
    expect(account).toBeInstanceOf(Account);
    expect(account.getFingerprint()).toBeTruthy();
  });
});

describe('loadAccount', () => {
  it('loads an existing account', async () => {
    await fs.mkdir(TEST_DIR, { recursive: true });
    await createAccount(TEST_DIR, 'Test User', 'test@example.com', 'passphrase');
    const loaded = await loadAccount(TEST_DIR, 'passphrase');
    expect(loaded).toBeInstanceOf(Account);
  });

  it('throws with wrong passphrase', async () => {
    await fs.mkdir(TEST_DIR, { recursive: true });
    await createAccount(TEST_DIR, 'Test User', 'test@example.com', 'passphrase');
    await expect(loadAccount(TEST_DIR, 'wrong')).rejects.toThrow();
  });
});
