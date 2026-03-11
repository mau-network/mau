# Mau TypeScript Implementation

TypeScript/JavaScript implementation of the [Mau P2P social network protocol](https://github.com/mau-network/mau).

**Works in both Browser and Node.js environments** with pluggable storage backends.

## Features

- ✅ **Universal**: Works in browser (IndexedDB) and Node.js (filesystem)
- ✅ **WebRTC P2P**: Native browser-to-browser communication over data channels
- ✅ **PGP Authentication**: Challenge-response authentication over WebRTC
- ✅ **OpenPGP**: Ed25519 and RSA key generation, signing, and encryption
- ✅ **P2P Sync**: HTTP-style protocol over WebRTC or traditional HTTP
- ✅ **Versioning**: Automatic content versioning with SHA-256 checksums
- ✅ **Type-safe**: Full TypeScript definitions
- ✅ **Zero native dependencies**: Pure JavaScript, runs anywhere

## Security Note

⚠️ **HTTP Client**: The HTTP client does not yet implement mTLS authentication. For authenticated P2P connections, use the WebRTC client which implements PGP-based challenge-response authentication. Traditional mTLS over HTTPS requires X.509 certificates and is planned for a future release.

## Browser vs Node.js

### ✅ Works Everywhere (Browser + Node.js)
- Account management (PGP key generation)
- File encryption/decryption
- WebRTC P2P connections
- HTTP client sync
- Static peer resolver
- DHT resolver (HTTP-based)

### ⚠️ Node.js Only
- **DNS resolver** - Requires UDP sockets (not available in browsers)
- **HTTP Server** - Use WebRTC or browser extensions for serving files

**Browser Peer Discovery:** Use `staticResolver` or `dhtResolver` (HTTP-based). The DNS resolver will gracefully return `null` in browser environments.

## Installation

```bash
npm install @mau-network/mau
```

## Quick Start

### Node.js Example

```typescript
import { createAccount, loadAccount, File, Client } from '@mau-network/mau';

// Create a new account
const account = await createAccount(
  './my-mau-data',    // Root directory
  'Alice',            // Name
  'alice@mau.network', // Email
  'strong-passphrase'  // Passphrase
);

console.log('Account fingerprint:', account.getFingerprint());
console.log('Public key:', account.getPublicKey());

// Write a post
const file = File.create(account, account.storage, 'hello.json');
await file.writeJSON({
  '@type': 'SocialMediaPosting',
  headline: 'Hello, Mau!',
  articleBody: 'My first post on Mau',
  datePublished: new Date().toISOString(),
});

// List all files
const files = await File.list(account, account.storage);
for (const f of files) {
  console.log('File:', f.getName());
}

// Add a friend
const friendPublicKey = '-----BEGIN PGP PUBLIC KEY BLOCK-----\n...';
const friendFingerprint = await account.addFriend(friendPublicKey);

// Sync with a peer
const client = Client.create(
  account,
  account.storage,
  { fingerprint: friendFingerprint, address: '192.168.1.100:8080' }
);

const stats = await client.sync();
console.log('Downloaded:', stats.downloaded, 'Updated:', stats.updated);
```

### Browser Example

```typescript
import { createAccount, File } from '@mau-network/mau';

// Create account (uses IndexedDB automatically)
const account = await createAccount(
  'mau-data',         // Root path in IndexedDB
  'Bob',
  'bob@mau.network',
  'strong-passphrase'
);

// Write a post
const file = File.create(account, account.storage, 'post-1.json');
await file.writeJSON({
  '@type': 'SocialMediaPosting',
  headline: 'Posted from browser!',
  datePublished: new Date().toISOString(),
});

// Read it back
const content = await file.readJSON();
console.log(content);
```

### Server Example (Express)

```typescript
import express from 'express';
import { loadAccount, Server } from '@mau-network/mau';

const app = express();

const account = await loadAccount('./my-mau-data', 'my-passphrase');
const server = new Server(account, account.storage);

// Mount Mau server
app.use(server.expressMiddleware());

app.listen(8080, () => {
  console.log('Mau server listening on port 8080');
  console.log('Fingerprint:', account.getFingerprint());
});
```

### Server Example (Node.js http)

```typescript
import http from 'http';
import { loadAccount, Server } from '@mau-network/mau';

const account = await loadAccount('./my-mau-data', 'my-passphrase');
const server = new Server(account, account.storage);

const httpServer = http.createServer(server.nodeHandler());

httpServer.listen(8080, () => {
  console.log('Mau server listening on port 8080');
});
```

### Browser P2P with WebRTC

```typescript
import { createAccount, WebRTCServer, WebRTCClient, WebSocketSignaling } from '@mau-network/mau';

// ============== Peer A: Start Server ==============
const aliceAccount = await createAccount('mau-alice', 'Alice', 'alice@mau.network', 'secret');
const aliceFpr = aliceAccount.getFingerprint();

// Start WebRTC server (accepts incoming connections)
const server = new WebRTCServer(aliceAccount, aliceAccount.storage);

// Connect to signaling server
const signaling = new WebSocketSignaling('wss://signaling.mau.network', aliceFpr);

// Handle incoming connection offers
signaling.onMessage(async (message) => {
  if (message.type === 'offer') {
    const connectionId = `conn-${Date.now()}`;
    const answer = await server.acceptConnection(connectionId, message.data);
    
    // Send answer back through signaling
    await signaling.send({
      from: aliceFpr,
      to: message.from,
      type: 'answer',
      data: answer
    });
  }
});

console.log('Alice is ready:', aliceFpr);

// ============== Peer B: Connect to Alice ==============
const bobAccount = await createAccount('mau-bob', 'Bob', 'bob@mau.network', 'secret');
const bobFpr = bobAccount.getFingerprint();

// Follow Alice
await bobAccount.addFriend(aliceAccount.getPublicKey());

// Create WebRTC client
const client = new WebRTCClient(bobAccount, bobAccount.storage, aliceFpr);

// Create offer and send via signaling
const offer = await client.createOffer();
await signaling.send({
  from: bobFpr,
  to: aliceFpr,
  type: 'offer',
  data: offer
});

// Wait for answer
signaling.onMessage(async (message) => {
  if (message.type === 'answer' && message.from === aliceFpr) {
    await client.completeConnection(message.data);
    
    // Perform mTLS handshake
    const authenticated = await client.performMTLS();
    if (!authenticated) throw new Error('Authentication failed');
    
    // Now fetch files over WebRTC!
    const fileList = await client.fetchFileList();
    console.log('Files from Alice:', fileList);
    
    // Download a file
    const data = await client.downloadFile('hello.json');
    console.log('Downloaded:', data.length, 'bytes');
  }
});
```

### Signaling Server

Run a simple HTTP signaling server for WebRTC coordination:

```bash
# Start signaling server
npx tsx examples/signaling-server.ts 8080
```

Or deploy the `HTTPSignalingServer` to your infrastructure:

```typescript
import { HTTPSignalingServer } from '@mau-network/mau/examples/signaling-server';

const server = new HTTPSignalingServer();
await server.start(8080);

console.log('Signaling server running on port 8080');
```

For full examples, see:
- [`examples/browser-example.ts`](examples/browser-example.ts) - Complete browser P2P flow
- [`examples/signaling-server.ts`](examples/signaling-server.ts) - HTTP signaling server

## API Reference

### Account

```typescript
// Create new account
const account = await Account.create(storage, rootPath, {
  name: 'Alice',
  email: 'alice@mau.network',
  passphrase: 'strong-passphrase',
  algorithm: 'ed25519', // or 'rsa'
});

// Load existing account
const account = await Account.load(storage, rootPath, 'passphrase');

// Get account info
account.getFingerprint();  // => '3a7b2c...'
account.getName();          // => 'Alice'
account.getEmail();         // => 'alice@mau.network'
account.getPublicKey();     // => '-----BEGIN PGP PUBLIC KEY BLOCK-----...'

// Manage friends
await account.addFriend(armoredPublicKey);
await account.removeFriend(fingerprint);
account.getFriends();       // => ['fpr1', 'fpr2', ...]
account.isFriend(fpr);      // => true/false
```

### File

```typescript
// Create file reference
const file = File.create(account, storage, 'my-post.json');

// Write content (encrypted + signed automatically)
await file.writeText('Hello, world!');
await file.writeJSON({ message: 'Hello' });
await file.write(new Uint8Array([1, 2, 3]));

// Read content (decrypts + verifies automatically)
const text = await file.readText();
const json = await file.readJSON();
const data = await file.read();

// File operations
await file.delete();
const size = await file.getSize();
const checksum = await file.getChecksum();

// Versioning
const versions = await file.getVersions();
for (const version of versions) {
  console.log(await version.readText());
}

// List files
const files = await File.list(account, storage);
const friendFiles = await File.listFriend(account, storage, friendFingerprint);
```

### Client

```typescript
// Create client
const client = new Client(account, storage, peerFingerprint, {
  timeout: 30000,
  resolvers: [/* custom resolvers */],
});

// Or use convenience method
const client = Client.create(account, storage, {
  fingerprint: 'peer-fingerprint',
  address: '192.168.1.100:8080',
});

// Sync files
const stats = await client.sync();
// => { downloaded: 5, updated: 2, errors: 0 }

// Fetch file list
const files = await client.fetchFileList();
const filesAfter = await client.fetchFileList(new Date('2024-01-01'));

// Download specific file
const data = await client.downloadFile('post.json');
const version = await client.downloadFileVersion('post.json', 'abc123...');
```

### Server

```typescript
// Create server
const server = new Server(account, storage, {
  resultsLimit: 20,
  port: 8080,
});

// Express middleware
app.use(server.expressMiddleware());

// Node.js http handler
http.createServer(server.nodeHandler()).listen(8080);

// Handle requests manually
const response = await server.handleRequest({
  method: 'GET',
  url: '/p2p/3a7b2c.../post.json',
  path: '/p2p/3a7b2c.../post.json',
  query: {},
  headers: {},
});
```

### WebRTCServer

```typescript
// Create WebRTC server (for browser)
const server = new WebRTCServer(account, storage, {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  allowedPeers: ['allowed-fingerprint-1', 'allowed-fingerprint-2'], // optional whitelist
});

// Accept incoming connection
const answer = await server.acceptConnection(connectionId, offer);

// Add ICE candidate
await server.addIceCandidate(connectionId, candidate);

// Listen for signaling events (ICE candidates)
server.onSignaling(connectionId, (signal) => {
  // Send via signaling channel
  signalingChannel.send(signal);
});

// Get active connections
const connections = server.getConnections();

// Close connection
server.closeConnection(connectionId);

// Stop server
server.stop();
```

### WebRTCClient

```typescript
// Create WebRTC client
const client = new WebRTCClient(account, storage, peerFingerprint, {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  timeout: 30000,
});

// Create offer
const offer = await client.createOffer();

// Or accept offer and create answer
const answer = await client.acceptOffer(offer);

// Complete connection with answer
await client.completeConnection(answer);

// Add ICE candidate
await client.addIceCandidate(candidate);

// Perform mTLS authentication
const authenticated = await client.performMTLS();
if (!authenticated) throw new Error('Authentication failed');

// HTTP-style requests over data channel
const fileList = await client.fetchFileList();
const fileData = await client.downloadFile('post.json');
const versionData = await client.downloadFileVersion('post.json', 'hash123');

// Close connection
client.close();
```

### Signaling

```typescript
// WebSocket signaling (browser)
import { WebSocketSignaling } from '@mau-network/mau';

const signaling = new WebSocketSignaling('wss://signaling.example.com', fingerprint);

// Send signaling message
await signaling.send({
  from: myFingerprint,
  to: peerFingerprint,
  type: 'offer',
  data: offerData,
});

// Receive messages
signaling.onMessage((message) => {
  console.log('Received:', message.type, 'from', message.from);
});

// Close connection
signaling.close();

// HTTP signaling (polling-based)
import { HTTPSignaling } from '@mau-network/mau';

const signaling = new HTTPSignaling('https://signaling.example.com', fingerprint);

// Start polling for messages
signaling.startPolling();

// Send and receive messages (same interface as WebSocket)
signaling.onMessage((message) => { /* ... */ });
await signaling.send(message);

// Stop polling
signaling.stopPolling();

// Signaled connection helper
import { SignaledConnection } from '@mau-network/mau';

const connection = new SignaledConnection(signaling, myFingerprint, peerFingerprint);

// Callbacks for connection events
connection.onOffer(async (offer) => { /* handle offer */ });
connection.onAnswer(async (answer) => { /* handle answer */ });
connection.onICECandidate(async (candidate) => { /* handle ICE */ });

// Send connection events
await connection.sendOffer(offer);
await connection.sendAnswer(answer);
await connection.sendICECandidate(candidate);
```

### Storage

```typescript
// Auto-detect storage backend
const storage = await createStorage();

// Or use specific backend
import { FilesystemStorage, BrowserStorage } from '@mau-network/mau';

const fsStorage = new FilesystemStorage();
const browserStorage = new BrowserStorage();

// Storage interface
await storage.readFile(path);         // => Uint8Array
await storage.writeFile(path, data);  // => void
await storage.readText(path);         // => string
await storage.writeText(path, text);  // => void
await storage.exists(path);           // => boolean
await storage.mkdir(path);            // => void
await storage.readDir(path);          // => string[]
await storage.remove(path);           // => void
await storage.stat(path);             // => { size, isDirectory }
storage.join('a', 'b', 'c');          // => 'a/b/c'
```

## Architecture

### Storage Abstraction

Mau uses a storage abstraction layer that works in both environments:

- **Node.js**: Uses native `fs` module for filesystem operations
- **Browser**: Uses IndexedDB for persistent storage

Both implement the same `Storage` interface, making code portable.

### Browser P2P with WebRTC

In browser environments, Mau uses **WebRTC data channels** instead of traditional HTTP:

```
┌─────────────┐                    ┌─────────────┐
│  Browser A  │◄──WebRTC Channel──►│  Browser B  │
│             │                    │             │
│ WebRTCServer│                    │WebRTCClient │
│   (Alice)   │                    │   (Bob)     │
└─────────────┘                    └─────────────┘
       │                                  │
       └────── Signaling Server ─────────┘
              (offer/answer/ICE)
```

**How it works:**

1. **Signaling Phase**: Peers exchange WebRTC offers/answers via a signaling server (WebSocket or HTTP)
2. **Connection**: WebRTC establishes a direct peer-to-peer connection with data channel
3. **mTLS Handshake**: Peers authenticate each other using PGP keys over the data channel
4. **HTTP Protocol**: HTTP-style requests/responses flow over the authenticated channel

**Key Components:**

- **WebRTCServer**: Accepts incoming connections in the browser
- **WebRTCClient**: Initiates connections to peers
- **Signaling**: Coordinates WebRTC connection setup (offers, answers, ICE candidates)
- **mTLS over Data Channel**: Authenticates peers using PGP signatures

**Benefits:**

- No server infrastructure needed (except lightweight signaling)
- True peer-to-peer communication
- End-to-end encrypted (WebRTC + PGP)
- Works across NATs/firewalls (with STUN/TURN)
- Same HTTP-style protocol as traditional client/server

### Encryption Flow

1. **Write**: Content → Sign with private key → Encrypt for recipients → Save
2. **Read**: Load → Decrypt with private key → Verify signature → Return content

All files are encrypted by default to:
- Your own public key (so you can read them)
- All friends' public keys (so they can read them)

### File Versioning

When a file is modified, the old version is automatically archived:

```
hello.json              ← Latest version
hello.json.versions/
  ├── abc123...         ← Version 1 (SHA-256 checksum)
  └── def456...         ← Version 2
```

## Protocol Compatibility

This implementation follows the [Mau specification](https://github.com/mau-network/mau/blob/master/README.md):

- ✅ PGP/OpenPGP for identity and encryption
- ✅ HTTP endpoints: `/p2p/<fpr>`, `/p2p/<fpr>/<file>`, `/p2p/<fpr>/<file>.versions/<hash>`
- ✅ JSON-LD / Schema.org for content structure
- ✅ SHA-256 checksums for content verification
- ⚠️ mTLS authentication (requires additional setup)
- ⚠️ Kademlia DHT (not yet implemented)
- ⚠️ mDNS discovery (not yet implemented)

## Limitations

- **Certificate Generation**: TLS certificate generation requires additional libraries (not included)
- **DHT**: Kademlia DHT not yet implemented
- **mDNS**: Local network discovery not yet implemented
- **UPnP**: Port forwarding not yet implemented

These are planned for future releases.

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Lint
npm run lint

# Format
npm run format
```

## License

GPL-3.0 - Same as the [main Mau project](https://github.com/mau-network/mau)

## Contributing

Contributions welcome! Please open issues and PRs on the [main Mau repository](https://github.com/mau-network/mau).

## Resources

- [Mau Specification](https://github.com/mau-network/mau)
- [Documentation](https://github.com/mau-network/mau/tree/master/docs)
- [Schema.org Vocabulary](https://schema.org/docs/full.html)
- [OpenPGP.js](https://openpgpjs.org/)
