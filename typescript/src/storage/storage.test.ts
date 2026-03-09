/**
 * Tests for Storage
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { setupIndexedDBMocks, cleanupIndexedDBMocks } from '../__mocks__/indexeddb';
import { createStorage } from '../storage';
import { BrowserStorage } from '../storage/browser';

describe('createStorage', () => {
  it('should create storage', async () => {
    const storage = await createStorage();
    expect(storage).toBeDefined();
  });
});

describe('BrowserStorage', () => {
  let storage: BrowserStorage;

  beforeAll(() => {
    setupIndexedDBMocks();
  });

  afterAll(() => {
    cleanupIndexedDBMocks();
  });

  beforeEach(() => {
    storage = new BrowserStorage();
  });

  it('should create instance', () => {
    expect(storage).toBeDefined();
  });

  it('should join paths', () => {
    const joined = storage.join('a', 'b', 'c');
    expect(joined).toBe('a/b/c');
  });

  it('should handle empty parts in join', () => {
    const joined = storage.join('a', '', 'c');
    expect(joined).toBe('a/c');
  });

  it('should write and read text', async () => {
    await storage.writeText('test.txt', 'hello world');
    const text = await storage.readText('test.txt');
    expect(text).toBe('hello world');
  });

  it('should write and read binary', async () => {
    const data = new Uint8Array([1, 2, 3, 4, 5]);
    await storage.writeFile('test.bin', data);
    const read = await storage.readFile('test.bin');
    expect(read).toEqual(data);
  });

  it('should check if file exists', async () => {
    await storage.writeText('exists.txt', 'content');
    
    const exists = await storage.exists('exists.txt');
    const notExists = await storage.exists('notexists.txt');

    expect(exists).toBe(true);
    expect(notExists).toBe(false);
  });

  it('should get file stats', async () => {
    const data = new Uint8Array([1, 2, 3]);
    await storage.writeFile('stats.bin', data);

    const stats = await storage.stat('stats.bin');

    expect(stats.size).toBe(3);
    expect(stats.isDirectory).toBe(false);
  });

  it('should handle directory creation', async () => {
    // mkdir is a no-op for browser storage
    await storage.mkdir('dir1/dir2/dir3');
    // Should not throw
  });

  it('should list directory contents', async () => {
    await storage.writeText('dir/file1.txt', 'content1');
    await storage.writeText('dir/file2.txt', 'content2');
    await storage.writeText('dir/sub/file3.txt', 'content3');

    const entries = await storage.readDir('dir');

    expect(entries).toContain('file1.txt');
    expect(entries).toContain('file2.txt');
    expect(entries.length).toBeGreaterThan(0);
  });

  it('should remove file', async () => {
    await storage.writeText('remove.txt', 'content');
    
    expect(await storage.exists('remove.txt')).toBe(true);

    await storage.remove('remove.txt');

    expect(await storage.exists('remove.txt')).toBe(false);
  });

  it('should handle reading non-existent file', async () => {
    await expect(storage.readText('nonexistent.txt')).rejects.toThrow();
  });

  it('should handle stat on non-existent file', async () => {
    await expect(storage.stat('nonexistent.txt')).rejects.toThrow();
  });

  it('should handle readDir on non-existent directory', async () => {
    const entries = await storage.readDir('nonexistent-dir');
    expect(entries).toEqual([]);
  });

  it('should handle multiple writes to same file', async () => {
    await storage.writeText('multi.txt', 'first');
    await storage.writeText('multi.txt', 'second');
    
    const text = await storage.readText('multi.txt');
    expect(text).toBe('second');
  });

  it('should handle large binary data', async () => {
    const largeData = new Uint8Array(10000);
    for (let i = 0; i < largeData.length; i++) {
      largeData[i] = i % 256;
    }
    
    await storage.writeFile('large.bin', largeData);
    const read = await storage.readFile('large.bin');
    
    expect(read.length).toBe(largeData.length);
    expect(read).toEqual(largeData);
  });

  it('should handle nested directory paths', async () => {
    await storage.writeText('a/b/c/d/file.txt', 'nested');
    const text = await storage.readText('a/b/c/d/file.txt');
    expect(text).toBe('nested');
  });
});
