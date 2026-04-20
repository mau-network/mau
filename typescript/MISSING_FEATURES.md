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

**Spec Reference:** README.md - "A Message can be encrypted to one person for a private end-to-end chat or multiple recipients for a group chat."

**Status:** ✅ FIXED in PR #93

**Details:**
- `File.write()` previously always used `this.account.getAllPublicKeys()` (line 143 in file.ts)
- Now supports optional `recipients` parameter for selective encryption
- Added `Account.getPublicKeys(fingerprints)` method
- 7 new tests covering DM and group message scenarios

**Fixed:** Can now create private messages and group-specific content via:
```typescript
await file.writeJSON(data, { recipients: [aliceFingerprint, bobFingerprint] });
```
