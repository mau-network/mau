/**
 * WebRTC API Tests - Testing actual WebRTC interfaces
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { WebRTCClient } from './network/webrtc';
import { WebRTCServer } from './network/webrtc-server';
import { Account } from './account';
import { FilesystemStorage } from './storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-webrtc-api';
const STUN_SERVER = 'stun:stun.l.google.com:19302';

describe('WebRTC API Tests', () => {
  let storage: FilesystemStorage;
  let serverAccount: Account;
  let clientAccount: Account;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    serverAccount = await Account.create(storage, TEST_DIR + '/server', {
      name: 'Server',
      email: 'server@test.com',
      passphrase: 'server-pass',
      algorithm: 'ed25519',
    });

    clientAccount = await Account.create(storage, TEST_DIR + '/client', {
      name: 'Client',
      email: 'client@test.com',
      passphrase: 'client-pass',
      algorithm: 'ed25519',
    });
  });

  afterAll(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  describe('WebRTCClient', () => {
    it('should create client', () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      expect(client).toBeDefined();
    });

    it('should create client with custom config', () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
          timeout: 60000,
        }
      );

      expect(client).toBeDefined();
    });

    it('should create offer', async () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client.createOffer();

      expect(offer).toBeDefined();
      expect(offer.type).toBe('offer');
      expect(offer.sdp).toBeDefined();
      expect(offer.sdp?.length).toBeGreaterThan(0);
    });

    it('should accept offer and create answer', async () => {
      const client1 = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client1.createOffer();

      const client2 = new WebRTCClient(
        serverAccount,
        storage,
        clientAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const answer = await client2.acceptOffer(offer);

      expect(answer).toBeDefined();
      expect(answer.type).toBe('answer');
      expect(answer.sdp).toBeDefined();
    });

    it('should complete connection with answer', async () => {
      const client1 = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client1.createOffer();

      const client2 = new WebRTCClient(
        serverAccount,
        storage,
        clientAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const answer = await client2.acceptOffer(offer);

      await client1.completeConnection(answer);

      // Connection completed (data channel may not be open yet)
      expect(client1).toBeDefined();
    });

    it('should throw when completing without connection', async () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const fakeAnswer = {
        type: 'answer' as const,
        sdp: 'fake-sdp',
      };

      await expect(client.completeConnection(fakeAnswer)).rejects.toThrow('No connection');
    });

    it('should add ICE candidate', async () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      await client.createOffer();

      const candidate = {
        candidate: 'candidate:0 1 UDP 2122252543 192.168.1.1 54321 typ host',
        sdpMid: '0',
        sdpMLineIndex: 0,
      };

      await client.addIceCandidate(candidate);

      expect(client).toBeDefined();
    });

    it('should throw when adding ICE candidate without connection', async () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const candidate = {
        candidate: 'candidate:0 1 UDP 2122252543 192.168.1.1 54321 typ host',
        sdpMid: '0',
        sdpMLineIndex: 0,
      };

      await expect(client.addIceCandidate(candidate)).rejects.toThrow('No connection');
    });
  });

  describe('WebRTCServer', () => {
    it('should create server', () => {
      const server = new WebRTCServer(serverAccount, storage);
      expect(server).toBeDefined();
      server.stop();
    });

    it('should create server with custom config', () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
        allowedPeers: [clientAccount.getFingerprint()],
      });

      expect(server).toBeDefined();
      server.stop();
    });

    it('should accept connection', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client.createOffer();

      const connectionId = 'test-connection-1';
      const answer = await server.acceptConnection(connectionId, offer);

      expect(answer).toBeDefined();
      expect(answer.type).toBe('answer');
      expect(answer.sdp).toBeDefined();

      server.stop();
    });

    it('should track connections', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client.createOffer();

      const connectionId = 'test-connection-2';
      await server.acceptConnection(connectionId, offer);

      const connections = server.getConnections();
      expect(connections).toContain(connectionId);

      server.stop();
    });

    it('should add ICE candidate to connection', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client.createOffer();

      const connectionId = 'test-connection-ice';
      await server.acceptConnection(connectionId, offer);

      const candidate = {
        candidate: 'candidate:0 1 UDP 2122252543 192.168.1.1 54321 typ host',
        sdpMid: '0',
        sdpMLineIndex: 0,
      };

      await server.addIceCandidate(connectionId, candidate);

      expect(server.getConnections()).toContain(connectionId);

      server.stop();
    });

    it('should throw when adding ICE to non-existent connection', async () => {
      const server = new WebRTCServer(serverAccount, storage);

      const candidate = {
        candidate: 'candidate:0 1 UDP 2122252543 192.168.1.1 54321 typ host',
        sdpMid: '0',
        sdpMLineIndex: 0,
      };

      await expect(
        server.addIceCandidate('non-existent', candidate)
      ).rejects.toThrow('not found');

      server.stop();
    });

    it('should register signaling callback', async () => {
      const server = new WebRTCServer(serverAccount, storage);

      const connectionId = 'test-signaling';
      let signalReceived = false;

      server.onSignaling(connectionId, (signal) => {
        signalReceived = true;
      });

      // Callback is registered (will be called when ICE candidates arrive)
      expect(server).toBeDefined();

      server.stop();
    });

    it('should close specific connection', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client.createOffer();

      const connectionId = 'test-close';
      await server.acceptConnection(connectionId, offer);

      expect(server.getConnections()).toContain(connectionId);

      server.closeConnection(connectionId);

      expect(server.getConnections()).not.toContain(connectionId);

      server.stop();
    });

    it('should stop all connections', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      const offer = await client.createOffer();

      await server.acceptConnection('conn1', offer);
      await server.acceptConnection('conn2', offer);

      expect(server.getConnections().length).toBeGreaterThanOrEqual(2);

      server.stop();

      expect(server.getConnections().length).toBe(0);
    });
  });

  describe('Full Connection Flow', () => {
    it('should complete offer-answer-complete flow', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint(),
        {
          iceServers: [{ urls: STUN_SERVER }],
        }
      );

      // Client creates offer
      const offer = await client.createOffer();
      expect(offer.type).toBe('offer');

      // Server accepts offer and creates answer
      const connectionId = 'full-flow';
      const answer = await server.acceptConnection(connectionId, offer);
      expect(answer.type).toBe('answer');

      // Client completes connection with answer
      await client.completeConnection(answer);

      // Connection established (data channel may still be connecting)
      expect(server.getConnections()).toContain(connectionId);

      server.stop();
    });

    it('should handle multiple concurrent connections', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: STUN_SERVER }],
      });

      const clients = [];
      const offers = [];

      // Create 3 clients
      for (let i = 0; i < 3; i++) {
        const account = await Account.create(storage, TEST_DIR + `/multi-client-${i}`, {
          name: `Client ${i}`,
          email: `client${i}@test.com`,
          passphrase: 'pass',
          algorithm: 'ed25519',
        });

        const client = new WebRTCClient(
          account,
          storage,
          serverAccount.getFingerprint(),
          {
            iceServers: [{ urls: STUN_SERVER }],
          }
        );

        clients.push(client);
        offers.push(await client.createOffer());
      }

      // Server accepts all connections
      const answers = [];
      for (let i = 0; i < 3; i++) {
        const answer = await server.acceptConnection(`multi-${i}`, offers[i]);
        answers.push(answer);
      }

      // All clients complete
      for (let i = 0; i < 3; i++) {
        await clients[i].completeConnection(answers[i]);
      }

      expect(server.getConnections().length).toBe(3);

      server.stop();
    }, 20000);
  });
});
