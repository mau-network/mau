# Mau TypeScript Implementation - Spec Compliance Issues

## Missing Features

### 1. MDNS Service Discovery (§ Peer-to-Peer Stack)
**Spec Requirement:** "If local network is desired the service may announce itself on the network using MDNS multicast."

**Status:** Not implemented

**Impact:** Local network peer discovery unavailable. Users cannot discover peers on the same LAN without manual address configuration.

**Spec Format:**
```
5D000B2F2C040A1675B49D7F0C7CB7DC36999D56._mau._tcp.local.
|------------ Fingerprint --------------|app-|protocol|domain|
```

### 2. NAT Traversal (§ Listening on internet requests)
**Spec Requirement:** "The program is responsible for allowing the user to receive connections from outside of the local network by utilizing NAT traversal protocols such as UPNP, NAT-PMP, or Hole punching."

**Status:** Not implemented

**Impact:** Users behind NAT cannot receive inbound connections. While DHT relay signaling helps establish WebRTC connections, there's no automatic port forwarding via UPNP/NAT-PMP for the HTTP server.

### 3. Friend Keys Encryption (§ Why are friends' keys encrypted?)
**Spec Requirement:** "All friends' public keys should be encrypted with the account key"

**Rationale (from spec):** "To make sure the friend public key is added by the account instead of a malicious program. If the public key is written in a plain format it means adding a friend is not an authenticated operation any program can do it without the user's permission"

**Status:** **BUG** - Keys saved in binary format but NOT encrypted

**Location:** `src/account.ts:187-189`
```typescript
const friendKeyPath = this.storage.join(this.getMauDir(), `${fingerprint}.pgp`);
const binaryKey = publicKey.write();
await this.storage.writeFile(friendKeyPath, binaryKey); // Should encrypt with account key
```

**Impact:** Friend keys can be tampered with by malicious programs. This defeats the authentication mechanism for the contact list.

**Fix Required:** Encrypt `binaryKey` with account's public key before writing to disk.

## Implementation Quality

All core features are implemented:
- ✅ PGP identity management
- ✅ File encryption/signing
- ✅ Versioning
- ✅ HTTP server interface (file list, file download, Range support)
- ✅ Kademlia DHT over WebRTC with relay signaling
- ✅ Client sync with retry and backoff
- ✅ mTLS challenge-response authentication

## Summary

**Critical Bug:** Friend keys not encrypted (security issue)  
**Missing Features:** MDNS discovery (2 implementations), NAT traversal (requires native libs)  

The TypeScript implementation is otherwise solid and follows the spec closely.
