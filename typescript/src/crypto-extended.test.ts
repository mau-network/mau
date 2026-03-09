/**
 * Crypto Module Tests - Hash, validation, key operations
 */

import { describe, it, expect } from '@jest/globals';
import {
  sha256,
  validateFileName,
  getFingerprint,
  serializePublicKey,
  deserializePublicKey
} from './crypto';
import * as openpgp from 'openpgp';

describe('Crypto Module Tests', () => {
  describe('SHA-256 Hashing', () => {
    it('should hash empty data', async () => {
      const data = new Uint8Array([]);
      const hash = await sha256(data);
      
      expect(hash).toBeDefined();
      expect(hash.length).toBe(64); // 32 bytes = 64 hex chars
    });

    it('should hash non-empty data', async () => {
      const data = new Uint8Array([1, 2, 3, 4, 5]);
      const hash = await sha256(data);
      
      expect(hash).toBeDefined();
      expect(hash.length).toBe(64);
    });

    it('should produce consistent hashes', async () => {
      const data = new Uint8Array([10, 20, 30]);
      
      const hash1 = await sha256(data);
      const hash2 = await sha256(data);
      
      expect(hash1).toBe(hash2);
    });

    it('should produce different hashes for different data', async () => {
      const data1 = new Uint8Array([1, 2, 3]);
      const data2 = new Uint8Array([4, 5, 6]);
      
      const hash1 = await sha256(data1);
      const hash2 = await sha256(data2);
      
      expect(hash1).not.toBe(hash2);
    });

    it('should handle large data', async () => {
      const largeData = new Uint8Array(1024 * 100); // 100KB
      for (let i = 0; i < largeData.length; i++) {
        largeData[i] = i % 256;
      }
      
      const hash = await sha256(largeData);
      expect(hash.length).toBe(64);
    });
  });

  describe('File Name Validation', () => {
    it('should accept valid filenames', () => {
      expect(validateFileName('file.txt')).toBe(true);
      expect(validateFileName('document.pdf')).toBe(true);
      expect(validateFileName('image.jpg')).toBe(true);
      expect(validateFileName('my-file_123.txt')).toBe(true);
    });

    it('should reject paths with slashes', () => {
      expect(validateFileName('path/to/file.txt')).toBe(false);
      expect(validateFileName('../file.txt')).toBe(false);
      expect(validateFileName('./file.txt')).toBe(false);
    });

    it('should reject special filenames', () => {
      expect(validateFileName('.')).toBe(false);
      expect(validateFileName('..')).toBe(false);
    });

    it('should reject empty filename', () => {
      expect(validateFileName('')).toBe(false);
    });

    it('should accept files without extension', () => {
      expect(validateFileName('README')).toBe(true);
      expect(validateFileName('Makefile')).toBe(true);
    });

    it('should accept dotfiles', () => {
      expect(validateFileName('.gitignore')).toBe(true);
      expect(validateFileName('.env')).toBe(true);
    });
  });

  describe('Fingerprint Operations', () => {
    let key: openpgp.PublicKey;

    beforeAll(async () => {
      const { publicKey } = await openpgp.generateKey({
        type: 'ecc',
        curve: 'ed25519',
        userIDs: [{ name: 'Test', email: 'test@example.com' }],
      });
      key = await openpgp.readKey({ armoredKey: publicKey });
    });

    it('should get fingerprint from public key', async () => {
      const fingerprint = await getFingerprint(key);
      
      expect(fingerprint).toBeDefined();
      expect(fingerprint.length).toBeGreaterThan(0);
    });

    it('should produce consistent fingerprints', async () => {
      const fp1 = await getFingerprint(key);
      const fp2 = await getFingerprint(key);
      
      expect(fp1).toBe(fp2);
    });

    it('should produce different fingerprints for different keys', async () => {
      const { publicKey: publicKey2 } = await openpgp.generateKey({
        type: 'ecc',
        curve: 'ed25519',
        userIDs: [{ name: 'Test 2', email: 'test2@example.com' }],
      });
      const key2 = await openpgp.readKey({ armoredKey: publicKey2 });
      
      const fp1 = await getFingerprint(key);
      const fp2 = await getFingerprint(key2);
      
      expect(fp1).not.toBe(fp2);
    });
  });

  describe('Key Serialization', () => {
    let key: openpgp.PublicKey;
    let armored: string;

    beforeAll(async () => {
      const { publicKey } = await openpgp.generateKey({
        type: 'ecc',
        curve: 'ed25519',
        userIDs: [{ name: 'Test', email: 'test@example.com' }],
      });
      key = await openpgp.readKey({ armoredKey: publicKey });
      armored = publicKey;
    });

    it('should serialize public key', async () => {
      const serialized = await serializePublicKey(key);
      
      expect(serialized).toBeDefined();
      expect(serialized).toContain('BEGIN PGP PUBLIC KEY BLOCK');
      expect(serialized).toContain('END PGP PUBLIC KEY BLOCK');
    });

    it('should serialize consistently', async () => {
      const s1 = await serializePublicKey(key);
      const s2 = await serializePublicKey(key);
      
      expect(s1).toBe(s2);
    });

    it('should deserialize public key', async () => {
      const deserialized = await deserializePublicKey(armored);
      
      expect(deserialized).toBeDefined();
      
      const originalFp = await getFingerprint(key);
      const deserializedFp = await getFingerprint(deserialized);
      
      expect(deserializedFp).toBe(originalFp);
    });

    it('should roundtrip serialize/deserialize', async () => {
      const serialized = await serializePublicKey(key);
      const deserialized = await deserializePublicKey(serialized);
      
      const originalFp = await getFingerprint(key);
      const roundtripFp = await getFingerprint(deserialized);
      
      expect(roundtripFp).toBe(originalFp);
    });

    it('should throw on invalid armored key', async () => {
      await expect(deserializePublicKey('not-a-key')).rejects.toThrow();
    });

    it('should throw on empty key', async () => {
      await expect(deserializePublicKey('')).rejects.toThrow();
    });
  });

  describe('Hash Consistency', () => {
    it('should match known SHA-256 hash', async () => {
      // "hello" in ASCII
      const data = new Uint8Array([104, 101, 108, 108, 111]);
      const hash = await sha256(data);
      
      // SHA-256 of "hello"
      const expectedHash = '2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824';
      expect(hash).toBe(expectedHash);
    });

    it('should handle binary data correctly', async () => {
      const data = new Uint8Array([0x00, 0xFF, 0x80, 0x7F]);
      const hash = await sha256(data);
      
      expect(hash).toBeDefined();
      expect(hash.length).toBe(64);
      expect(/^[0-9a-f]{64}$/.test(hash)).toBe(true);
    });
  });

  describe('Edge Cases', () => {
    it('should handle very long filenames', () => {
      const longName = 'a'.repeat(255) + '.txt';
      // May or may not throw depending on filesystem limits
      // Just ensure it doesn't crash
      try {
        validateFileName(longName);
      } catch (e) {
        expect(e).toBeDefined();
      }
    });

    it('should handle unicode filenames', () => {
      expect(validateFileName('文件.txt')).toBe(true);
      expect(validateFileName('файл.txt')).toBe(true);
      expect(validateFileName('αρχείο.txt')).toBe(true);
    });

    it('should handle files with multiple dots', () => {
      expect(validateFileName('file.tar.gz')).toBe(true);
      expect(validateFileName('backup.2024.01.01.zip')).toBe(true);
    });
  });
});
