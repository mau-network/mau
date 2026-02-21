# Investigation: Can Mau Account Identity Be Modified?

## Question
Does the Mau package API support changing account name/email after creation?

## Answer: NO

## Evidence

### 1. Account API (account.go)
**Read-only methods ONLY:**
- `Account.Name() string` - getter only
- `Account.Email() string` - getter only
- `Account.Identity() (string, error)` - getter only
- `Account.Fingerprint() Fingerprint` - getter only

**No setter methods exist:**
- ❌ No `SetName()`
- ❌ No `SetEmail()`
- ❌ No `UpdateIdentity()`
- ❌ No `ChangeIdentity()`

### 2. Entity Structure (golang.org/x/crypto/openpgp)
```go
type Entity struct {
    PrimaryKey  *packet.PublicKey
    PrivateKey  *packet.PrivateKey
    Identities  map[string]*Identity  // Map of identities
    Revocations []*packet.Signature
    Subkeys     []Subkey
}

type Identity struct {
    Name          string              // "Full Name (comment) <email>"
    UserId        *packet.UserId      // Contains name and email
    SelfSignature *packet.Signature   // Cryptographic signature
    Signatures    []*packet.Signature
}
```

**The Identity is cryptographically signed:**
- `SelfSignature` binds the UserId to the PrimaryKey
- Modifying name/email would invalidate the signature
- Invalid signatures = broken PGP verification

### 3. Account Creation (NewAccount)
```go
func NewAccount(root, name, email, passphrase string) (*Account, error) {
    entity, err := openpgp.NewEntity(name, "", email, &packet.Config{
        DefaultHash:            crypto.SHA256,
        DefaultCompressionAlgo: packet.CompressionZIP,
        RSABits:                rsaKeyLength,
    })
    // ... save encrypted entity ...
}
```

Identity is set **once** at creation time and saved to encrypted file.

### 4. Account Loading (OpenAccount)
```go
func OpenAccount(rootPath, passphrase string) (*Account, error) {
    // ... decrypt and read entity from file ...
    entityList, err := openpgp.ReadKeyRing(reader)
    // Returns the entity exactly as saved
}
```

Loads the entity from disk - no modification happens.

## Why Identity Cannot Be Changed

### Cryptographic Reasons:
1. **Self-Signature**: Identity is signed with the private key
2. **Fingerprint**: Derived from the public key, not the identity
3. **Trust Chain**: Friends verify identity via signature
4. **Message Encryption**: Uses fingerprint, not name/email

### Technical Reasons:
1. **No API**: Neither Mau nor openpgp provide identity modification methods
2. **Immutable Design**: PGP entities are designed to be permanent
3. **File Storage**: Entity is serialized/encrypted as-is

### Theoretical Workarounds (NOT RECOMMENDED):

#### Option A: Add New Identity to Existing Key
```go
// This would require:
entity.Identities["new identity"] = &Identity{
    Name:   "New Name <new@email.com>",
    UserId: packet.NewUserId("New Name", "", "new@email.com"),
}
// Then self-sign it and save
```
**Problems:**
- Multiple identities confuse which one is "primary"
- Would need to modify openpgp package usage
- Not supported by Mau's Account structure (returns first identity only)

#### Option B: Create New Account and Migrate
```go
// 1. Create new account with new name/email
newAcc := NewAccount(root, "New Name", "new@email.com", passphrase)

// 2. Export old account's friends
oldFriends := oldAcc.ListFriends()

// 3. Import friends to new account
for _, friend := range oldFriends {
    newAcc.ImportFriend(friend)
}

// 4. Migrate posts (re-encrypt with new key)
// 5. Notify all friends of fingerprint change
```
**Problems:**
- New fingerprint = loses all existing connections
- Friends must manually re-add you
- All old messages can't be decrypted with new key
- Massive disruption to the network

## Conclusion

**The Mau package intentionally does NOT support changing name/email** because:
1. PGP identities are designed to be permanent
2. Cryptographic signatures bind identity to key
3. Changing identity would break the trust model
4. No safe way to propagate changes to all friends

**The GUI fix (making fields read-only) is CORRECT.**

## Recommendation

If identity change is truly needed, implement **Option B** as a dedicated feature:
- "Migrate to New Account" wizard
- Clear warnings about consequences
- Automated friend re-addition process
- Export/import posts (as plaintext, re-encrypt)
- This would be a major feature (50-100 hours work)

For now, accept that PGP identity is permanent by design.
