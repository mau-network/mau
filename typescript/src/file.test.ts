/**
 * Tests for File operations
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Account } from '../account.js';
import { File } from '../file.js';
import { FilesystemStorage } from '../storage/filesystem.js';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-file';

describe('File', () => {
  let account: Account;
  let storage: FilesystemStorage;

  beforeEach(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    account = await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
    });
  });

  afterEach(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  it('should write and read text', async () => {
    const file = File.create(account, storage, 'test.txt');
    await file.writeText('Hello, World!');

    const content = await file.readText();
    expect(content).toBe('Hello, World!');
  });

  it('should write and read JSON', async () => {
    const file = File.create(account, storage, 'test.json');
    const data = { message: 'Hello', count: 42 };
    
    await file.writeJSON(data);
    const loaded = await file.readJSON();

    expect(loaded).toEqual(data);
  });

  it('should create versions on update', async () => {
    const file = File.create(account, storage, 'versioned.txt');
    
    await file.writeText('Version 1');
    await file.writeText('Version 2');
    await file.writeText('Version 3');

    const versions = await file.getVersions();
    expect(versions.length).toBe(2); // 2 previous versions
  });

  it('should list files', async () => {
    await File.create(account, storage, 'file1.txt').writeText('Content 1');
    await File.create(account, storage, 'file2.txt').writeText('Content 2');
    await File.create(account, storage, 'file3.txt').writeText('Content 3');

    const files = await File.list(account, storage);
    expect(files.length).toBe(3);
  });

  it('should delete files', async () => {
    const file = File.create(account, storage, 'delete-me.txt');
    await file.writeText('This will be deleted');

    let files = await File.list(account, storage);
    expect(files.length).toBe(1);

    await file.delete();

    files = await File.list(account, storage);
    expect(files.length).toBe(0);
  });
});
