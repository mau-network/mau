# AGENTS.md - TypeScript Implementation Guide

This document provides guidance for AI agents working on the Mau TypeScript implementation.

## Project Structure

```
typescript/
├── src/                    # Source code
│   ├── account.ts         # Account/identity management
│   ├── client.ts          # HTTP client for peer sync
│   ├── server.ts          # HTTP server for serving files
│   ├── file.ts            # File operations with encryption
│   ├── index.ts           # Public API exports
│   ├── crypto/            # PGP encryption/signing
│   ├── network/           # P2P networking (WebRTC, resolvers)
│   ├── storage/           # Storage backends (filesystem, IndexedDB)
│   └── types/             # TypeScript type definitions
├── examples/              # Usage examples
├── dist/                  # Compiled JavaScript (git-ignored)
├── coverage/              # Test coverage reports (git-ignored)
├── README.md              # User documentation
├── BROWSER.md             # Browser testing guide
└── package.json           # Dependencies and scripts

Test files: `*.test.ts` alongside source files
```

## Core Principles

### 1. Universal Compatibility
- **Must work in both Node.js and Browser**
- Use feature detection, not environment detection
- Storage abstraction (`FilesystemStorage` vs `BrowserStorage`)
- Conditional imports for Node.js-only modules

### 2. Zero Native Dependencies
- Pure JavaScript/TypeScript only
- No C/C++ bindings, no native addons
- Use `wrtc` polyfill for Node.js WebRTC testing only (dev dependency)

### 3. Type Safety
- All public APIs must have TypeScript definitions
- Use strict mode (`"strict": true` in `tsconfig.json`)
- Export types from `src/types/index.ts`

### 4. Test Coverage
- Target: >50% branch coverage (current: ~44%)
- Every public method should have at least one test
- Integration tests for critical paths (sync, encryption)

### 5. Security First
- All files are PGP-encrypted before storage
- Signatures verified on read
- mTLS authentication for WebRTC connections
- No plaintext secrets in memory longer than necessary

## Development Workflow

### Building
```bash
npm run build          # TypeScript → dist/
npm run build:browser  # Vite bundle for browsers
```

### Testing
```bash
npm test                      # Run all tests with coverage
npm test -- --watch           # Watch mode
npm test src/file.test.ts     # Run specific test file
npm test -- --no-coverage     # Faster runs without coverage
```

### Linting
```bash
npm run lint                  # ESLint check
npm run format                # Prettier format
```

### Manual Testing
```bash
# Browser testing
npm run dev                   # Start dev server
# Open http://localhost:5173/test-standalone.html

# Node.js testing
node test-integration.mjs

# Automated browser testing
node test-browser.cjs
```

## Key Implementation Details

### Storage Abstraction
```typescript
interface Storage {
  exists(path: string): Promise<boolean>;
  readFile(path: string): Promise<Uint8Array>;
  writeFile(path: string, data: Uint8Array): Promise<void>;
  readText(path: string): Promise<string>;
  writeText(path: string, text: string): Promise<void>;
  readDir(path: string): Promise<string[]>;
  mkdir(path: string): Promise<void>;
  remove(path: string): Promise<void>;
  stat(path: string): Promise<{ size: number; isDirectory: boolean; modifiedTime?: number }>;
  join(...parts: string[]): string;
}
```

Implementations:
- **FilesystemStorage** (Node.js): Uses `fs/promises`
- **BrowserStorage** (Browser): Uses IndexedDB

### Peer Discovery Resolvers
```typescript
type FingerprintResolver = (
  fingerprint: Fingerprint,
  timeout?: number
) => Promise<string | null>;
```

Available resolvers:
- **staticResolver** ✅ Universal: Hardcoded address map
- **dhtResolver** ✅ Universal: Kademlia DHT via `/kad/find_peer` endpoint (HTTP-based)
- **dnsResolver** ⚠️ Node.js only: DNS TXT record lookup (`_mau.<fingerprint>.<domain>`)
- **mdnsResolver** ⚠️ Node.js only: Local network discovery (`_mau._tcp.local`)
- **combinedResolver** ✅ Universal: Try multiple resolvers in parallel

### File Encryption Flow
```
Write:
  data → sign (with private key) → encrypt (to public keys) → PGP armor → storage

Read:
  storage → PGP armor → decrypt (with private key) → verify (with public keys) → data
```

### WebRTC Architecture
- **WebRTCClient**: Initiates connections, creates offers
- **WebRTCServer**: Accepts connections, handles answers
- **mTLS**: Certificate verification after WebRTC data channel opens
- **HTTP-over-datachannel**: Text-based HTTP/1.1 protocol

## Common Patterns

