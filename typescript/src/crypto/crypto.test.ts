/**
 * Tests for Crypto functions
 */

import { describe, it, expect } from '@jest/globals';
import {
  generateKeyPair,
  serializePublicKey,
  serializePrivateKey,
  deserializePublicKey,
  deserializePrivateKey,
  getFingerprint,
  sign,
  verify,
  signAndEncrypt,
  decryptAndVerify,
  sha256,
} from '../crypto';

describe('Crypto', () => {
  it('should generate Ed25519 key pair', async () => {
    const { privateKey, publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
      algorithm: 'ed25519',
    });

    expect(privateKey).toBeDefined();
    expect(publicKey).toBeDefined();
  });

  it('should generate RSA key pair', async () => {
    const { privateKey, publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
      algorithm: 'rsa',
      rsaBits: 2048,
    });

    expect(privateKey).toBeDefined();
    expect(publicKey).toBeDefined();
  }, 30000);

  it('should serialize and deserialize public key', async () => {
    const { publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
      algorithm: 'ed25519',
    });

    const serialized = await serializePublicKey(publicKey);
    expect(serialized).toContain('BEGIN PGP PUBLIC KEY BLOCK');

    const deserialized = await deserializePublicKey(serialized);
    expect(deserialized).toBeDefined();
  });

  it('should serialize and deserialize private key', async () => {
    const passphrase = 'test-passphrase';
    const { privateKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase,
      algorithm: 'ed25519',
    });

    const serialized = await serializePrivateKey(privateKey);
    expect(serialized).toContain('BEGIN PGP PRIVATE KEY BLOCK');

    const deserialized = await deserializePrivateKey(serialized, passphrase);
    expect(deserialized).toBeDefined();
  });

  it('should get fingerprint from key', async () => {
    const { publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase: 'test-passphrase',
      algorithm: 'ed25519',
    });

    const fingerprint = await getFingerprint(publicKey);
    expect(fingerprint).toBeDefined();
    expect(fingerprint.length).toBeGreaterThan(0);
    expect(/^[0-9a-f]+$/.test(fingerprint)).toBe(true);
  });

  it('should sign and verify data', async () => {
    const passphrase = 'test-passphrase';
    const { privateKey, publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase,
      algorithm: 'ed25519',
    });

    const data = new TextEncoder().encode('test message');

    const signature = await sign(data, privateKey);
    expect(signature).toBeDefined();

    const isValid = await verify(data, signature, publicKey);
    expect(isValid).toBe(true);
  });

  it('should fail verification with wrong data', async () => {
    const passphrase = 'test-passphrase';
    const { privateKey, publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase,
      algorithm: 'ed25519',
    });

    const data = new TextEncoder().encode('test message');
    const wrongData = new TextEncoder().encode('wrong message');

    const signature = await sign(data, privateKey);
    const isValid = await verify(wrongData, signature, publicKey);

    expect(isValid).toBe(false);
  });

  it('should sign and encrypt data', async () => {
    const passphrase = 'test-passphrase';
    const { privateKey, publicKey } = await generateKeyPair({
      name: 'Test User',
      email: 'test@example.com',
      passphrase,
      algorithm: 'ed25519',
    });

    const data = new TextEncoder().encode('secret message');

    const encrypted = await signAndEncrypt(data, privateKey, [publicKey]);
    expect(encrypted).toBeDefined();

    const result = await decryptAndVerify(encrypted, privateKey, [publicKey]);
    expect(result.data).toEqual(data);
    expect(result.verified).toBe(true);
  });

  it('should encrypt for multiple recipients', async () => {
    const { privateKey: privateKey1, publicKey: publicKey1 } = await generateKeyPair({
      name: 'User 1',
      email: 'user1@example.com',
      passphrase: 'pass1',
      algorithm: 'ed25519',
    });

    const { privateKey: privateKey2, publicKey: publicKey2 } = await generateKeyPair({
      name: 'User 2',
      email: 'user2@example.com',
      passphrase: 'pass2',
      algorithm: 'ed25519',
    });

    const data = new TextEncoder().encode('shared secret');

    const encrypted = await signAndEncrypt(data, privateKey1, [publicKey1, publicKey2]);

    const result1 = await decryptAndVerify(encrypted, privateKey1, [publicKey1]);
    const result2 = await decryptAndVerify(encrypted, privateKey2, [publicKey1]);

    expect(result1.data).toEqual(data);
    expect(result2.data).toEqual(data);
  });

  it('should compute SHA-256 checksum', async () => {
    const data = new TextEncoder().encode('test data');
    const checksum = await sha256(data);

    expect(checksum).toBeDefined();
    expect(checksum.length).toBe(64); // SHA-256 hex string
    expect(/^[0-9a-f]{64}$/.test(checksum)).toBe(true);
  });

  it('should produce same checksum for same data', async () => {
    const data = new TextEncoder().encode('test data');
    const checksum1 = await sha256(data);
    const checksum2 = await sha256(data);

    expect(checksum1).toBe(checksum2);
  });

  it('should produce different checksum for different data', async () => {
    const data1 = new TextEncoder().encode('test data 1');
    const data2 = new TextEncoder().encode('test data 2');

    const checksum1 = await sha256(data1);
    const checksum2 = await sha256(data2);

    expect(checksum1).not.toBe(checksum2);
  });
});
