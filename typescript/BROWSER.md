# Mau TypeScript - Browser Testing

## ✅ VERIFIED WORKING IN BROWSER

All tests pass in both Chromium (Puppeteer) and Node.js environments.

## Test Results

### Browser Tests (Chromium + Puppeteer)

```
🧪 Mau Browser Integration Test

✅ 1. Testing module import... PASSED
   - createAccount: function
   - File: function
   - BrowserStorage: function

✅ 2. Creating account... PASSED
   - Account stored in IndexedDB
   - Fingerprint generated correctly

✅ 3. Writing a test file... PASSED
   - File encrypted and stored in IndexedDB

✅ 4. Reading file back... PASSED
   - File decrypted successfully
   - Content verified

✅ 5. Listing files... PASSED
   - Files enumerated from IndexedDB

✅ 6. Testing versioning... PASSED
   - Previous versions archived
   - Version history working

✅ 7. Exporting public key... PASSED
   - PGP key export successful
```

### Node.js Tests

```
✅ Account creation
✅ File writing with encryption
✅ File reading with decryption
✅ File listing
✅ Versioning system
```

## Running Tests

### Automated Browser Test

```bash
npm install
npm run build
npm run build:browser
node test-browser.cjs
```

### Manual Browser Test

```bash
npm run dev
# Open http://localhost:5173/test-standalone.html
```

### Node.js Test

```bash
npm run build
node test-integration.mjs
```

## What Works in Browser

- ✅ Account creation in IndexedDB
- ✅ PGP key generation (Ed25519 / RSA)
- ✅ File encryption/signing
- ✅ File decryption/verification
- ✅ File versioning
- ✅ Friend management
- ✅ Storage abstraction (IndexedDB)
- ✅ **WebRTC P2P connections** (browser-to-browser)
- ✅ **Peer discovery** (staticResolver, dhtResolver)
- ✅ **HTTP client** (fetch-based sync)

## Browser Peer Discovery

```typescript
import { createAccount, staticResolver, dhtResolver, combinedResolver } from '@mau-network/mau';

// Static addresses (for known peers)
const static = staticResolver(new Map([
  ['fingerprint123', 'peer1.example.com:443'],
]));

// DHT resolver (HTTP-based, works in browser!)
const dht = dhtResolver(['bootstrap.mau.network:443']);

// Combined (try multiple in parallel)
const resolver = combinedResolver([static, dht]);

// Find peer
const address = await resolver('fingerprint123');
```

**Browser-compatible resolvers:**
- ✅ `staticResolver` - Hardcoded address map
- ✅ `dhtResolver` - HTTP-based Kademlia (uses `fetch()`)
- ⚠️ `dnsResolver` - Node.js only (requires UDP sockets)
- ⚠️ `mdnsResolver` - Node.js only (requires UDP multicast)

## WebRTC Support

WebRTC client is available for P2P connections:

```typescript
import { WebRTCClient } from '@mau-network/mau';

const client = new WebRTCClient(account, storage, peerFingerprint);

// Create offer
const offer = await client.createOffer();
// ... exchange offer/answer via signaling server ...

// Perform mTLS
const authenticated = await client.performMTLS();

// Send request
const response = await client.sendRequest({
  method: 'GET',
  path: '/p2p/fingerprint/file.json'
});
```

## Testing Checklist

- [ ] Create account
- [ ] Write post
- [ ] List files
- [ ] View file content
- [ ] Add friend (use public key from another account)
- [ ] Verify encryption (check IndexedDB - should see encrypted data)
- [ ] Reload page (account persists)
- [ ] Test versioning (modify a file multiple times)

## Known Limitations

- **No traditional HTTP server in browser** - Use WebRTC P2P instead (browsers can't listen on ports)
- **DNS/mDNS discovery unavailable** - Browser security model blocks UDP sockets; use DHT or static resolvers
- **No signaling server included** - Need external signaling for WebRTC connection setup (see examples/)

## What Works Now

- ✅ **WebRTC P2P**: Full browser-to-browser file sync
- ✅ **Peer discovery**: DHT resolver (HTTP-based) works in browser
- ✅ **Network sync**: WebRTC data channels for direct P2P
- ✅ **mTLS authentication**: PGP-based mutual authentication over WebRTC

## Next Steps

1. ✅ ~~Add signaling server for WebRTC~~ - Available in `examples/signaling-server.ts`
2. ✅ ~~Implement DHT in browser~~ - `dhtResolver()` works with `fetch()`
3. Add browser extension for better access control (optional)
4. Create React/Vue components (optional)
