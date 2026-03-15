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

### 1. Browser-Only Architecture
- **Designed exclusively for modern browsers**
- Uses IndexedDB for storage (via `BrowserStorage`)
- WebRTC for peer-to-peer communication
- Tests run in Node.js with polyfills (`fake-indexeddb`, `@roamhq/wrtc`)

### 2. Zero Native Dependencies
- Pure JavaScript/TypeScript only
- No C/C++ bindings, no native addons
- Polyfills used only for testing (not in production builds)

### 3. Type Safety
- All public APIs must have TypeScript definitions
- Use strict mode (`"strict": true` in `tsconfig.json`)
- Export types from `src/types/index.ts`

### 4. Test Coverage
- Target: >50% branch coverage (current: ~41%)
- Every public method should have at least one test
- Integration tests for critical paths (sync, encryption)
- Focus on high-impact areas: `account.ts` (40%), `client.ts` (47%), `server.ts` (48%)
- Low-coverage areas need attention: `dht.ts` (13%), `signaling.ts` (20%), `webrtc-server.ts` (15%)

### 5. Security First
- All files are PGP-encrypted before storage
- Signatures verified on read
- PGP-based authentication for WebRTC connections
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

### Documentation
```bash
npm run docs                  # Generate HTML docs → docs/
npm run docs:serve            # Generate + serve at http://localhost:8000
```

### Manual Testing
```bash
# Browser testing
npm run dev                   # Start dev server at http://localhost:5173
npm run preview               # Preview production build

# Node.js testing
node test-integration.mjs

# Automated browser testing (Playwright)
npx playwright test
```

**Test Files:**
- `*.test.ts` - Jest unit/integration tests (run in Node.js with polyfills)
- Test environment uses `fake-indexeddb` and `@roamhq/wrtc` to simulate browser APIs

## Key Implementation Details

### Project Architecture

The codebase is organized into these key modules:

- **Core**: `account.ts`, `file.ts`, `client.ts`, `server.ts`, `index.ts`
- **Cryptography**: `crypto/pgp.ts`, `crypto/index.ts` - OpenPGP operations
- **Networking**: `network/webrtc.ts`, `network/webrtc-server.ts`, `network/resolvers.ts`, `network/dht.ts`, `network/signaling.ts`
- **Storage**: `storage/browser.ts`, `storage/index.ts` (filesystem storage removed)
- **Types**: `types/index.ts` - TypeScript type definitions

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

Implementation:
- **BrowserStorage**: Uses IndexedDB for persistent storage in browsers

### Peer Discovery Resolvers
```typescript
type FingerprintResolver = (
  fingerprint: Fingerprint,
  timeout?: number
) => Promise<string | null>;
```

Available resolvers:
- **staticResolver**: Hardcoded address map
- **dhtResolver**: Kademlia DHT via `/kad/find_peer` endpoint (HTTP-based)
- **combinedResolver**: Try multiple resolvers in parallel

**Note:** DNS and mDNS resolvers were removed as they require Node.js-specific UDP socket access.

### File Encryption Flow
```
Write:
  data → sign (with private key) → encrypt (to public keys) → PGP armor → storage

Read:
  storage → PGP armor → decrypt (with private key) → verify (with public keys) → data
```

### WebRTC Architecture
- **WebRTCClient** (`network/webrtc.ts`): Initiates connections, creates offers
- **WebRTCServer** (`network/webrtc-server.ts`): Accepts connections, handles answers
- **Signaling** (`network/signaling.ts`): Coordinates peer connection establishment
- **mTLS**: PGP-based challenge-response authentication after data channel opens
- **HTTP-over-datachannel**: Text-based HTTP/1.1 protocol for file synchronization

**Note:** WebRTC implementation uses native browser APIs. In Node.js tests, `@roamhq/wrtc` provides polyfill.

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
  let storage: BrowserStorage;
  let account: Account;

  beforeEach(async () => {
    storage = await BrowserStorage.create();
    account = await Account.create(storage, TEST_DIR, options);
  });

  afterEach(async () => {
    try {
      await storage.remove(TEST_DIR);
    } catch {
      // Ignore cleanup errors
    }
  });

  it('should do something', async () => {
    // Test implementation
  });
});
```

### Integration Tests
- Use BrowserStorage (backed by fake-indexeddb in tests)
- Clean up in `afterEach` using `storage.remove()`
- Test full workflows (create account → write file → sync → verify)

### Mocking
```typescript
// Use Jest mocks sparingly
const mockResolver = jest.fn().mockResolvedValue('peer:8080');
```

## TypeScript Configuration

- **Target**: ES2022 (modern JavaScript features)
- **Module System**: ES2022 modules (`.js` extensions in imports)
- **Strict Mode**: Enabled (all strict type checks)
- **Output**: `dist/` directory with declaration files (`.d.ts`)
- **Source Maps**: Enabled for debugging

**Important:** All imports must use `.js` extensions even for `.ts` files:
```typescript
// ✅ Correct
import { Account } from './account.js';