### Creating Storage-aware Objects
```typescript
// Use factory methods that accept storage
const account = await Account.create(storage, rootDir, options);
const file = File.create(account, storage, 'filename.json');
const client = Client.create(account, storage, peer);
```

### Async/Await Everywhere
- All I/O operations are async
- Use `Promise.all()` for parallel operations
- Use `Promise.race()` for timeouts

### Error Handling
```typescript
// Custom error classes with codes
throw new PeerNotFoundError();  // code: 'PEER_NOT_FOUND'
throw new InvalidFileNameError('reason');  // code: 'INVALID_FILE_NAME'

// Catch specific errors
try {
  await client.sync();
} catch (err) {
  if (err instanceof PeerNotFoundError) {
    // Handle peer not found
  }
}
```

## Testing Patterns

### Unit Tests
```typescript
describe('FeatureName', () => {
  let storage: FilesystemStorage;
  let account: Account;

  beforeEach(async () => {
    storage = new FilesystemStorage();
    await fs.mkdir(TEST_DIR, { recursive: true });
    account = await Account.create(storage, TEST_DIR, options);
  });

  afterEach(async () => {
    await fs.rm(TEST_DIR, { recursive: true, force: true });
  });

  it('should do something', async () => {
    // Test implementation
  });
});
```

### Integration Tests
- Use real filesystem (create temp directories)
- Clean up in `afterEach`
- Test full workflows (create account → write file → sync → verify)

### Mocking
```typescript
// Use Jest mocks sparingly
const mockResolver = jest.fn().mockResolvedValue('peer:8080');
```

## Code Quality Guidelines

### Naming Conventions
- **Classes**: PascalCase (`Account`, `File`, `WebRTCClient`)
- **Functions/methods**: camelCase (`createAccount`, `writeJSON`)
- **Constants**: UPPER_SNAKE_CASE (`SERVER_RESULT_LIMIT`, `DHT_K`)
- **Private methods**: prefix with `_` (underscore) or use TypeScript `private`

### File Organization
```typescript
// 1. Imports (grouped: types, libraries, internal)
import type { Storage } from './types/index.js';
import * as fs from 'fs/promises';
import { Account } from './account.js';

// 2. Type definitions
interface InternalType {
  // ...
}

// 3. Constants
const DEFAULT_TIMEOUT = 30000;

// 4. Main class/functions
export class Thing {
  // ...
}

// 5. Helper functions (private)
function internalHelper() {
  // ...
}
```

### Comments
- Use JSDoc for public APIs
- Inline comments for complex logic only
- TODO comments must include issue number or context

### Async Best Practices
```typescript
// ✅ Good: Parallel operations
const [file1, file2] = await Promise.all([
  storage.readFile('file1.txt'),
  storage.readFile('file2.txt'),
]);

// ❌ Bad: Sequential when parallel is possible
const file1 = await storage.readFile('file1.txt');
const file2 = await storage.readFile('file2.txt');

// ✅ Good: Timeout handling
await Promise.race([
  operation(),
  new Promise((_, reject) => 
    setTimeout(() => reject(new Error('Timeout')), 5000)
  ),
]);
```

## Dependencies

### Production (Universal)
- **openpgp**: PGP encryption/signing (RFC 4880) - works everywhere
- **node-fetch**: HTTP client for Node.js (polyfill) - not needed in browser

### Optional (Node.js only)
- **dns2**: DNS client for TXT record lookups (requires UDP sockets)
- **multicast-dns**: mDNS/DNS-SD for local network discovery (requires UDP multicast)
- **k-bucket**: Kademlia routing table

**Note:** Optional dependencies use dynamic imports and fail gracefully in browser environments.

### Development
- **typescript**: TypeScript compiler
- **jest**: Test framework
- **@roamhq/wrtc**: WebRTC polyfill for Node.js testing
- **playwright**: Browser automation
- **vite**: Browser bundler

## When Making Changes

### Adding New Features
1. Update type definitions in `src/types/index.ts`
2. Implement in appropriate module
3. Export from `src/index.ts` if public API
4. Write tests (unit + integration if needed)
5. Update README.md with usage example
6. Update BROWSER.md if browser-specific

### Fixing Bugs
1. Write a failing test that reproduces the bug
2. Fix the bug
3. Verify test passes
4. Check for similar issues in related code

### Refactoring
1. Ensure tests pass before starting
2. Make small, incremental changes
3. Run tests after each change
4. Avoid changing behavior and structure simultaneously

