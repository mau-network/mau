/**
 * Mau TypeScript Implementation
 * 
 * A peer-to-peer social network protocol implementation for browser and Node.js.
 * 
 * @packageDocumentation
 */

// Core exports
export { Account } from './account.js';
export { Client } from './client.js';
export { Server } from './server.js';
export { File } from './file.js';

// Storage exports
export { createStorage, FilesystemStorage, BrowserStorage } from './storage/index.js';

// Crypto exports
export * from './crypto/index.js';

// Network exports
export * from './network/index.js';

// Type exports
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
  ServerRequest,
  ServerResponse,
} from './types/index.js';

export {
  MAU_DIR_NAME,
  ACCOUNT_KEY_FILENAME,
  SYNC_STATE_FILENAME,
  FILE_PERM,
  DIR_PERM,
  HTTP_TIMEOUT_MS,
  SERVER_RESULT_LIMIT,
  URI_PROTOCOL_NAME,
  DHT_B,
  DHT_K,
  DHT_ALPHA,
  DHT_STALL_PERIOD_MS,
  DHT_PING_MIN_BACKOFF_MS,
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

/**
 * Convenience function to create a new account
 */
export async function createAccount(
  rootPath: string,
  name: string,
  email: string,
  passphrase: string,
  options: { algorithm?: 'ed25519' | 'rsa'; rsaBits?: 2048 | 4096 } = {}
) {
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
export async function loadAccount(rootPath: string, passphrase: string) {
  const storage = await createStorage();
  return await Account.load(storage, rootPath, passphrase);
}
