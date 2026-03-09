# Mau TypeScript - Browser Testing

## Quick Start

1. **Install dependencies:**
   ```bash
   npm install
   ```

2. **Build for browser:**
   ```bash
   npm run build:browser
   ```

3. **Start dev server:**
   ```bash
   npm run dev
   ```

4. **Open browser:**
   Navigate to `http://localhost:5173/demo.html`

## What Works

- ✅ Account creation in IndexedDB
- ✅ Post writing with encryption
- ✅ File listing
- ✅ Friend management
- ✅ Content viewing

## Browser Storage

All data is stored in IndexedDB under the database name `mau-storage`.

You can inspect it in Chrome DevTools → Application → IndexedDB.

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
