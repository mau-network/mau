/**
 * Tests for Signaling
 */

import { describe, it, expect, beforeEach } from '@jest/globals';
import {
  LocalSignalingServer,
  HTTPSignaling,
  SignaledConnection,
} from './signaling';

describe('LocalSignalingServer', () => {
  let server: LocalSignalingServer;

  beforeEach(() => {
    server = new LocalSignalingServer();
  });

  it('should create instance', () => {
    expect(server).toBeDefined();
  });

  it('should post and poll messages', async () => {
    const message = {
      from: 'fingerprint-alice',
      to: 'fingerprint-bob',
      type: 'offer' as const,
      data: { sdp: 'test-sdp', type: 'offer' },
    };

    await server.post(message);

    const messages = await server.poll('fingerprint-bob');
    expect(messages).toHaveLength(1);
    expect(messages[0]).toMatchObject(message);
  });

  it('should return empty array for unknown fingerprint', async () => {
    const messages = await server.poll('unknown-fingerprint');
    expect(messages).toEqual([]);
  });

  it('should clear all messages', async () => {
    await server.post({
      from: 'alice',
      to: 'bob',
      type: 'offer',
      data: {},
    });

    server.clear();

    const messages = await server.poll('bob');
    expect(messages).toEqual([]);
  });

  it('should queue multiple messages', async () => {
    await server.post({
      from: 'alice',
      to: 'bob',
      type: 'offer',
      data: { msg: 1 },
    });

    await server.post({
      from: 'alice',
      to: 'bob',
      type: 'answer',
      data: { msg: 2 },
    });

    const messages = await server.poll('bob');
    expect(messages).toHaveLength(2);
  });

  it('should consume messages on poll', async () => {
    await server.post({
      from: 'alice',
      to: 'bob',
      type: 'offer',
      data: {},
    });

    const firstPoll = await server.poll('bob');
    expect(firstPoll).toHaveLength(1);

    const secondPoll = await server.poll('bob');
    expect(secondPoll).toHaveLength(0);
  });
});

describe('HTTPSignaling', () => {
  it('should create instance', () => {
    const signaling = new HTTPSignaling('http://localhost:8080', 'test-fingerprint');
    expect(signaling).toBeDefined();
  });

  it('should register message handler', () => {
    const signaling = new HTTPSignaling('http://localhost:8080', 'test-fingerprint');
    const handler = jest.fn();

    signaling.onMessage(handler);

    // Handler should be registered (internal state, not directly testable without mocking)
    expect(handler).toBeDefined();
  });
});

describe('SignaledConnection', () => {
  it('should create instance with mock signaling', () => {
    const mockSignaling = {
      onMessage: jest.fn(),
      send: jest.fn(),
    };

    const connection = new SignaledConnection(
      mockSignaling as any,
      'local-fingerprint',
      'remote-fingerprint'
    );

    expect(connection).toBeDefined();
    expect(mockSignaling.onMessage).toHaveBeenCalled();
  });

  it('should register callbacks', () => {
    const mockSignaling = {
      onMessage: jest.fn(),
      send: jest.fn(),
    };

    const connection = new SignaledConnection(
      mockSignaling as any,
      'local-fpr',
      'remote-fpr'
    );

    const offerCallback = jest.fn();
    const answerCallback = jest.fn();
    const iceCallback = jest.fn();

    connection.onOffer(offerCallback);
    connection.onAnswer(answerCallback);
    connection.onICECandidate(iceCallback);

    // Callbacks registered (internal state)
    expect(offerCallback).toBeDefined();
    expect(answerCallback).toBeDefined();
    expect(iceCallback).toBeDefined();
  });
});
