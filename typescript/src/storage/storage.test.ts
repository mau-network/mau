/**
 * Tests for Browser Storage using fake-indexeddb
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import 'fake-indexeddb/auto';
import { BrowserStorage } from './browser';

describe('BrowserStorage', () => {
  let storage: BrowserStorage;

  beforeEach(async () => {
    storage = await BrowserStorage.create();
  });

  afterEach(() => {
    // Cleanup happens automatically with fake-indexeddb
  });

  describe('File Operations', () => {
    it('should write and read file', async () => {
      const data = new Uint8Array([1, 2, 3, 4, 5]);
      await storage.writeFile('test.txt', data);

      const read = await storage.readFile('test.txt');
      expect(read).toEqual(data);
    });

    it('should check if file exists', async () => {
      const data = new Uint8Array([1, 2, 3]);
      await storage.writeFile('exists.txt', data);

      expect(await storage.exists('exists.txt')).toBe(true);
      expect(await storage.exists('not-exists.txt')).toBe(false);
    });

    it('should delete file', async () => {
      const data = new Uint8Array([1, 2, 3]);
      await storage.writeFile('delete-me.txt', data);

      expect(await storage.exists('delete-me.txt')).toBe(true);
      await storage.remove('delete-me.txt');
      expect(await storage.exists('delete-me.txt')).toBe(false);
    });

    it('should list files', async () => {
      await storage.writeFile('file1.txt', new Uint8Array([1]));
      await storage.writeFile('file2.txt', new Uint8Array([2]));
      await storage.writeFile('file3.txt', new Uint8Array([3]));

      // Wait for IndexedDB to commit
      await new Promise(resolve => setTimeout(resolve, 50));

      // Check they exist first
      expect(await storage.exists('file1.txt')).toBe(true);
      expect(await storage.exists('file2.txt')).toBe(true);

      const files = await storage.readDir('');
      console.log('Files in root:', files);
      
      expect(files.length).toBeGreaterThanOrEqual(3);
      expect(files).toEqual(expect.arrayContaining(['file1.txt', 'file2.txt', 'file3.txt']));
    });

    it('should handle subdirectories in paths', async () => {
      const data = new Uint8Array([1, 2, 3]);
      await storage.writeFile('dir/subdir/file.txt', data);

      expect(await storage.exists('dir/subdir/file.txt')).toBe(true);
      const read = await storage.readFile('dir/subdir/file.txt');
      expect(read).toEqual(data);
    });

    it('should throw when reading non-existent file', async () => {
      await expect(storage.readFile('not-exists.txt')).rejects.toThrow();
    });

    it('should handle empty file', async () => {
      const empty = new Uint8Array([]);
      await storage.writeFile('empty.txt', empty);

      const read = await storage.readFile('empty.txt');
      expect(read).toEqual(empty);
    });

    it('should overwrite existing file', async () => {
      await storage.writeFile('overwrite.txt', new Uint8Array([1, 2, 3]));
      await storage.writeFile('overwrite.txt', new Uint8Array([4, 5, 6]));

      const read = await storage.readFile('overwrite.txt');
      expect(read).toEqual(new Uint8Array([4, 5, 6]));
    });

    it('should handle large files', async () => {
      const largeData = new Uint8Array(1024 * 1024); // 1MB
      for (let i = 0; i < largeData.length; i++) {
        largeData[i] = i % 256;
      }

      await storage.writeFile('large.bin', largeData);
      const read = await storage.readFile('large.bin');
      expect(read).toEqual(largeData);
    });

    it('should list files with different extensions', async () => {
      await storage.writeFile('file.txt', new Uint8Array([1]));
      await storage.writeFile('file.json', new Uint8Array([2]));
      await storage.writeFile('file.bin', new Uint8Array([3]));

      await new Promise(resolve => setTimeout(resolve, 10));

      const files = await storage.readDir('');
      expect(files.length).toBeGreaterThanOrEqual(3);
      expect(files).toEqual(expect.arrayContaining(['file.txt', 'file.json', 'file.bin']));
    });

    it('should handle paths with trailing slashes', async () => {
      const data = new Uint8Array([1, 2, 3]);
      await storage.writeFile('dir/file.txt', data);

      const files = await storage.readDir('dir/');
      expect(files).toContain('file.txt');
    });

    it('should delete all files in listing', async () => {
      await storage.writeFile('del1.txt', new Uint8Array([1]));
      await storage.writeFile('del2.txt', new Uint8Array([2]));
      await storage.writeFile('del3.txt', new Uint8Array([3]));

      const files = await storage.readDir('');
      for (const file of files) {
        if (file.startsWith('del')) {
          await storage.remove(file);
        }
      }

      const remaining = await storage.readDir('');
      expect(remaining.filter(f => f.startsWith('del'))).toEqual([]);
    });

    it('should handle unicode filenames', async () => {
      const data = new Uint8Array([1, 2, 3]);
      await storage.writeFile('файл.txt', data);

      expect(await storage.exists('файл.txt')).toBe(true);
      const read = await storage.readFile('файл.txt');
      expect(read).toEqual(data);
    });

    it('should handle special characters in filenames', async () => {
      const data = new Uint8Array([1, 2, 3]);
      const filename = 'file with spaces & symbols.txt';
      await storage.writeFile(filename, data);

      expect(await storage.exists(filename)).toBe(true);
      const read = await storage.readFile(filename);
      expect(read).toEqual(data);
    });
  });

  describe('Directory Operations', () => {
    it('should create directory structure implicitly', async () => {
      await storage.writeFile('a/b/c/file.txt', new Uint8Array([1]));
      expect(await storage.exists('a/b/c/file.txt')).toBe(true);
    });

    it('should list empty directory', async () => {
      const files = await storage.readDir('nonexistent/');
      expect(files).toEqual([]);
    });

    it('should handle nested directory listing', async () => {
      await storage.writeFile('root/sub1/file1.txt', new Uint8Array([1]));
      await storage.writeFile('root/sub2/file2.txt', new Uint8Array([2]));
      await storage.writeFile('root/file3.txt', new Uint8Array([3]));

      const files = await storage.readDir('root/');
      expect(files.length).toBeGreaterThanOrEqual(3);
    });
  });

  describe('Error Handling', () => {
    it('should handle invalid paths', async () => {
      await expect(storage.readFile('')).rejects.toThrow();
    });

    it('should handle deletion of non-existent file', async () => {
      // remove() doesn't throw, it succeeds silently
      await expect(storage.remove('does-not-exist.txt')).resolves.not.toThrow();
    });
  });

  describe('Storage Persistence', () => {
    it('should persist data across multiple operations', async () => {
      await storage.writeFile('persist1.txt', new Uint8Array([1]));
      await storage.writeFile('persist2.txt', new Uint8Array([2]));
      await storage.writeFile('persist3.txt', new Uint8Array([3]));

      const data1 = await storage.readFile('persist1.txt');
      const data2 = await storage.readFile('persist2.txt');
      const data3 = await storage.readFile('persist3.txt');

      expect(data1).toEqual(new Uint8Array([1]));
      expect(data2).toEqual(new Uint8Array([2]));
      expect(data3).toEqual(new Uint8Array([3]));
    });
  });
});
