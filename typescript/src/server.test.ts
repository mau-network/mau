/**
 * Tests for Server
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Server } from './server';
import { Account } from './account';
import { File } from './file';
import { BrowserStorage } from './storage/browser';

const TEST_DIR = 'test-data-server';

describe('Server', () => {
  let storage: BrowserStorage;
  let account: Account;
  let server: Server;

  beforeEach(async () => {
    storage = await BrowserStorage.create();

    account = await Account.create(storage, TEST_DIR, {
      name: 'Server User',
      email: 'server@example.com',
      passphrase: 'server-pass',
      algorithm: 'ed25519',
    });

    server = new Server(account, storage);

    // Create some test files
    const file1 = await account.createFile('test1.json');
    await file1.writeJSON({ message: 'test 1' });

    const file2 = await account.createFile('test2.json');
    await file2.writeJSON({ message: 'test 2' });
  });

  afterEach(async () => {
    try {
      await storage.remove(TEST_DIR);
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should create server instance', () => {
    expect(server).toBeDefined();
  });

  it('should handle file list request', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(200);
    expect(response.headers['Content-Type']).toBe('application/json');

    const body = JSON.parse(response.body as string);
    expect(body.files).toBeDefined();
    expect(body.files.length).toBeGreaterThan(0);
  });

  it('should handle file download request', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}/test1.json`,
      path: `/p2p/${account.getFingerprint()}/test1.json`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(200);
    expect(response.headers['Content-Type']).toBe('application/octet-stream');
    expect(response.body).toBeInstanceOf(Uint8Array);
  });

  it('should return 404 for non-existent file', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}/nonexistent.json`,
      path: `/p2p/${account.getFingerprint()}/nonexistent.json`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(404);
  });

  it('should return 404 for wrong fingerprint', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/wrongfingerprint123456789/test1.json`,
      path: `/p2p/wrongfingerprint123456789/test1.json`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(404);
  });

  it('should return 404 for invalid path', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: '/invalid/path',
      path: '/invalid/path',
      query: {},
      headers: {},
    });

    expect(response.status).toBe(404);
  });

  it('should return 405 for non-GET methods', async () => {
    const response = await server.handleRequest({
      method: 'POST',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(405);
    expect(response.body).toBe('Method Not Allowed');
  });

  it('should handle version download request', async () => {
    // Create a file and modify it to create a version
    const file = await account.createFile('versioned.json');
    await file.writeJSON({ version: 1 });
    const version1 = await file.getChecksum();
    
    await file.writeJSON({ version: 2 });

    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}/versioned.json.versions/${version1}`,
      path: `/p2p/${account.getFingerprint()}/versioned.json.versions/${version1}`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(200);
    expect(response.body).toBeInstanceOf(Uint8Array);
  });

  it('should return 404 for non-existent version', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}/test1.json.versions/nonexistent`,
      path: `/p2p/${account.getFingerprint()}/test1.json.versions/nonexistent`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(404);
  });

  it('should limit file list results', async () => {
    // Create many files
    for (let i = 0; i < 25; i++) {
      const file = await account.createFile(`file${i}.json`);
      await file.writeJSON({ index: i });
    }

    const serverWithLimit = new Server(account, storage, { resultsLimit: 10 });

    const response = await serverWithLimit.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {},
    });

    expect(response.status).toBe(200);
    const body = JSON.parse(response.body as string);
    expect(body.files.length).toBeLessThanOrEqual(10);
  });

  it('should get server config', () => {
    const config = server.getConfig();
    expect(config).toBeDefined();
    expect(config.resultsLimit).toBe(20); // default
  });

  it('should filter files by If-Modified-Since header', async () => {
    // Get current timestamp
    const now = new Date();
    const futureDate = new Date(now.getTime() + 10000); // 10 seconds in the future

    // Create file list request with If-Modified-Since in the future
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {
        'if-modified-since': futureDate.toUTCString(),
      },
    });

    expect(response.status).toBe(200);
    const body = JSON.parse(response.body as string);
    
    // Since all files were created before the future date, none should be returned
    expect(body.files.length).toBe(0);
  });

  it('should include all files when If-Modified-Since is in the past', async () => {
    // Use a date well in the past
    const pastDate = new Date('2020-01-01');

    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {
        'if-modified-since': pastDate.toUTCString(),
      },
    });

    expect(response.status).toBe(200);
    const body = JSON.parse(response.body as string);
    
    // All files should be included
    expect(body.files.length).toBeGreaterThan(0);
  });

  it('should handle invalid If-Modified-Since header gracefully', async () => {
    const response = await server.handleRequest({
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {
        'if-modified-since': 'invalid-date',
      },
    });

    expect(response.status).toBe(200);
    const body = JSON.parse(response.body as string);
    
    // Should return all files when date is invalid
    expect(body.files.length).toBeGreaterThan(0);
  });
});
