/**
 * WebRTC Polyfill for Node.js
 * Wraps node-datachannel to match browser RTCPeerConnection API
 */

import { PeerConnection as NodePeerConnection } from 'node-datachannel';

// Polyfill RTCPeerConnection for Node.js environment
if (typeof global.RTCPeerConnection === 'undefined') {
  class RTCPeerConnectionPolyfill {
    private peer: any;
    private localDescriptionPromise: Promise<any> | null = null;
    private localDescriptionResolve: ((desc: any) => void) | null = null;

    constructor(config?: RTCConfiguration) {
      // Convert browser RTCConfiguration to node-datachannel RtcConfig
      const nodeConfig: any = {};
      
      if (config?.iceServers) {
        nodeConfig.iceServers = config.iceServers.map((server: any) => {
          if (typeof server.urls === 'string') {
            return server.urls;
          } else if (Array.isArray(server.urls)) {
            return server.urls[0];
          }
          return 'stun:stun.l.google.com:19302';
        });
      }

      // node-datachannel requires (name, config)
      this.peer = new NodePeerConnection('peer-' + Date.now(), nodeConfig);

      // Set up local description handler
      this.peer.onLocalDescription((sdp: string, type: string) => {
        if (this.localDescriptionResolve) {
          this.localDescriptionResolve({ type, sdp });
          this.localDescriptionResolve = null;
        }
      });
    }

    createDataChannel(label: string, options?: any) {
      return this.peer.createDataChannel(label, options);
    }

    async createOffer() {
      // node-datachannel doesn't have createOffer, but setting local description triggers it
      this.localDescriptionPromise = new Promise((resolve, reject) => {
        this.localDescriptionResolve = resolve;
        
        // Timeout after 5 seconds
        setTimeout(() => {
          if (this.localDescriptionResolve) {
            reject(new Error('createOffer timeout'));
            this.localDescriptionResolve = null;
          }
        }, 5000);
      });

      try {
        // Trigger offer creation by setting empty local description
        this.peer.setLocalDescription('offer');
      } catch (error) {
        this.localDescriptionResolve = null;
        throw error;
      }

      return this.localDescriptionPromise;
    }

    async createAnswer() {
      this.localDescriptionPromise = new Promise((resolve, reject) => {
        this.localDescriptionResolve = resolve;
        
        // Timeout after 5 seconds
        setTimeout(() => {
          if (this.localDescriptionResolve) {
            reject(new Error('createAnswer timeout'));
            this.localDescriptionResolve = null;
          }
        }, 5000);
      });

      try {
        // Trigger answer creation
        this.peer.setLocalDescription('answer');
      } catch (error) {
        this.localDescriptionResolve = null;
        throw error;
      }

      return this.localDescriptionPromise;
    }

    async setLocalDescription(desc: any) {
      // In node-datachannel, this happens via onLocalDescription callback
      // Already handled in createOffer/createAnswer
      return Promise.resolve();
    }

    async setRemoteDescription(desc: any) {
      return this.peer.setRemoteDescription(desc.sdp, desc.type);
    }

    async addIceCandidate(candidate: any) {
      return this.peer.addRemoteCandidate(candidate.candidate, candidate.sdpMid);
    }

    set ondatachannel(handler: any) {
      this.peer.onDataChannel(handler);
    }

    set onicecandidate(handler: any) {
      this.peer.onLocalCandidate((candidate: string, mid: string) => {
        handler({
          candidate: { candidate, sdpMid: mid, sdpMLineIndex: 0 }
        });
      });
    }

    close() {
      if (this.peer.close) {
        this.peer.close();
      }
    }
  }

  (global as any).RTCPeerConnection = RTCPeerConnectionPolyfill;
}
