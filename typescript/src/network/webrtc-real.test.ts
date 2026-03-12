/**
 * Real E2E Tests for WebRTC using @roamhq/wrtc
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { RTCPeerConnection, RTCSessionDescription, RTCIceCandidate } from '@roamhq/wrtc';
import { WebRTCClient } from './webrtc';
import { WebRTCServer } from './webrtc-server';
import { Account } from '../account';
import { FilesystemStorage } from '../storage/filesystem';
import * as fs from 'fs/promises';

const TEST_DIR = './test-data-webrtc-real';

// Make wrtc available globally
(global as any).RTCPeerConnection = RTCPeerConnection;
(global as any).RTCSessionDescription = RTCSessionDescription;
(global as any).RTCIceCandidate = RTCIceCandidate;

describe('Real WebRTC E2E Tests', () => {
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
    } catch (error) { /* Ignore cleanup errors */ }
  });

  it('should create peer connection', () => {
    const peer = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
    });
    
    expect(peer).toBeDefined();
    peer.close();
  });

  it('should create data channel', () => {
    const peer = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
    });
    
    const dc = peer.createDataChannel('mau', { ordered: true });
    expect(dc).toBeDefined();
    expect(dc.label).toBe('mau');
    
    peer.close();
  });

  it('should establish connection between two peers', async () => {
    return new Promise<void>((resolve, reject) => {
      const peer1 = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      const peer2 = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      // Exchange ICE candidates
      peer1.onicecandidate = (event) => {
        if (event.candidate) {
          peer2.addIceCandidate(event.candidate);
        }
      };

      peer2.onicecandidate = (event) => {
        if (event.candidate) {
          peer1.addIceCandidate(event.candidate);
        }
      };

      // Create data channel on peer1
      peer1.createDataChannel('test');

      dc1.onopen = () => {
        dc1.send('Hello from Peer1');
      };

      dc1.onmessage = (event) => {
        expect(event.data).toBe('Hello from Peer2');
        peer1.close();
        peer2.close();
        resolve();
      };

      // Wait for data channel on peer2
      peer2.ondatachannel = (event) => {
        const dc2 = event.channel;
        
        dc2.onmessage = (msgEvent) => {
          expect(msgEvent.data).toBe('Hello from Peer1');
          dc2.send('Hello from Peer2');
        };
      };

      // Exchange offers/answers
      peer1.createOffer().then((offer) => {
        return peer1.setLocalDescription(offer);
      }).then(() => {
        return peer2.setRemoteDescription(peer1.localDescription!);
      }).then(() => {
        return peer2.createAnswer();
      }).then((answer) => {
        return peer2.setLocalDescription(answer);
      }).then(() => {
        return peer1.setRemoteDescription(peer2.localDescription!);
      }).catch(reject);

      // Timeout
      setTimeout(() => {
        peer1.close();
        peer2.close();
        reject(new Error('Connection timeout'));
      }, 10000);
    });
  }, 15000);

  it('should exchange binary data', async () => {
    return new Promise<void>((resolve, reject) => {
      const peer1 = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      const peer2 = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      peer1.onicecandidate = (event) => {
        if (event.candidate) {
          peer2.addIceCandidate(event.candidate);
        }
      };

      peer2.onicecandidate = (event) => {
        if (event.candidate) {
          peer1.addIceCandidate(event.candidate);
        }
      };

      const testData = new Uint8Array([1, 2, 3, 4, 5]);
      peer1.createDataChannel('test');

      dc1.onopen = () => {
        dc1.send(testData);
      };

      peer2.ondatachannel = (event) => {
        const dc2 = event.channel;
        
        dc2.onmessage = (msgEvent) => {
          const received = new Uint8Array(msgEvent.data);
          expect(received).toEqual(testData);
          
          peer1.close();
          peer2.close();
          resolve();
        };
      };

      peer1.createOffer().then((offer) => {
        return peer1.setLocalDescription(offer);
      }).then(() => {
        return peer2.setRemoteDescription(peer1.localDescription!);
      }).then(() => {
        return peer2.createAnswer();
      }).then((answer) => {
        return peer2.setLocalDescription(answer);
      }).then(() => {
        return peer1.setRemoteDescription(peer2.localDescription!);
      }).catch(reject);

      setTimeout(() => {
        peer1.close();
        peer2.close();
        reject(new Error('Timeout'));
      }, 10000);
    });
  }, 15000);

  it('should handle connection state changes', async () => {
    return new Promise<void>((resolve, reject) => {
      const peer1 = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      const peer2 = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
      });

      peer1.onicecandidate = (event) => {
        if (event.candidate) {
          peer2.addIceCandidate(event.candidate);
        }
      };

      peer2.onicecandidate = (event) => {
        if (event.candidate) {
          peer1.addIceCandidate(event.candidate);
        }
      };

      peer1.onconnectionstatechange = () => {
        if (peer1.connectionState === 'connected') {
          expect(peer1.connectionState).toBe('connected');
          peer1.close();
          peer2.close();
          resolve();
        }
      };

      peer1.createDataChannel('test');

      peer1.createOffer().then((offer) => {
        return peer1.setLocalDescription(offer);
      }).then(() => {
        return peer2.setRemoteDescription(peer1.localDescription!);
      }).then(() => {
        return peer2.createAnswer();
      }).then((answer) => {
        return peer2.setLocalDescription(answer);
      }).then(() => {
        return peer1.setRemoteDescription(peer2.localDescription!);
      }).catch(reject);

      setTimeout(() => {
        peer1.close();
        peer2.close();
        reject(new Error('State change timeout'));
      }, 10000);
    });
  }, 15000);

  it('should use WebRTCClient and WebRTCServer', async () => {
    const server = new WebRTCServer(serverAccount, storage);
    const client = new WebRTCClient(
      clientAccount,
      storage,
      serverAccount.getFingerprint()
    );

    const offer = await client.createOffer();
    expect(offer).toBeDefined();
    expect(offer.type).toBe('offer');

    const answer = await server.acceptConnection(
      clientAccount.getFingerprint(),
      offer
    );
    expect(answer).toBeDefined();
    expect(answer.type).toBe('answer');

    await client.completeConnection(answer);

    // Give time for connection to establish
    await new Promise(resolve => setTimeout(resolve, 100));

    client.close();
    server.closeConnection(clientAccount.getFingerprint());
  });

  it('should handle multiple concurrent connections', async () => {
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

    const offer1 = await client1.createOffer();
    const offer2 = await client2.createOffer();

    const answer1 = await server.acceptConnection(
      clientAccount.getFingerprint(),
      offer1
    );
    const answer2 = await server.acceptConnection(
      client2Account.getFingerprint(),
      offer2
    );

    await client1.completeConnection(answer1);
    await client2.completeConnection(answer2);

    const connections = server.getConnections();
    expect(connections).toContain(clientAccount.getFingerprint());
    expect(connections).toContain(client2Account.getFingerprint());

    client1.close();
    client2.close();
    server.closeConnection(clientAccount.getFingerprint());
    server.closeConnection(client2Account.getFingerprint());

    try {
      await fs.rm(TEST_DIR + '/client2', { recursive: true });
    } catch (error) { /* Ignore cleanup errors */ }
  });

  it('should close connections gracefully', () => {
    const peer = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
    });

    peer.createDataChannel('test');
    
    expect(() => {
      peer.close();
    }).not.toThrow();
  });
});
