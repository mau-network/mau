/**
 * Network Module - Public API
 */

// Resolver functions for peer discovery
export {
  staticResolver,
  dhtResolver,
  combinedResolver,
  retryResolver,
} from './resolvers.js';

// WebRTC client for P2P connections
export { WebRTCClient } from './webrtc.js';
export type { WebRTCConfig } from './webrtc.js';

// WebRTC server for accepting connections
export { WebRTCServer } from './webrtc-server.js';
export type { WebRTCServerConfig, WebRTCConnection } from './webrtc-server.js';

// Signaling mechanisms for WebRTC coordination
export {
  LocalSignalingServer,
  WebSocketSignaling,
  HTTPSignaling,
  SignaledConnection,
} from './signaling.js';
export type { SignalingMessage } from './signaling.js';

// Kademlia DHT for distributed peer discovery
export { KademliaDHT } from './dht.js';