### Performance Optimization
1. Profile first (don't guess)
2. Document the optimization with benchmarks
3. Avoid premature optimization
4. Test that behavior hasn't changed

## Common Gotchas

### Browser vs Node.js
```typescript
// ❌ Wrong: Environment detection
if (typeof window !== 'undefined') {
  // Browser code
}

// ✅ Right: Feature detection + storage abstraction
const storage = await createStorage();  // Auto-detects environment
```

**Node.js-Only Features:**

Some resolvers require Node.js-only capabilities (UDP sockets):

```typescript
// ⚠️ Node.js only - Will return null in browser
const dnsRes = dnsResolver('mau.network');
const mdnsRes = mdnsResolver();

// ✅ Works everywhere (browser + Node.js)
const staticRes = staticResolver(knownPeers);
const dhtRes = dhtResolver(['bootstrap1:443']);  // Uses fetch()

// ✅ Browser-safe combined resolver
const resolver = combinedResolver([
  staticResolver(knownPeers),
  dhtResolver(['bootstrap1:443']),
  // dnsResolver/mdnsResolver gracefully fail in browser
]);
```

**Why DNS/mDNS don't work in browsers:**
- Require UDP socket access (not available in browsers for security)
- Browsers can only use fetch/WebSocket/WebRTC
- Use DHT resolver (HTTP-based) or static resolver instead

**Dependencies:**
- `dns2` and `multicast-dns` are marked as `optionalDependencies`
- Dynamic imports with try/catch handle missing modules gracefully
- Browser bundles won't include these Node.js-only modules

### Async Constructor
```typescript
// ❌ Wrong: Async constructor
class Thing {
  constructor() {
    this.init();  // Can't await in constructor
  }
}

// ✅ Right: Factory method
class Thing {
  private constructor() {}
  
  static async create(): Promise<Thing> {
    const instance = new Thing();
    await instance.init();
    return instance;
  }
}
```

### PGP Key Handling
```typescript
// ❌ Wrong: Storing unencrypted private keys
localStorage.setItem('privateKey', privateKeyArmor);

// ✅ Right: Always encrypted with passphrase
await account.save(passphrase);  // Encrypts before storage
```

### File Paths
```typescript
// ❌ Wrong: Hardcoded separators
const path = rootDir + '/' + filename;

// ✅ Right: Use storage.join()
const path = storage.join(rootDir, filename);
```

### WebRTC Cleanup
```typescript
// ❌ Wrong: Forget to close connections
const client = new WebRTCClient(...);
await client.connect();
// ... use client ...

// ✅ Right: Always close
try {
  const client = new WebRTCClient(...);
  await client.connect();
  // ... use client ...
} finally {
  client.close();
}
```

## Debugging Tips

### Enable Debug Logging
```typescript
// Set environment variable
DEBUG=mau:* npm test

// Or in code
localStorage.setItem('debug', 'mau:*');  // Browser
process.env.DEBUG = 'mau:*';             // Node.js
```

### Inspect PGP Operations
```typescript
// Verify signatures are working
const file = File.create(account, storage, 'test.json');
await file.writeJSON({ test: true });
const data = await file.read();  // Should not throw

// Check signature manually
const publicKeys = account.getAllPublicKeys();
const { verified } = await decryptAndVerify(armor, privateKey, publicKeys);
console.log('Signature verified:', verified);
```

### WebRTC Connection Issues
```typescript
// Log ICE candidates
client.on('icecandidate', (candidate) => {
  console.log('ICE candidate:', candidate);
});

// Check connection state
console.log('Connection state:', client.connectionState);
console.log('Data channel state:', client.dataChannelState);
```

### Storage Inspection
```typescript
// Browser: Check IndexedDB
// Open DevTools → Application → IndexedDB → mau-storage → files

// Node.js: Check filesystem
console.log(await storage.readDir(rootDir));
```

## Release Checklist

Before submitting PR:
- [ ] All tests pass (`npm test`)
- [ ] No linting errors (`npm run lint`)
- [ ] Coverage threshold met (>50% branches)
- [ ] README.md updated with new features
- [ ] Examples work in both Node.js and browser
- [ ] TypeScript compiles without errors (`npm run build`)
- [ ] Bundle size is reasonable (`npm run build:browser`)

## Resources

- **Mau Specification**: `../docs/` directory
- **Go Implementation**: `../` (reference implementation)
- **OpenPGP Spec**: RFC 4880
- **WebRTC Spec**: W3C WebRTC 1.0
- **Kademlia Paper**: Original DHT paper by Maymounkov & Mazières

## Getting Help

- **Issues**: Check existing issues on GitHub
- **Discussions**: Use GitHub Discussions for questions
- **Code Review**: Tag @emad-elsaid for review
- **Testing**: Run `npm test -- --verbose` for detailed output

---

**Remember:** This implementation must work in both browser and Node.js. Test both environments before submitting changes.
