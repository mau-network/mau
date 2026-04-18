# Missing Features in TypeScript Implementation

## 1. mDNS Local Network Discovery

**Spec Reference:** Section "Local Network Discovery (mDNS-SD)"

**Status:** Not implemented

**Details:**
- Spec describes mDNS-SD for automatic peer discovery on LANs
- Service name: `_mau._tcp.local`
- Should announce fingerprint, IP, and port
- Currently only DHT resolver exists in `resolvers.ts`

**Impact:** Cannot discover peers on local networks without manual configuration

---

## 2. Group/Selective Recipient Encryption

**Spec Reference:** Step 8 "Share with Multiple Recipients (Group)"

**Status:** Partially implemented - always encrypts to ALL friends

**Details:**
- `File.write()` always uses `this.account.getAllPublicKeys()` (line 143 in file.ts)
- Spec shows encrypting to specific recipients: `gpg --encrypt -r alice -r bob`
- Need ability to pass custom recipient list for group messages/DMs

**Impact:** Cannot create private messages or group-specific content - everything is visible to all friends

**Proposed API:**
```typescript
await file.writeJSON(data, { recipients: [aliceFingerprint, bobFingerprint] });
```
