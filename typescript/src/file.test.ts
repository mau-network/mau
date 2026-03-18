/**
 * Tests for File operations
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Account } from './account';
import { BrowserStorage } from './storage/browser';

const TEST_DIR = 'test-data-file';

describe('File', () => {
  let account: Account;
  let storage: BrowserStorage;

  beforeEach(async () => {
    storage = await BrowserStorage.create();

    account = await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
    });
  });

  afterEach(async () => {
    try {
      await storage.remove(TEST_DIR);
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should write and read text', async () => {
    const file = await account.createFile('test.txt');
    await file.writeText('Hello, World!');

    const content = await file.readText();
    expect(content).toBe('Hello, World!');
  });

  it('should write and read JSON', async () => {
    const file = await account.createFile('test.json');
    const data = { message: 'Hello', count: 42 };
    
    await file.writeJSON(data);
    const loaded = await file.readJSON();

    expect(loaded).toEqual(data);
  });

  it('should create versions on update', async () => {
    const file = await account.createFile('versioned.txt');
    
    await file.writeText('Version 1');
    await file.writeText('Version 2');
    await file.writeText('Version 3');

    const versions = await file.getVersions();
    expect(versions.length).toBe(2); // 2 previous versions
  });

  it('should list files', async () => {
    const file1 = await account.createFile('file1.txt');
    await file1.writeText('Content 1');
    const file2 = await account.createFile('file2.txt');
    await file2.writeText('Content 2');
    const file3 = await account.createFile('file3.txt');
    await file3.writeText('Content 3');

    const files = await account.listFiles();
    expect(files.length).toBe(3);
  });

  it('should delete files', async () => {
    const file = await account.createFile('delete-me.txt');
    await file.writeText('This will be deleted');

    let files = await account.listFiles();
    expect(files.length).toBe(1);

    await file.delete();

    files = await account.listFiles();
    expect(files.length).toBe(0);
  });
});
