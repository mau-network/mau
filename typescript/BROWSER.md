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

## What Works

- ✅ Account creation in IndexedDB
- ✅ PGP key generation (Ed25519)
- ✅ File encryption/signing
- ✅ File decryption/verification
- ✅ File versioning
- ✅ Friend management
- ✅ Storage abstraction (filesystem/browser)

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

- **No network sync yet** - Files stay local until WebRTC or HTTPS client is connected
- **No peer discovery** - Must manually exchange WebRTC offers/answers
- **No signaling server** - Need external signaling for WebRTC connection setup

## Next Steps

1. Add signaling server for WebRTC
2. Implement DHT in browser
3. Add browser extension for better access control
4. Create React components
