/**
 * Signaling E2E Tests - LocalSignalingServer
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { LocalSignalingServer } from './signaling';
import { Account } from '../account';
import { FilesystemStorage } from '../storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-signaling';

describe('Signaling E2E Tests', () => {
  let storage: FilesystemStorage;
  let serverAccount: Account;
  let clientAccount: Account;

  beforeAll(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });

    serverAccount = await Account.create(storage, TEST_DIR + '/server', {
      name: 'Server User',
      email: 'server@example.com',
      passphrase: 'server-pass',
      algorithm: 'ed25519',
    });

    clientAccount = await Account.create(storage, TEST_DIR + '/client', {
      name: 'Client User',
      email: 'client@example.com',
      passphrase: 'client-pass',
      algorithm: 'ed25519',
    });
  });

  afterAll(async () => {
    try {
      await fs.rm(TEST_DIR, { recursive: true, force: true });
    } catch {}
  });

  describe('LocalSignalingServer - Post/Poll', () => {
    it('should create local signaling server', () => {
      const server = new LocalSignalingServer();
      expect(server).toBeDefined();
    });

    it('should post and poll offer message', async () => {
      const server = new LocalSignalingServer();

      const offer = {
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'mock-offer-sdp',
      };

      await server.post(offer);

      const messages = await server.poll(serverAccount.getFingerprint());
      expect(messages.length).toBe(1);
      expect(messages[0]).toEqual(offer);
    });

    it('should post and poll answer message', async () => {
      const server = new LocalSignalingServer();

      const answer = {
        type: 'answer' as const,
        from: serverAccount.getFingerprint(),
        to: clientAccount.getFingerprint(),
        data: 'mock-answer-sdp',
      };

      await server.post(answer);

      const messages = await server.poll(clientAccount.getFingerprint());
      expect(messages.length).toBe(1);
      expect(messages[0]).toEqual(answer);
    });

    it('should handle ICE candidate messages', async () => {
      const server = new LocalSignalingServer();

      const iceCandidate = {
        type: 'ice-candidate' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'candidate:1 1 UDP 2130706431 192.168.1.1 54321 typ host',
      };

      await server.post(iceCandidate);

      const messages = await server.poll(serverAccount.getFingerprint());
      expect(messages.length).toBe(1);
      expect(messages[0]).toEqual(iceCandidate);
    });

    it('should queue multiple messages', async () => {
      const server = new LocalSignalingServer();

      const message1 = {
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'offer-1',
      };

      const message2 = {
        type: 'ice-candidate' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'ice-1',
      };

      const message3 = {
        type: 'ice-candidate' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'ice-2',
      };

      await server.post(message1);
      await server.post(message2);
      await server.post(message3);

      const messages = await server.poll(serverAccount.getFingerprint());
      expect(messages.length).toBe(3);
      expect(messages[0]).toEqual(message1);
      expect(messages[1]).toEqual(message2);
      expect(messages[2]).toEqual(message3);
    });

    it('should clear messages after polling', async () => {
      const server = new LocalSignalingServer();

      const message = {
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'offer',
      };

      await server.post(message);

      // First poll
      const messages1 = await server.poll(serverAccount.getFingerprint());
      expect(messages1.length).toBe(1);

      // Second poll should be empty
      const messages2 = await server.poll(serverAccount.getFingerprint());
      expect(messages2).toEqual([]);
    });

    it('should handle bidirectional signaling', async () => {
      const server = new LocalSignalingServer();

      // Client sends offer
      await server.post({
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'offer-sdp',
      });

      // Server polls and gets offer
      const serverMessages = await server.poll(serverAccount.getFingerprint());
      expect(serverMessages.length).toBe(1);
      expect(serverMessages[0].type).toBe('offer');

      // Server sends answer
      await server.post({
        type: 'answer' as const,
        from: serverAccount.getFingerprint(),
        to: clientAccount.getFingerprint(),
        data: 'answer-sdp',
      });

      // Client polls and gets answer
      const clientMessages = await server.poll(clientAccount.getFingerprint());
      expect(clientMessages.length).toBe(1);
      expect(clientMessages[0].type).toBe('answer');
    });

    it('should handle messages for different recipients', async () => {
      const server = new LocalSignalingServer();
      
      const client2Fingerprint = 'client2-fingerprint';

      // Post to server
      await server.post({
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'offer-1',
      });

      // Post to client2
      await server.post({
        type: 'answer' as const,
        from: serverAccount.getFingerprint(),
        to: client2Fingerprint,
        data: 'answer-2',
      });

      // Poll for server - should get only offer
      const serverMessages = await server.poll(serverAccount.getFingerprint());
      expect(serverMessages.length).toBe(1);
      expect(serverMessages[0].type).toBe('offer');

      // Poll for client2 - should get only answer
      const client2Messages = await server.poll(client2Fingerprint);
      expect(client2Messages.length).toBe(1);
      expect(client2Messages[0].type).toBe('answer');

      // Poll for original client - should be empty
      const clientMessages = await server.poll(clientAccount.getFingerprint());
      expect(clientMessages).toEqual([]);
    });

    it('should return empty array when no messages', async () => {
      const server = new LocalSignalingServer();

      const messages = await server.poll(clientAccount.getFingerprint());
      expect(messages).toEqual([]);
    });

    it('should clear all pending messages', async () => {
      const server = new LocalSignalingServer();

      await server.post({
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'offer',
      });

      await server.post({
        type: 'answer' as const,
        from: serverAccount.getFingerprint(),
        to: clientAccount.getFingerprint(),
        data: 'answer',
      });

      server.clear();

      const serverMessages = await server.poll(serverAccount.getFingerprint());
      const clientMessages = await server.poll(clientAccount.getFingerprint());

      expect(serverMessages).toEqual([]);
      expect(clientMessages).toEqual([]);
    });

    it('should preserve message fields', async () => {
      const server = new LocalSignalingServer();

      const message = {
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'v=0\r\no=- 123 2 IN IP4 127.0.0.1\r\n',
        timestamp: Date.now(),
      };

      await server.post(message);

      const messages = await server.poll(serverAccount.getFingerprint());
      expect(messages[0]).toEqual(message);
    });

    it('should handle rapid post/poll cycles', async () => {
      const server = new LocalSignalingServer();

      for (let i = 0; i < 10; i++) {
        await server.post({
          type: 'ice-candidate' as const,
          from: clientAccount.getFingerprint(),
          to: serverAccount.getFingerprint(),
          data: `candidate-${i}`,
        });
      }

      const messages = await server.poll(serverAccount.getFingerprint());
      expect(messages.length).toBe(10);
      
      for (let i = 0; i < 10; i++) {
        expect(messages[i].data).toBe(`candidate-${i}`);
      }
    });

    it('should handle concurrent posts to different recipients', async () => {
      const server = new LocalSignalingServer();

      const posts = [
        server.post({
          type: 'offer' as const,
          from: 'peer1',
          to: 'peer2',
          data: 'offer1',
        }),
        server.post({
          type: 'offer' as const,
          from: 'peer3',
          to: 'peer4',
          data: 'offer2',
        }),
        server.post({
          type: 'offer' as const,
          from: 'peer5',
          to: 'peer6',
          data: 'offer3',
        }),
      ];

      await Promise.all(posts);

      const peer2Messages = await server.poll('peer2');
      const peer4Messages = await server.poll('peer4');
      const peer6Messages = await server.poll('peer6');

      expect(peer2Messages.length).toBe(1);
      expect(peer4Messages.length).toBe(1);
      expect(peer6Messages.length).toBe(1);
    });
  });

  describe('Complete Signaling Flow', () => {
    it('should handle full WebRTC signaling exchange', async () => {
      const server = new LocalSignalingServer();

      // 1. Client posts offer
      await server.post({
        type: 'offer' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'client-offer-sdp',
      });

      // 2. Server polls and receives offer
      const serverMessages1 = await server.poll(serverAccount.getFingerprint());
      expect(serverMessages1.length).toBe(1);
      expect(serverMessages1[0].type).toBe('offer');

      // 3. Server posts answer
      await server.post({
        type: 'answer' as const,
        from: serverAccount.getFingerprint(),
        to: clientAccount.getFingerprint(),
        data: 'server-answer-sdp',
      });

      // 4. Client polls and receives answer
      const clientMessages1 = await server.poll(clientAccount.getFingerprint());
      expect(clientMessages1.length).toBe(1);
      expect(clientMessages1[0].type).toBe('answer');

      // 5. Client posts ICE candidates
      await server.post({
        type: 'ice-candidate' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'client-ice-1',
      });

      await server.post({
        type: 'ice-candidate' as const,
        from: clientAccount.getFingerprint(),
        to: serverAccount.getFingerprint(),
        data: 'client-ice-2',
      });

      // 6. Server posts ICE candidates
      await server.post({
        type: 'ice-candidate' as const,
        from: serverAccount.getFingerprint(),
        to: clientAccount.getFingerprint(),
        data: 'server-ice-1',
      });

      // 7. Server polls and gets ICE candidates
      const serverMessages2 = await server.poll(serverAccount.getFingerprint());
      expect(serverMessages2.length).toBe(2);
      expect(serverMessages2.every(m => m.type === 'ice-candidate')).toBe(true);

      // 8. Client polls and gets ICE candidates
      const clientMessages2 = await server.poll(clientAccount.getFingerprint());
      expect(clientMessages2.length).toBe(1);
      expect(clientMessages2[0].type).toBe('ice-candidate');
    });
  });
});
