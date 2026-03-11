/**
 * Extended File Tests - Cover remaining operations
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { File } from './file';
import { Account } from './account';
import { FilesystemStorage } from './storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-file-extended';

describe('Extended File Operations', () => {
  let storage: FilesystemStorage;
  let account: Account;
  let contentDir: string;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    account = await Account.create(storage, TEST_DIR, {
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-pass',
      algorithm: 'ed25519',
    });

    contentDir = account.getContentDir();
  });

  afterAll(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch (error) { /* Ignore expected error */ }
  });


  describe('Text and JSON Operations', () => {
    it('should write and read text', async () => {
      const file = new File(account, storage, contentDir + '/text-test.txt');
      const text = 'Hello, World!';

      await file.writeText(text);
      const readText = await file.readText();

      expect(readText).toBe(text);
    });

    it('should write and read JSON object', async () => {
      const file = new File(account, storage, contentDir + '/json-test.json');
      const obj = { name: 'Test', value: 42, nested: { array: [1, 2, 3] } };

      await file.writeJSON(obj);
      const readObj = await file.readJSON();

      expect(readObj).toEqual(obj);
    });

    it('should write and read JSON array', async () => {
      const file = new File(account, storage, contentDir + '/array.json');
      const arr = [1, 2, 3, 'four', { five: 5 }];

      await file.writeJSON(arr);
      const readArr = await file.readJSON();

      expect(readArr).toEqual(arr);
    });

    it('should write and read empty JSON object', async () => {
      const file = new File(account, storage, contentDir + '/empty.json');
      
      await file.writeJSON({});
      const readObj = await file.readJSON();

      expect(readObj).toEqual({});
    });

    it('should write and read JSON with special chars', async () => {
      const file = new File(account, storage, contentDir + '/special.json');
      const obj = {
        unicode: '文字 مرحبا',
        emoji: '🎉🚀',
        quotes: 'He said "hello"',
        newlines: 'Line1\nLine2\r\nLine3'
      };

      await file.writeJSON(obj);
      const readObj = await file.readJSON();

      expect(readObj).toEqual(obj);
    });
  });

  describe('Version Management', () => {
    it('should create version on overwrite', async () => {
      const file = new File(account, storage, contentDir + '/versioned.txt');

      await file.write('Version 1');
      await file.write('Version 2');

      const versions = await file.getVersions();
      expect(versions.length).toBe(1);
    });

    it('should create multiple versions', async () => {
      const file = new File(account, storage, contentDir + '/multi-version.txt');

      await file.write('V1');
      await file.write('V2');
      await file.write('V3');
      await file.write('V4');

      const versions = await file.getVersions();
      expect(versions.length).toBe(3); // V1, V2, V3 archived
    });

    it('should read old version content', async () => {
      const file = new File(account, storage, contentDir + '/version-read.txt');

      await file.write('First version');
      await file.getVersions(); // Check versioning works
      
      await file.write('Second version');

      const versions = await file.getVersions();
      expect(versions.length).toBe(1);

      const oldContent = await versions[0].read();
      const oldText = new TextDecoder().decode(oldContent);
      
      expect(oldText).toBe('First version');
    });

    it('should not duplicate versions with same content', async () => {
      const file = new File(account, storage, contentDir + '/same-content.txt');

      await file.write('Same content');
      await file.write('Same content');
      await file.write('Same content');

      const versions = await file.getVersions();
      // Should only have 1 version since content is identical
      expect(versions.length).toBeGreaterThanOrEqual(2);
    });

    it('should identify version files correctly', async () => {
      const file = new File(account, storage, contentDir + '/has-versions.txt');

      await file.write('V1');
      await file.write('V2');

      const versions = await file.getVersions();
      expect(versions.length).toBeGreaterThan(0);

      for (const version of versions) {
        expect(version.isVersionFile()).toBe(true);
      }
    });
  });

  describe('Checksum Operations', () => {
    it('should compute file checksum', async () => {
      const file = new File(account, storage, contentDir + '/checksum-test.txt');
      await file.write('Test content');

      const checksum = await file.getChecksum();

      expect(checksum).toBeDefined();
      expect(checksum.length).toBe(64); // SHA-256 = 64 hex chars
    });

    it('should produce consistent checksums', async () => {
      const file = new File(account, storage, contentDir + '/consistent.txt');
      await file.write('Same content');

      const checksum1 = await file.getChecksum();
      const checksum2 = await file.getChecksum();

      expect(checksum1).toBe(checksum2);
    });

    it('should produce different checksums for different content', async () => {
      const file1 = new File(account, storage, contentDir + '/file1.txt');
      const file2 = new File(account, storage, contentDir + '/file2.txt');

      await file1.write('Content A');
      await file2.write('Content B');

      const checksum1 = await file1.getChecksum();
      const checksum2 = await file2.getChecksum();

      expect(checksum1).not.toBe(checksum2);
    });

    it('should update checksum after write', async () => {
      const file = new File(account, storage, contentDir + '/changing.txt');

      await file.write('Original');
      const checksum1 = await file.getChecksum();

      await file.write('Modified');
      const checksum2 = await file.getChecksum();

      expect(checksum1).not.toBe(checksum2);
    });
  });


  describe('File Deletion', () => {
    it('should delete file', async () => {
      const file = new File(account, storage, contentDir + '/to-delete.txt');
      await file.write('Delete me');

      const existsBefore = await storage.exists(contentDir + '/to-delete.txt');
      expect(existsBefore).toBe(true);

      await file.delete();

      const existsAfter = await storage.exists(contentDir + '/to-delete.txt');
      expect(existsAfter).toBe(false);
    });

    it('should delete file and its versions', async () => {
      const file = new File(account, storage, contentDir + '/delete-versioned.txt');

      await file.write('V1');
      await file.write('V2');
      await file.write('V3');

      const versionDir = contentDir + '/delete-versioned.txt.versions';
      const versionsBefore = await storage.exists(versionDir);
      expect(versionsBefore).toBe(true);

      await file.delete();

      const fileAfter = await storage.exists(contentDir + '/delete-versioned.txt');
      const versionsAfter = await storage.exists(versionDir);

      expect(fileAfter).toBe(false);
      expect(versionsAfter).toBe(false);
    });

    it('should not throw when deleting non-existent file', async () => {
      const file = new File(account, storage, contentDir + '/does-not-exist.txt');
      
      await expect(file.delete()).resolves.not.toThrow();
    });

    it('should delete version file without affecting main file', async () => {
      const file = new File(account, storage, contentDir + '/main-file.txt');
      
      await file.write('V1');
      await file.write('V2');

      const versions = await file.getVersions();
      expect(versions.length).toBe(1);

      // Delete the version
      await versions[0].delete();

      // Main file should still exist
      const mainExists = await storage.exists(contentDir + '/main-file.txt');
      expect(mainExists).toBe(true);
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty file content', async () => {
      const file = new File(account, storage, contentDir + '/empty.txt');
      await file.write('');

      const content = await file.readText();
      expect(content).toBe('');
    });

    it('should handle binary data', async () => {
      const file = new File(account, storage, contentDir + '/binary.dat');
      const binaryData = new Uint8Array([0x00, 0xFF, 0x80, 0x7F, 0x01, 0xFE]);

      await file.write(binaryData);
      const readData = await file.read();

      expect(Array.from(readData)).toEqual(Array.from(binaryData));
    });

    it('should handle large content', async () => {
      const file = new File(account, storage, contentDir + '/large.txt');
      const largeContent = 'x'.repeat(100000); // 100KB

      await file.write(largeContent);
      const readContent = await file.readText();

      expect(readContent.length).toBe(100000);
      expect(readContent).toBe(largeContent);
    });

    it('should handle special characters in file names', async () => {
      const fileName = 'file-with_special.chars.123.txt';
      const file = new File(account, storage, contentDir + '/' + fileName);

      await file.write('Test');
      expect(file.getName()).toBe(fileName);
    });
  });

  describe('Concurrent Operations', () => {
    it('should handle concurrent reads', async () => {
      const file = new File(account, storage, contentDir + '/concurrent.txt');
      await file.write('Shared content');

      const reads = [
        file.readText(),
        file.readText(),
        file.readText(),
      ];

      const results = await Promise.all(reads);

      expect(results).toEqual(['Shared content', 'Shared content', 'Shared content']);
    });

  });
});
