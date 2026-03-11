/**
 * Mau TypeScript Implementation - Type Definitions
 * 
 * Core types and interfaces for the Mau P2P social network protocol.
 */

/** Fingerprint represents a PGP key fingerprint (160-bit hex string) */
export type Fingerprint = string;

/** Peer represents a network peer with fingerprint and address */
export interface Peer {
  fingerprint: Fingerprint;
  address: string; // hostname:port or ip:port without protocol
}

/** FileListItem represents a file in the peer's file list */
export interface FileListItem {
  path: string;
  size: number;
  sum: string; // SHA-256 checksum
}

/** File metadata and versioning information */
export interface MauFile {
  path: string;
  name: string;
  isVersion: boolean;
}

/** Account configuration options */
export interface AccountOptions {
  name: string;
  email: string;
  passphrase: string;
  algorithm?: 'ed25519' | 'rsa';
  rsaBits?: 2048 | 4096;
  /** Key expiration in years (default: 2 years, 0 = no expiration) */
  expirationYears?: number;
}

/** Storage interface - abstraction for filesystem or localStorage */
export interface Storage {
  /** Check if a path exists */
  exists(path: string): Promise<boolean>;
  
  /** Read file as buffer */
  readFile(path: string): Promise<Uint8Array>;
  
  /** Write file from buffer */
  writeFile(path: string, data: Uint8Array): Promise<void>;
  
  /** Read file as text (UTF-8) */
  readText(path: string): Promise<string>;
  
  /** Write file from text (UTF-8) */
  writeText(path: string, text: string): Promise<void>;
  
  /** List directory entries */
  readDir(path: string): Promise<string[]>;
  
  /** Create directory (recursive) */
  mkdir(path: string): Promise<void>;
  
  /** Remove file or directory */
  remove(path: string): Promise<void>;
  
  /** Get file stats */
  stat(path: string): Promise<{ size: number; isDirectory: boolean; modifiedTime?: number }>;
  
  /** Join path components */
  join(...parts: string[]): string;
}

/** Client configuration */
export interface ClientConfig {
  timeout?: number; // HTTP timeout in milliseconds (default: 30000)
  dnsNames?: string[]; // DNS names for certificate
  resolvers?: FingerprintResolver[]; // Peer discovery resolvers
}

/** Server configuration */
export interface ServerConfig {
  bootstrapNodes?: Peer[]; // Known peers for DHT bootstrapping
  resultsLimit?: number; // Max results per query (default: 20)
  port?: number; // Listen port
  enableMDNS?: boolean; // Enable mDNS discovery
  enableDHT?: boolean; // Enable Kademlia DHT
  enableUPnP?: boolean; // Enable UPnP port forwarding
}

/** Sync state tracking */
export interface SyncState {
  [fingerprint: string]: number; // fingerprint -> last sync timestamp (Unix ms)
}

/** Fingerprint resolver function type */
export type FingerprintResolver = (
  fingerprint: Fingerprint,
  timeout?: number
) => Promise<string | null>;

/** TLS certificate information */
export interface CertificateInfo {
  cert: Uint8Array; // DER-encoded certificate
  key: Uint8Array; // DER-encoded private key
  fingerprint: Fingerprint;
}

/** HTTP response for file list */
export interface FileListResponse {
  files: FileListItem[];
}

/** Constants */
export const MAU_DIR_NAME = '.mau';
export const ACCOUNT_KEY_FILENAME = 'account.pgp';
export const SYNC_STATE_FILENAME = 'sync_state.json';
export const FILE_PERM = 0o600;
export const DIR_PERM = 0o700;
export const HTTP_TIMEOUT_MS = 30000;
export const SERVER_RESULT_LIMIT = 20;
export const URI_PROTOCOL_NAME = 'mau';

// Kademlia DHT constants
export const DHT_B = 160; // Number of bits for key space
export const DHT_K = 20; // Max bucket size (replication parameter)
export const DHT_ALPHA = 3; // Parallelism factor for lookups
export const DHT_STALL_PERIOD_MS = 3600000; // 1 hour
export const DHT_PING_MIN_BACKOFF_MS = 30000; // 30 seconds

/** Errors */
export class MauError extends Error {
  constructor(message: string, public code: string) {
    super(message);
    this.name = 'MauError';
  }
}

export class PassphraseRequiredError extends MauError {
  constructor() {
    super('Passphrase must be specified', 'PASSPHRASE_REQUIRED');
  }
}

export class IncorrectPassphraseError extends MauError {
  constructor() {
    super('Incorrect passphrase', 'INCORRECT_PASSPHRASE');
  }
}

export class NoIdentityError extends MauError {
  constructor() {
    super("Can't find identity", 'NO_IDENTITY');
  }
}

export class AccountAlreadyExistsError extends MauError {
  constructor() {
    super('Account already exists', 'ACCOUNT_ALREADY_EXISTS');
  }
}

export class InvalidFileNameError extends MauError {
  constructor(reason: string) {
    super(`Invalid file name: ${reason}`, 'INVALID_FILE_NAME');
  }
}

export class FriendNotFollowedError extends MauError {
  constructor() {
    super('Friend is not being followed', 'FRIEND_NOT_FOLLOWED');
  }
}

export class PeerNotFoundError extends MauError {
  constructor() {
    super("Couldn't find peer", 'PEER_NOT_FOUND');
  }
}

export class IncorrectPeerCertificateError extends MauError {
  constructor() {
    super('Incorrect peer certificate', 'INCORRECT_PEER_CERTIFICATE');
  }
}
