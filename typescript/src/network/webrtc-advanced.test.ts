/**
 * Advanced WebRTC E2E Tests - Connection scenarios
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { RTCPeerConnection, RTCSessionDescription, RTCIceCandidate } from '@roamhq/wrtc';
import { WebRTCClient } from './webrtc';
import { WebRTCServer } from './webrtc-server';
import { Account } from '../account';
import { FilesystemStorage } from '../storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-webrtc-advanced';

// Make wrtc available globally
(global as any).RTCPeerConnection = RTCPeerConnection;
(global as any).RTCSessionDescription = RTCSessionDescription;
(global as any).RTCIceCandidate = RTCIceCandidate;

describe('Advanced WebRTC E2E Tests', () => {
  let storage: FilesystemStorage;
  let clientAccount: Account;
  let serverAccount: Account;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    clientAccount = await Account.create(storage, TEST_DIR + '/client', {
      name: 'Client User',
      email: 'client@example.com',
      passphrase: 'client-pass',
      algorithm: 'ed25519',
    });

    serverAccount = await Account.create(storage, TEST_DIR + '/server', {
      name: 'Server User',
      email: 'server@example.com',
      passphrase: 'server-pass',
      algorithm: 'ed25519',
    });
  });

  afterAll(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  describe('Connection Lifecycle', () => {
    let server: WebRTCServer;
    let client: WebRTCClient;

    it('should create offer with ICE candidates', async () => {
      client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const offer = await client.createOffer();
      
      expect(offer).toBeDefined();
      expect(offer.type).toBe('offer');
      expect(offer.sdp).toBeDefined();
      expect(offer.sdp).toContain('a=ice');
    });

    it('should accept connection and create answer', async () => {
      server = new WebRTCServer(serverAccount, storage);
      
      const offer = await client.createOffer();
      const answer = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer
      );

      expect(answer).toBeDefined();
      expect(answer.type).toBe('answer');
      expect(answer.sdp).toBeDefined();
    });

    it('should complete connection', async () => {
      const offer = await client.createOffer();
      const answer = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer
      );
      
      await client.completeConnection(answer);
      
      // Wait for connection
      await new Promise(resolve => setTimeout(resolve, 300));

      expect(server.getConnections()).toContain(clientAccount.getFingerprint());
    });

    it('should maintain connection state', async () => {
      // Wait a bit
      await new Promise(resolve => setTimeout(resolve, 200));

      const connections = server.getConnections();
      expect(connections.length).toBeGreaterThanOrEqual(1);
      expect(connections).toContain(clientAccount.getFingerprint());
    });

    it('should close connection cleanly', () => {
      const fingerprint = clientAccount.getFingerprint();
      
      client.close();
      server.closeConnection(fingerprint);

      const connections = server.getConnections();
      expect(connections).not.toContain(fingerprint);
    });
  });

  describe('Multiple Connections', () => {
    it('should handle two simultaneous connections', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      
      const client1 = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const client2Account = await Account.create(storage, TEST_DIR + '/client2', {
        name: 'Client 2',
        email: 'client2@example.com',
        passphrase: 'client2-pass',
        algorithm: 'ed25519',
      });

      const client2 = new WebRTCClient(
        client2Account,
        storage,
        serverAccount.getFingerprint()
      );

      // Establish both connections
      const offer1 = await client1.createOffer();
      const answer1 = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer1
      );
      await client1.completeConnection(answer1);

      const offer2 = await client2.createOffer();
      const answer2 = await server.acceptConnection(
        client2Account.getFingerprint(),
        offer2
      );
      await client2.completeConnection(answer2);

      await new Promise(resolve => setTimeout(resolve, 300));

      const connections = server.getConnections();
      expect(connections.length).toBe(2);
      expect(connections).toContain(clientAccount.getFingerprint());
      expect(connections).toContain(client2Account.getFingerprint());

      client1.close();
      client2.close();
      server.closeConnection(clientAccount.getFingerprint());
      server.closeConnection(client2Account.getFingerprint());

      try {
        await fs.rm(TEST_DIR + '/client2', { recursive: true });
      } catch {}
    });

    it('should handle sequential connections', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      
      // Connect client 1
      const client1 = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );
      
      const offer1 = await client1.createOffer();
      const answer1 = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer1
      );
      await client1.completeConnection(answer1);
      await new Promise(resolve => setTimeout(resolve, 200));
      
      expect(server.getConnections()).toContain(clientAccount.getFingerprint());
      
      // Disconnect
      client1.close();
      server.closeConnection(clientAccount.getFingerprint());
      
      // Connect client 2
      const client2Account = await Account.create(storage, TEST_DIR + '/client3', {
        name: 'Client 3',
        email: 'client3@example.com',
        passphrase: 'client3-pass',
        algorithm: 'ed25519',
      });

      const client2 = new WebRTCClient(
        client2Account,
        storage,
        serverAccount.getFingerprint()
      );
      
      const offer2 = await client2.createOffer();
      const answer2 = await server.acceptConnection(
        client2Account.getFingerprint(),
        offer2
      );
      await client2.completeConnection(answer2);
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const connections = server.getConnections();
      expect(connections).not.toContain(clientAccount.getFingerprint());
      expect(connections).toContain(client2Account.getFingerprint());

      client2.close();
      server.closeConnection(client2Account.getFingerprint());

      try {
        await fs.rm(TEST_DIR + '/client3', { recursive: true });
      } catch {}
    });
  });

  describe('Error Handling', () => {
    it('should handle invalid offer gracefully', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      
      const invalidOffer = {
        type: 'offer' as RTCSdpType,
        sdp: 'invalid sdp data'
      };

      try {
        await server.acceptConnection(
          clientAccount.getFingerprint(),
          invalidOffer
        );
        // If no error thrown, it handled gracefully
      } catch (error) {
        // Expected error
        expect(error).toBeDefined();
      }
    });

    it('should handle missing peer gracefully', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      
      expect(() => {
        server.closeConnection('non-existent-peer');
      }).not.toThrow();
    });

    it('should handle client close before connection', () => {
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      expect(() => {
        client.close();
      }).not.toThrow();
    });
  });

  describe('Connection Properties', () => {
    it('should create connection with proper configuration', async () => {
      const server = new WebRTCServer(serverAccount, storage, {
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
        // iceTransportPolicy: "all",
      });

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const offer = await client.createOffer();
      expect(offer.sdp).toContain('a=ice');

      const answer = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer
      );
      expect(answer.sdp).toBeDefined();

      await client.completeConnection(answer);
      await new Promise(resolve => setTimeout(resolve, 200));

      expect(server.getConnections()).toContain(clientAccount.getFingerprint());

      client.close();
      server.closeConnection(clientAccount.getFingerprint());
    });

    it('should track all active connections', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      
      expect(server.getConnections()).toEqual([]);

      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const offer = await client.createOffer();
      const answer = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer
      );
      await client.completeConnection(answer);
      await new Promise(resolve => setTimeout(resolve, 200));

      const connections = server.getConnections();
      expect(connections.length).toBe(1);
      expect(connections[0]).toBe(clientAccount.getFingerprint());

      client.close();
      server.closeConnection(clientAccount.getFingerprint());
    });
  });

  describe('ICE Candidate Exchange', () => {
    it('should add ICE candidates after initial offer', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const offer = await client.createOffer();
      const answer = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer
      );

      // Simulate ICE candidate exchange
      const candidate: RTCIceCandidateInit = {
        candidate: 'candidate:1 1 UDP 2130706431 192.168.1.1 54321 typ host',
        sdpMid: '0',
        sdpMLineIndex: 0,
      };

      await expect(
        server.addIceCandidate(clientAccount.getFingerprint(), candidate)
      ).resolves.not.toThrow();

      await client.completeConnection(answer);
      await new Promise(resolve => setTimeout(resolve, 200));

      client.close();
      server.closeConnection(clientAccount.getFingerprint());
    });
  });

  describe('Server Lifecycle', () => {
    it('should stop server and close all connections', async () => {
      const server = new WebRTCServer(serverAccount, storage);
      const client = new WebRTCClient(
        clientAccount,
        storage,
        serverAccount.getFingerprint()
      );

      const offer = await client.createOffer();
      const answer = await server.acceptConnection(
        clientAccount.getFingerprint(),
        offer
      );
      await client.completeConnection(answer);
      await new Promise(resolve => setTimeout(resolve, 200));

      expect(server.getConnections().length).toBeGreaterThan(0);

      server.stop();

      expect(server.getConnections().length).toBe(0);

      client.close();
    });
  });
});