// ❌ Wrong
import { Account } from './account';
```

## API Export Policy

### Public API (Exported from `src/index.ts`)

**Core Classes:**
- `Account`, `Client`, `Server`, `File` - Main user-facing classes
- `BrowserStorage`, `createStorage` - Storage backend

**Networking:**
- `WebRTCClient`, `WebRTCServer` - WebRTC P2P communication
- `LocalSignalingServer`, `WebSocketSignaling`, `HTTPSignaling`, `SignaledConnection` - Signaling mechanisms
- `staticResolver`, `dhtResolver`, `combinedResolver`, `retryResolver` - Peer discovery
- `KademliaDHT` - Distributed hash table for peer discovery

**Utilities:**
- `validateFileName`, `normalizeFingerprint`, `formatFingerprint` - Crypto utilities
- `createAccount`, `loadAccount` - Convenience functions

**Types & Errors:**
- All type definitions (interfaces, type aliases)
- All error classes (`MauError`, `PeerNotFoundError`, etc.)

**NOT Exported (Internal Constants):**
- `MAU_DIR_NAME`, `ACCOUNT_KEY_FILENAME`, `SYNC_STATE_FILENAME` - Internal file structure
- `FILE_PERM`, `DIR_PERM` - Internal permissions
- `HTTP_TIMEOUT_MS`, `SERVER_RESULT_LIMIT` - Configure via `ClientConfig`/`ServerConfig` instead
- `DHT_B`, `DHT_K`, `DHT_ALPHA`, etc. - Internal DHT implementation details

### Internal API (Not Exported)

**Implementation Details:**
- PGP operations (`generateKeyPair`, `signAndEncrypt`, `decryptAndVerify`) - wrapped by Account class
- Internal helper functions in `crypto/pgp.ts`
- Low-level HTTP protocol handlers
- Storage implementation internals

### When Adding New Features

1. **Default to internal** - Only export what users need
2. **Use explicit exports** - Avoid `export *` wildcards in `src/index.ts`
3. **Document public APIs** - Use JSDoc for exported symbols
4. **Mark internal symbols** - Use JSDoc `@internal` tag or leading underscore for private methods

Example:
```typescript
// src/my-feature.ts

/**
 * @internal
 * Internal helper function - not exported from index.ts
 */
export function _internalHelper() { }

/**
 * Public API function - exported from index.ts
 * @param data Input data to process
 */
export function publicFeature(data: string) {
  return _internalHelper();
}
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

### Production (Browser)
- **openpgp**: PGP encryption/signing (RFC 4880)
- **@peculiar/x509**: X.509 certificate handling for WebRTC authentication
- **idb**: IndexedDB wrapper for browser storage
- **p-retry**: Robust retry mechanism for network operations
- **k-bucket**: Kademlia routing table for DHT

### Development
- **typescript**: TypeScript compiler (v5.3.3)
- **jest**: Test framework with ts-jest for TypeScript support
- **@roamhq/wrtc**: WebRTC polyfill for Node.js testing
- **fake-indexeddb**: IndexedDB mock for Node.js tests
- **vite**: Browser bundler and dev server
- **typedoc**: API documentation generator
- **eslint** + **prettier**: Code quality and formatting

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

## Performance Considerations

### Parallel Operations
Always parallelize independent async operations:

```typescript
// ✅ Fast: Parallel execution
const [account, config] = await Promise.all([
  loadAccount(dir, passphrase),
  loadConfig(configPath),
]);

// ❌ Slow: Sequential execution
const account = await loadAccount(dir, passphrase);
const config = await loadConfig(configPath);  // Waits unnecessarily
```

### Retry Strategies
Network operations use `p-retry` for resilience:

```typescript
import pRetry from 'p-retry';

await pRetry(
  async () => {
    const response = await fetch(url);
    if (!response.ok) throw new Error('HTTP error');
    return response;
  },
  { retries: 3, minTimeout: 1000 }
);
```

### Memory Management
- Dispose of large buffers (`Uint8Array`) after processing
- Close WebRTC connections explicitly to free resources
- Clear PGP keys from memory after use when possible

## Common Gotchas

