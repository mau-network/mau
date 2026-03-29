/**
 * Real E2E Tests for WebRTC using @roamhq/wrtc
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { RTCPeerConnection, RTCSessionDescription, RTCIceCandidate } from '@roamhq/wrtc';
import { WebRTCClient } from './webrtc';
import { WebRTCServer } from './webrtc-server';
import { Account } from '../account';
import { BrowserStorage } from '../storage/browser';

const TEST_DIR = 'test-data-webrtc-real';

// Make wrtc available globally
interface GlobalWithWebRTC {
  RTCPeerConnection: typeof RTCPeerConnection;
  RTCSessionDescription: typeof RTCSessionDescription;
  RTCIceCandidate: typeof RTCIceCandidate;
}

(global as unknown as GlobalWithWebRTC).RTCPeerConnection = RTCPeerConnection;
(global as unknown as GlobalWithWebRTC).RTCSessionDescription = RTCSessionDescription;
(global as unknown as GlobalWithWebRTC).RTCIceCandidate = RTCIceCandidate;

describe('Real WebRTC E2E Tests', () => {
  let storage: BrowserStorage;
  let clientAccount: Account;
  let serverAccount: Account;

  beforeAll(async () => {
    storage = await BrowserStorage.create();

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
      await storage.remove(TEST_DIR);
    } catch {
      // Ignore cleanup errors
    }
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
      const dc1 = peer1.createDataChannel('test');

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
      const dc1 = peer1.createDataChannel('test');

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
    const server = new WebRTCServer(serverAccount, storage, { iceServers: [] });
    const client = new WebRTCClient(
      clientAccount,
      storage,
      serverAccount.getFingerprint(),
      { iceServers: [] }
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

    // Wait for ICE gathering to finish before closing to avoid
    // "ICE candidate on closed connection" errors when running in parallel.
    await client.waitForICEGathering();

    client.close();
    server.closeConnection(clientAccount.getFingerprint());
  });

  it('should handle multiple concurrent connections', async () => {
    const server = new WebRTCServer(serverAccount, storage, { iceServers: [] });

    const client1 = new WebRTCClient(
      clientAccount,
      storage,
      serverAccount.getFingerprint(),
      { iceServers: [] }
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
      serverAccount.getFingerprint(),
      { iceServers: [] }
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

    // Wait for ICE gathering on both clients before asserting/closing
    await Promise.all([client1.waitForICEGathering(), client2.waitForICEGathering()]);

    const connections = server.getConnections();
    expect(connections).toContain(clientAccount.getFingerprint());
    expect(connections).toContain(client2Account.getFingerprint());

    client1.close();
    client2.close();
    server.closeConnection(clientAccount.getFingerprint());
    server.closeConnection(client2Account.getFingerprint());

    try {
      await storage.remove(TEST_DIR + '/client2');
    } catch {
      // Ignore cleanup errors
    }
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
