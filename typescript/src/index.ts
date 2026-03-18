/**
 * Mau TypeScript Implementation
 * 
 * A peer-to-peer social network protocol implementation for browsers.
 * 
 * @packageDocumentation
 */

// ============================================================================
// Core Classes
// ============================================================================

export { Account } from './account.js';
export { Client } from './client.js';
export { Server } from './server.js';
export { File } from './file.js';

// ============================================================================
// Storage Backends
// ============================================================================

export { createStorage, BrowserStorage } from './storage/index.js';

// ============================================================================
// Networking & P2P
// ============================================================================

// Peer discovery resolvers
export {
  staticResolver,
  dhtResolver,
  combinedResolver,
  retryResolver,
} from './network/index.js';

// WebRTC client and server
export {
  WebRTCClient,
  WebRTCServer,
} from './network/index.js';

// Signaling for WebRTC coordination
export {
  LocalSignalingServer,
  WebSocketSignaling,
  HTTPSignaling,
  SignaledConnection,
} from './network/index.js';

// Distributed Hash Table (DHT)
export { KademliaDHT } from './network/index.js';

// ============================================================================
// Type Definitions
// ============================================================================

// Core types
export type {
  Storage,
  Fingerprint,
  Peer,
  FileListItem,
  MauFile,
  AccountOptions,
  ClientConfig,
  ServerConfig,
  SyncState,
  FingerprintResolver,
  CertificateInfo,
  FileListResponse,
} from './types/index.js';

// Server request/response types
export type { ServerRequest, ServerResponse } from './server.js';

// Network types
export type {
  WebRTCConfig,
  WebRTCServerConfig,
  WebRTCConnection,
  SignalingMessage,
} from './network/index.js';

// ============================================================================
// Error Classes
// ============================================================================

export {
  MauError,
  PassphraseRequiredError,
  IncorrectPassphraseError,
  NoIdentityError,
  AccountAlreadyExistsError,
  InvalidFileNameError,
  FriendNotFollowedError,
  PeerNotFoundError,
  IncorrectPeerCertificateError,
} from './types/index.js';

// ============================================================================
// Convenience Functions
// ============================================================================

/**
 * Convenience function to create a new account
 */
export async function createAccount(
  rootPath: string,
  name: string,
  email: string,
  passphrase: string,
  options: { algorithm?: 'ed25519' | 'rsa'; rsaBits?: 2048 | 4096 } = {}
): Promise<import('./account.js').Account> {
  const { createStorage } = await import('./storage/index.js');
  const { Account } = await import('./account.js');
  const storage = await createStorage();
  return await Account.create(storage, rootPath, {
    name,
    email,
    passphrase,
    ...options,
  });
}

/**
 * Convenience function to load an existing account
 */
export async function loadAccount(rootPath: string, passphrase: string): Promise<import('./account.js').Account> {
  const { createStorage } = await import('./storage/index.js');
  const { Account } = await import('./account.js');
  const storage = await createStorage();
  return await Account.load(storage, rootPath, passphrase);
}
