# Mau TypeScript Implementation

TypeScript/JavaScript implementation of the [Mau P2P social network protocol](https://github.com/mau-network/mau).

**Works in both Browser and Node.js environments** with pluggable storage backends.

## Features

- ✅ **Universal**: Works in browser (IndexedDB) and Node.js (filesystem)
- ✅ **OpenPGP**: Ed25519 and RSA key generation, signing, and encryption
- ✅ **P2P Sync**: HTTP-based file synchronization with peers
- ✅ **Versioning**: Automatic content versioning with SHA-256 checksums
- ✅ **Type-safe**: Full TypeScript definitions
- ✅ **Zero dependencies on native modules**: Pure JavaScript crypto

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
