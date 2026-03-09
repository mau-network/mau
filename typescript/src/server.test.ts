/**
 * Tests for Server
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { Server } from './server';
import { Account } from './account';
import { File } from './file';
import { FilesystemStorage } from './storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-server';

describe('Server', () => {
  let storage: FilesystemStorage;
  let account: Account;
  let server: Server;

  beforeEach(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    account = await Account.create(storage, TEST_DIR, {
      name: 'Server User',
      email: 'server@example.com',
      passphrase: 'server-pass',
      algorithm: 'ed25519',
    });

    server = new Server(account, storage);

    // Create some test files
    const file1 = File.create(account, storage, 'test1.json');
    await file1.writeJSON({ message: 'test 1' });

    const file2 = File.create(account, storage, 'test2.json');
    await file2.writeJSON({ message: 'test 2' });
  });

  afterEach(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch (err) { /* cleanup error ignored */ }
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
    const file = File.create(account, storage, 'versioned.json');
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
      const file = File.create(account, storage, `file${i}.json`);
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

  it('should create Express middleware', () => {
    const middleware = server.expressMiddleware();
    expect(middleware).toBeDefined();
    expect(typeof middleware).toBe('function');
  });

  it('should create Node.js handler', () => {
    const handler = server.nodeHandler();
    expect(handler).toBeDefined();
    expect(typeof handler).toBe('function');
  });

  it('should handle Express middleware request', async () => {
    const middleware = server.expressMiddleware();

    const mockReq = {
      method: 'GET',
      url: `/p2p/${account.getFingerprint()}`,
      path: `/p2p/${account.getFingerprint()}`,
      query: {},
      headers: {},
    };

    const mockRes = {
      status: jest.fn().mockReturnThis(),
      setHeader: jest.fn(),
      send: jest.fn(),
    };

    const mockNext = jest.fn();

    await middleware(mockReq, mockRes, mockNext);

    expect(mockRes.status).toHaveBeenCalledWith(200);
    expect(mockRes.send).toHaveBeenCalled();
  });

  it('should pass non-p2p requests to next in Express middleware', async () => {
    const middleware = server.expressMiddleware();

    const mockReq = {
      method: 'GET',
      url: '/other/path',
      path: '/other/path',
      query: {},
      headers: {},
    };

    const mockRes = {
      status: jest.fn(),
      setHeader: jest.fn(),
      send: jest.fn(),
    };

    const mockNext = jest.fn();

    await middleware(mockReq, mockRes, mockNext);

    expect(mockNext).toHaveBeenCalled();
    expect(mockRes.status).not.toHaveBeenCalled();
  });
});
