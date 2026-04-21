/**
 * Selective Encryption Tests
 * 
 * Tests for File.write() with custom recipient lists
 */

import { describe, it, expect, beforeAll } from '@jest/globals';
import { Account } from './account.js';
import { BrowserStorage } from './storage/browser.js';

describe('File Selective Encryption', () => {
  let storage: BrowserStorage;
  let alice: Account;
  let bobFingerprint: string;
  let charlieFingerprint: string;

  beforeAll(async () => {
    storage = await BrowserStorage.create();
    
    // Create Alice's account
    alice = await Account.create(storage, '/alice', {
      name: 'Alice',
      email: 'alice@example.com',
      passphrase: 'alice-secret',
    });

    // Create temporary accounts for Bob and Charlie to get their fingerprints
    const bob = await Account.create(storage, '/bob', {
      name: 'Bob',
      email: 'bob@example.com',
      passphrase: 'bob-secret',
    });
    bobFingerprint = bob.getFingerprint();

    const charlie = await Account.create(storage, '/charlie', {
      name: 'Charlie',
      email: 'charlie@example.com',
      passphrase: 'charlie-secret',
    });
    charlieFingerprint = charlie.getFingerprint();

    // Add Bob and Charlie as Alice's friends
    await alice.addFriend(bob.getPublicKey());
    await alice.addFriend(charlie.getPublicKey());
  });

  it('should encrypt for all friends by default (no recipients specified)', async () => {
    const file = await alice.createFile('public.json.pgp');
    await file.writeJSON({ message: 'Hello everyone!' });

    // File should exist
    const content = await file.readJSON();
    expect(content).toEqual({ message: 'Hello everyone!' });
  });

  it('should encrypt only for specified recipients', async () => {
    const file = await alice.createFile('private-to-bob.json.pgp');
    
    // Encrypt only for Bob (Alice is always included by crypto layer)
    await file.writeJSON(
      { message: 'Secret message for Bob' },
      { recipients: [bobFingerprint] }
    );

    // Alice can read it (own message)
    const data = await file.readJSON();
    expect(data).toEqual({ message: 'Secret message for Bob' });
  });

  it('should encrypt for multiple specific recipients', async () => {
    const file = await alice.createFile('group-message.json.pgp');
    
    // Encrypt for both Bob and Charlie
    await file.writeJSON(
      { message: 'Group message' },
      { recipients: [bobFingerprint, charlieFingerprint] }
    );

    // Alice can read (own message)
    const content = await file.readJSON();
    expect(content).toEqual({ message: 'Group message' });
  });

  it('should include Alice when encrypting for specific recipients', async () => {
    const file = await alice.createFile('self-readable.json.pgp');
    
    // Encrypt only for Bob
    await file.writeJSON(
      { message: 'Bob only' },
      { recipients: [bobFingerprint] }
    );

    // Alice should still be able to read her own file
    const data = await file.readJSON();
    expect(data).toEqual({ message: 'Bob only' });
  });

  it('should throw error for unknown fingerprint', async () => {
    const file = await alice.createFile('invalid.json.pgp');
    
    await expect(
      file.writeJSON(
        { message: 'Test' },
        { recipients: ['AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'] }
      )
    ).rejects.toThrow('Fingerprints not found');
  });

  it('should support writeText with recipients', async () => {
    const file = await alice.createFile('text-message.pgp');
    
    await file.writeText(
      'Private text for Bob',
      { recipients: [bobFingerprint] }
    );

    const text = await file.readText();
    expect(text).toBe('Private text for Bob');
  });

  it('should support write (binary) with recipients', async () => {
    const file = await alice.createFile('binary-message.pgp');
    const binaryData = new TextEncoder().encode('Binary data for Charlie');
    
    await file.write(binaryData, { recipients: [charlieFingerprint] });

    const result = await file.read();
    expect(new TextDecoder().decode(result)).toBe('Binary data for Charlie');
  });
});