### Storage Usage
```typescript
// ✅ Correct: Always use BrowserStorage
const storage = await createStorage();  // Creates BrowserStorage with IndexedDB

// ✅ Or create directly
const storage = await BrowserStorage.create();
```

**Browser-Only Architecture:**

This package is designed exclusively for browsers:

```typescript
// ✅ Available: Browser-compatible peer discovery
const staticRes = staticResolver(knownPeers);
const dhtRes = dhtResolver(['bootstrap1:443']);  // Uses fetch()

// ✅ Combined resolver works in browsers
const resolver = combinedResolver([
  staticResolver(knownPeers),
  dhtResolver(['bootstrap1:443']),
]);
```

**Why this is browser-only:**
- Uses IndexedDB for storage (not filesystem)
- WebRTC for P2P communication
- No Node.js-specific APIs (fs, http, net, dgram)
- Tests use polyfills to simulate browser environment

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

## Troubleshooting

### Common Issues

**"Cannot find module" errors:**
- Ensure all imports use `.js` extensions (ES modules requirement)
- Check that `type: "module"` is set in package.json

**WebRTC connection failures:**
- Verify signaling server is reachable
- Check firewall/NAT configuration for peer-to-peer connectivity
- Enable debug logging: `DEBUG=mau:* npm test`

**Test timeouts:**
- Increase Jest timeout: `jest.setTimeout(30000)`
- Check for unclosed connections (WebRTC, HTTP servers)
- Use `--detectOpenHandles` to find leaks

**Browser IndexedDB errors:**
- Clear browser storage: DevTools → Application → Clear Storage
- Check browser compatibility (IndexedDB API required)
- Verify HTTPS context (required for some browser features)

## Release Checklist

Before submitting PR:
- [ ] All tests pass (`npm test`)
- [ ] No linting errors (`npm run lint`)
- [ ] Code formatted (`npm run format`)
- [ ] Coverage maintained or improved (target: >50% branches)
- [ ] README.md updated with new features
- [ ] Examples work in both Node.js and browser
- [ ] TypeScript compiles without errors (`npm run build`)
- [ ] Bundle size is reasonable (`npm run build:browser`)
- [ ] API documentation updated (`npm run docs`)
- [ ] No console warnings in browser tests

## Security Considerations

### PGP Key Management
- Private keys are **always** encrypted at rest with a passphrase
- Never log or transmit unencrypted private keys
- Use secure random sources for key generation (`crypto.getRandomValues`)

### Network Security
- WebRTC uses PGP-based challenge-response authentication (not traditional X.509 mTLS)
- HTTP client does **not** yet implement certificate verification (use WebRTC for authenticated connections)
- All file contents are encrypted with OpenPGP before storage

### Input Validation
- Filename sanitization prevents directory traversal attacks
- PGP signature verification prevents tampered content
- Fingerprint validation ensures peer identity

## CI/CD Integration

GitHub Actions workflows handle:
- **Tests**: Automated test suite with coverage reporting
- **Linting**: ESLint checks on every push
- **Browser Tests**: Playwright-based browser automation
- **Documentation**: Auto-generated API docs with TypeDoc

**Local CI simulation:**
```bash
npm run lint && npm test && npm run build && npm run build:browser
```

## Resources

- **Mau Specification**: `../docs/` directory
- **Go Implementation**: `../` (reference implementation)
- **OpenPGP Spec**: RFC 4880
- **WebRTC Spec**: W3C WebRTC 1.0
- **Kademlia Paper**: Original DHT paper by Maymounkov & Mazières
- **TypeDoc**: [https://typedoc.org](https://typedoc.org)
- **Jest**: [https://jestjs.io](https://jestjs.io)
- **Vite**: [https://vitejs.dev](https://vitejs.dev)

## Getting Help

- **Issues**: Check existing issues on GitHub
- **Discussions**: Use GitHub Discussions for questions
- **Code Review**: Tag @emad-elsaid for review
- **Testing**: Run `npm test -- --verbose` for detailed output
- **Debug Mode**: Set `DEBUG=mau:*` environment variable

---

**Remember:** This implementation must work in both browser and Node.js. Test both environments before submitting changes.

## Quick Reference

| Task | Command |
|------|---------|
| Install dependencies | `npm install` |
| Run tests | `npm test` |
| Run tests (watch) | `npm test -- --watch` |
| Check coverage | `npm test -- --coverage` |
| Build for Node.js | `npm run build` |
| Build for browser | `npm run build:browser` |
| Start dev server | `npm run dev` |
| Lint code | `npm run lint` |
| Format code | `npm run format` |
| Generate docs | `npm run docs` |
| Serve docs | `npm run docs:serve` |
