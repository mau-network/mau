# API Reference

Complete TypeScript API reference for the Mau peer-to-peer protocol implementation.

## Table of Contents

- [Core Classes](#core-classes)
  - [Account](#account)
  - [Client](#client)
  - [Server](#server)
  - [File](#file)
- [Storage](#storage)
- [Networking](#networking)
  - [Peer Discovery](#peer-discovery)
  - [WebRTC](#webrtc)
  - [DHT (Distributed Hash Table)](#dht-distributed-hash-table)
  - [Signaling](#signaling)
- [Types](#types)
- [Error Classes](#error-classes)
- [Convenience Functions](#convenience-functions)

---

## Core Classes

### Account

**Import:** `import { Account } from 'mau'`

Manages user identity, PGP keys, and friends (keyring).

#### Static Methods

##### `Account.create(storage, rootPath, options)`

Creates a new Mau account with a fresh PGP key pair.

**Parameters:**
- `storage: Storage` - Storage backend (IndexedDB/filesystem)
- `rootPath: string` - Root directory for account data
- `options: AccountOptions`
  - `name: string` - Account holder's name
  - `email: string` - Account holder's email
  - `passphrase: string` - Password to encrypt private key
  - `algorithm?: 'ed25519' | 'rsa'` - Key algorithm (default: `'ed25519'`)
  - `rsaBits?: 2048 | 4096` - RSA key size if using RSA (default: `4096`)

**Returns:** `Promise<Account>`

**Example:**
```typescript
const storage = await createStorage();
const account = await Account.create(storage, '/account', {
  name: 'Alice',
  email: 'alice@example.com',
  passphrase: 'strong-password',
  algorithm: 'ed25519'
});
```

##### `Account.load(storage, rootPath, passphrase)`

Loads an existing account from storage.

**Parameters:**
- `storage: Storage` - Storage backend
- `rootPath: string` - Account root directory
- `passphrase: string` - Password to decrypt private key

**Returns:** `Promise<Account>`

**Throws:**
- `NoIdentityError` - Account doesn't exist at `rootPath`
- `IncorrectPassphraseError` - Wrong passphrase

**Example:**
```typescript
const account = await Account.load(storage, '/account', 'password');
```

#### Instance Methods

##### `getFingerprint()`

Returns the account's PGP fingerprint (unique identifier).

**Returns:** `Fingerprint` (lowercase hex string)

##### `getPublicKey()`

Returns the public key in ASCII-armored format.

**Returns:** `string`

##### `getName()`

Returns the account holder's name.

**Returns:** `string`

##### `getEmail()`

Returns the account holder's email.

**Returns:** `string`

##### `follow(fingerprint, publicKey)`

Adds a friend to the account's keyring.

**Parameters:**
- `fingerprint: Fingerprint` - Friend's fingerprint
- `publicKey: string` - Friend's armored public key

**Returns:** `Promise<void>`

**Example:**
```typescript
await account.follow(
  'a1b2c3d4e5f6...',
  '-----BEGIN PGP PUBLIC KEY BLOCK-----...'
);
```

##### `unfollow(fingerprint)`

Removes a friend from the keyring.

**Parameters:**
- `fingerprint: Fingerprint` - Friend's fingerprint

**Returns:** `Promise<void>`

##### `listFollowing()`

Lists all friends in the keyring.

**Returns:** `Fingerprint[]`

##### `createClient(peer, resolvers?, config?)`

Creates a client to sync with a specific peer.

**Parameters:**
- `peer: Fingerprint` - Target peer's fingerprint
- `resolvers?: FingerprintResolver[]` - Peer discovery resolvers
- `config?: ClientConfig` - Client configuration

**Returns:** `Client`

**Example:**
```typescript
const client = account.createClient(
  peerFingerprint,
  [staticResolver(peers), dhtResolver(dht)]
);
```

##### `createServer(config?)`

Creates a server to serve files to peers.

**Parameters:**
- `config?: ServerConfig` - Server configuration

**Returns:** `Server`

##### `writeFile(filename, data)`

Writes encrypted, signed JSON data.

**Parameters:**
- `filename: string` - File name (e.g., `'post.json'`)
- `data: any` - JSON-serializable data

**Returns:** `Promise<void>`

**Throws:**
- `InvalidFileNameError` - Invalid filename format

##### `readFile(fingerprint, filename)`

Reads and verifies a file (own or friend's).

**Parameters:**
- `fingerprint: Fingerprint` - File owner's fingerprint
- `filename: string` - File name

**Returns:** `Promise<any>` - Decrypted JSON data

**Throws:**
- `FriendNotFollowedError` - Trying to read non-friend's file

##### `listFiles(fingerprint?)`

Lists files for account or a friend.

**Parameters:**
- `fingerprint?: Fingerprint` - Target fingerprint (default: own)

**Returns:** `Promise<FileListItem[]>`

##### `deleteFile(filename)`

Deletes a file from the account.

**Parameters:**
- `filename: string` - File to delete

**Returns:** `Promise<void>`

##### `saveSyncState(peer, state)`

Persists sync state for a peer.

**Parameters:**
- `peer: Fingerprint` - Peer's fingerprint
- `state: SyncState` - Sync state object

**Returns:** `Promise<void>`

##### `loadSyncState(peer)`

Loads sync state for a peer.

**Parameters:**
- `peer: Fingerprint` - Peer's fingerprint

**Returns:** `Promise<SyncState | null>`

---

### Client

**Import:** `import { Client } from 'mau'`

P2P HTTP client for file synchronization with peers.

#### Constructor

```typescript
new Client(account, storage, peer, config?)
```

**Note:** Usually created via `account.createClient()`.

#### Instance Methods

##### `sync()`

Synchronizes files from the peer.

**Returns:** `Promise<{ downloaded: number; skipped: number }>`

**Example:**
```typescript
const stats = await client.sync();
console.log(`Downloaded ${stats.downloaded} new files`);
```

##### `listFiles()`

Lists files available on the peer.

**Returns:** `Promise<FileListItem[]>`

---

### Server

**Import:** `import { Server } from 'mau'`

HTTP/TLS server for serving files to peers.

#### Constructor

```typescript
new Server(account, config?)
```

**Note:** Usually created via `account.createServer()`.

**Config:**
- `port?: number` - Listen port (default: `8443`)
- `hostname?: string` - Listen hostname (default: `'0.0.0.0'`)
- `requestHandler?: (req: ServerRequest) => Promise<ServerResponse | null>` - Custom handler

#### Instance Methods

##### `start()`

Starts the server.

**Returns:** `Promise<void>`

**Example:**
```typescript
const server = account.createServer({ port: 8443 });
await server.start();
console.log('Server running on port 8443');
```

##### `stop()`

Stops the server.

**Returns:** `Promise<void>`

##### `port()`

Returns the listening port.

**Returns:** `number`

##### `hostname()`

Returns the listening hostname.

**Returns:** `string`

##### `address()`

Returns the full server address.

**Returns:** `string` (e.g., `'localhost:8443'`)

---

### File

**Import:** `import { File } from 'mau'`

Represents an encrypted, signed Mau file.

#### Static Methods

##### `File.encrypt(privateKey, publicKey, data)`

Encrypts and signs JSON data.

**Parameters:**
- `privateKey: PrivateKey` - Signer's private key
- `publicKey: PublicKey` - Recipient's public key (can be same as private)
- `data: any` - JSON-serializable data

**Returns:** `Promise<string>` - Encrypted message (ASCII-armored PGP)

##### `File.decrypt(privateKey, publicKey, data)`

Decrypts and verifies a file.

**Parameters:**
- `privateKey: PrivateKey` - Recipient's private key
- `publicKey: PublicKey` - Signer's public key
- `data: string` - Encrypted message

**Returns:** `Promise<any>` - Decrypted JSON data

**Throws:** Error if signature verification fails

##### `File.serialize(data)`

Serializes data to JSON string.

**Parameters:**
- `data: any` - Data to serialize

**Returns:** `string`

##### `File.deserialize(json)`

Deserializes JSON string to data.

**Parameters:**
- `json: string` - JSON string

**Returns:** `any`

---

## Storage

### createStorage

**Import:** `import { createStorage } from 'mau'`

Creates an appropriate storage backend for the environment.

**Returns:** `Promise<Storage>`

**Example:**
```typescript
const storage = await createStorage(); // IndexedDB in browser
```

### BrowserStorage

**Import:** `import { BrowserStorage } from 'mau'`

IndexedDB-based storage for browsers.

**Methods:** See `Storage` interface in Types section.

---

## Networking

### Peer Discovery

#### staticResolver

**Import:** `import { staticResolver } from 'mau'`

Resolves fingerprints from a static peer map.

**Signature:**
```typescript
function staticResolver(peers: Map<Fingerprint, string[]>): FingerprintResolver
```

**Parameters:**
- `peers: Map<Fingerprint, string[]>` - Map of fingerprint → addresses

**Returns:** `FingerprintResolver`

**Example:**
```typescript
const resolver = staticResolver(new Map([
  ['abc123...', ['peer1.local:8443', 'peer1.onion:8443']],
  ['def456...', ['peer2.local:8443']]
]));
```

#### dhtResolver

**Import:** `import { dhtResolver } from 'mau'`

Resolves fingerprints via Kademlia DHT.

**Signature:**
```typescript
function dhtResolver(dht: KademliaDHT): FingerprintResolver
```

**Parameters:**
- `dht: KademliaDHT` - DHT instance

**Returns:** `FingerprintResolver`

**Example:**
```typescript
const dht = new KademliaDHT(fingerprint);
await dht.start();
const resolver = dhtResolver(dht);
```

#### combinedResolver

**Import:** `import { combinedResolver } from 'mau'`

Combines multiple resolvers (tries in order).

**Signature:**
```typescript
function combinedResolver(...resolvers: FingerprintResolver[]): FingerprintResolver
```

**Parameters:**
- `...resolvers: FingerprintResolver[]` - Resolvers to combine

**Returns:** `FingerprintResolver`

**Example:**
```typescript
const resolver = combinedResolver(
  staticResolver(staticPeers),
  dhtResolver(dht)
);
```

#### retryResolver

**Import:** `import { retryResolver } from 'mau'`

Wraps a resolver with automatic retry logic.

**Signature:**
```typescript
function retryResolver(
  resolver: FingerprintResolver,
  options?: { retries?: number; minTimeout?: number; maxTimeout?: number }
): FingerprintResolver
```

**Parameters:**
- `resolver: FingerprintResolver` - Base resolver
- `options?: object`
  - `retries?: number` - Max retries (default: `3`)
  - `minTimeout?: number` - Min backoff (ms, default: `1000`)
  - `maxTimeout?: number` - Max backoff (ms, default: `10000`)

**Returns:** `FingerprintResolver`

**Example:**
```typescript
const resolver = retryResolver(dhtResolver(dht), {
  retries: 5,
  minTimeout: 2000
});
```

---

### WebRTC

#### WebRTCClient

**Import:** `import { WebRTCClient } from 'mau'`

WebRTC-based P2P client (no central server).

**Constructor:**
```typescript
new WebRTCClient(account, storage, config?)
```

**Config:**
- `signaling?: SignaledConnection` - Signaling channel
- `iceServers?: RTCIceServer[]` - STUN/TURN servers

**Methods:**
- `connect(peer: Fingerprint): Promise<RTCDataChannel>`
- `close(): void`

**Example:**
```typescript
const signaling = new WebSocketSignaling('wss://signal.example.com');
const client = new WebRTCClient(account, storage, { signaling });
const channel = await client.connect(peerFingerprint);
```

#### WebRTCServer

**Import:** `import { WebRTCServer } from 'mau'`

WebRTC-based P2P server.

**Constructor:**
```typescript
new WebRTCServer(account, config?)
```

**Config:**
- `signaling?: SignaledConnection` - Signaling channel
- `iceServers?: RTCIceServer[]` - STUN/TURN servers

**Methods:**
- `start(): Promise<void>`
- `stop(): Promise<void>`

**Example:**
```typescript
const signaling = new WebSocketSignaling('wss://signal.example.com');
const server = new WebRTCServer(account, { signaling });
await server.start();
```

---

### DHT (Distributed Hash Table)

#### KademliaDHT

**Import:** `import { KademliaDHT } from 'mau'`

Kademlia-based distributed hash table for peer discovery.

**Constructor:**
```typescript
new KademliaDHT(nodeId: Fingerprint, options?)
```

**Options:**
- `k?: number` - Bucket size (default: `20`)
- `alpha?: number` - Concurrency parameter (default: `3`)
- `bucketRefreshInterval?: number` - Refresh interval (ms, default: `3600000`)

**Methods:**

##### `start()`

Starts the DHT node.

**Returns:** `Promise<void>`

##### `stop()`

Stops the DHT node.

**Returns:** `Promise<void>`

##### `bootstrap(peers: string[])`

Bootstraps the DHT with known peers.

**Parameters:**
- `peers: string[]` - Bootstrap peer addresses

**Returns:** `Promise<void>`

##### `announce(key: string, value: string)`

Announces a key-value pair to the DHT.

**Parameters:**
- `key: string` - Key (usually fingerprint)
- `value: string` - Value (usually address)

**Returns:** `Promise<void>`

##### `lookup(key: string)`

Looks up values for a key.

**Parameters:**
- `key: string` - Key to look up

**Returns:** `Promise<string[]>` - Array of values

**Example:**
```typescript
const dht = new KademliaDHT(account.getFingerprint());
await dht.start();
await dht.bootstrap(['bootstrap.mau.network:8443']);
await dht.announce(account.getFingerprint(), 'my-address:8443');

const addresses = await dht.lookup(peerFingerprint);
```

---

### Signaling

Signaling channels coordinate WebRTC connections.

#### LocalSignalingServer

**Import:** `import { LocalSignalingServer } from 'mau'`

In-process signaling server (for testing).

**Constructor:**
```typescript
new LocalSignalingServer()
```

**Methods:**
- `connect(peerId: string): SignaledConnection`

#### WebSocketSignaling

**Import:** `import { WebSocketSignaling } from 'mau'`

WebSocket-based signaling client.

**Constructor:**
```typescript
new WebSocketSignaling(url: string)
```

**Parameters:**
- `url: string` - WebSocket URL (e.g., `'wss://signal.example.com'`)

#### HTTPSignaling

**Import:** `import { HTTPSignaling } from 'mau'`

HTTP polling-based signaling client.

**Constructor:**
```typescript
new HTTPSignaling(url: string, peerId: string)
```

**Parameters:**
- `url: string` - HTTP base URL
- `peerId: string` - Peer ID

---

## Types

### Core Types

#### Storage

Abstraction over filesystem/IndexedDB.

**Methods:**
- `join(...paths: string[]): string`
- `exists(path: string): Promise<boolean>`
- `mkdir(path: string): Promise<void>`
- `readText(path: string): Promise<string>`
- `writeText(path: string, content: string): Promise<void>`
- `readdir(path: string): Promise<string[]>`
- `remove(path: string): Promise<void>`

#### Fingerprint

PGP key fingerprint (40-char lowercase hex string).

```typescript
type Fingerprint = string;
```

#### Peer

Peer information.

```typescript
interface Peer {
  fingerprint: Fingerprint;
  addresses: string[];
}
```

#### FileListItem

File metadata.

```typescript
interface FileListItem {
  name: string;
  size: number;
  modTime: number; // Unix timestamp
  fingerprint: Fingerprint;
}
```

#### MauFile

Represents a Mau file structure.

```typescript
interface MauFile {
  data: any; // Decrypted JSON content
  signature: string; // PGP signature
  fingerprint: Fingerprint; // Signer's fingerprint
}
```

#### AccountOptions

Options for creating an account.

```typescript
interface AccountOptions {
  name: string;
  email: string;
  passphrase: string;
  algorithm?: 'ed25519' | 'rsa';
  rsaBits?: 2048 | 4096;
}
```

#### ClientConfig

Client configuration.

```typescript
interface ClientConfig {
  timeout?: number; // HTTP timeout (ms)
  dnsNames?: string[]; // DNS names for cert validation
  resolvers?: FingerprintResolver[]; // Peer discovery
  fetchImpl?: typeof fetch; // Custom fetch
}
```

#### ServerConfig

Server configuration.

```typescript
interface ServerConfig {
  port?: number;
  hostname?: string;
  requestHandler?: (req: ServerRequest) => Promise<ServerResponse | null>;
}
```

#### SyncState

Tracks sync progress with a peer.

```typescript
interface SyncState {
  lastSync: number; // Unix timestamp
  knownFiles: Set<string>; // File names
}
```

#### FingerprintResolver

Peer discovery function.

```typescript
type FingerprintResolver = (fingerprint: Fingerprint) => Promise<string[]>;
```

### Network Types

#### WebRTCConfig

WebRTC client configuration.

```typescript
interface WebRTCConfig {
  signaling?: SignaledConnection;
  iceServers?: RTCIceServer[];
}
```

#### WebRTCServerConfig

WebRTC server configuration.

```typescript
interface WebRTCServerConfig {
  signaling?: SignaledConnection;
  iceServers?: RTCIceServer[];
}
```

#### SignalingMessage

WebRTC signaling message.

```typescript
interface SignalingMessage {
  type: 'offer' | 'answer' | 'ice-candidate';
  from: string;
  to: string;
  data: any;
}
```

---

## Error Classes

All errors extend `MauError`.

### MauError

Base error class.

```typescript
class MauError extends Error {
  code: string;
}
```

### PassphraseRequiredError

Thrown when passphrase is missing.

**Code:** `'PASSPHRASE_REQUIRED'`

### IncorrectPassphraseError

Thrown when passphrase is wrong.

**Code:** `'INCORRECT_PASSPHRASE'`

### NoIdentityError

Thrown when account doesn't exist.

**Code:** `'NO_IDENTITY'`

### AccountAlreadyExistsError

Thrown when trying to create duplicate account.

**Code:** `'ACCOUNT_ALREADY_EXISTS'`

### InvalidFileNameError

Thrown for invalid filenames.

**Code:** `'INVALID_FILE_NAME'`

### FriendNotFollowedError

Thrown when accessing non-friend's data.

**Code:** `'FRIEND_NOT_FOLLOWED'`

### PeerNotFoundError

Thrown when peer cannot be resolved.

**Code:** `'PEER_NOT_FOUND'`

### IncorrectPeerCertificateError

Thrown when peer's certificate doesn't match.

**Code:** `'INCORRECT_PEER_CERTIFICATE'`

---

## Convenience Functions

### createAccount

**Import:** `import { createAccount } from 'mau'`

One-step account creation.

**Signature:**
```typescript
function createAccount(
  rootPath: string,
  name: string,
  email: string,
  passphrase: string,
  options?: { algorithm?: 'ed25519' | 'rsa'; rsaBits?: 2048 | 4096 }
): Promise<Account>
```

**Example:**
```typescript
const account = await createAccount(
  '/account',
  'Alice',
  'alice@example.com',
  'password123'
);
```

### loadAccount

**Import:** `import { loadAccount } from 'mau'`

One-step account loading.

**Signature:**
```typescript
function loadAccount(rootPath: string, passphrase: string): Promise<Account>
```

**Example:**
```typescript
const account = await loadAccount('/account', 'password123');
```

---

## Next Steps

- **[HTTP API Reference](07-http-api.md)** - Server endpoints and protocols
- **[Building Social Apps](08-building-social-apps.md)** - Practical patterns
- **[Schema.org Types](12-schema-types.md)** - Structured data vocabulary
- **[Troubleshooting](13-troubleshooting.md)** - Common issues and fixes

---

**Related:**
- [Introduction](01-introduction.md)
- [Core Concepts](02-core-concepts.md)
- [Quick Start](03a-quickstart-gpg.md)
