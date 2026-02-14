# AGENTS.md - AI Assistant Guidelines for Mau

This file provides context and guidelines for AI assistants (LLMs) working on the Mau codebase.

## Project Overview

**Mau** is a convention for building peer-to-peer (P2P) Web 2.0 applications. It provides:
- Decentralized file sharing with PGP encryption
- Friend-based authentication and authorization
- DHT-based peer discovery (Kademlia)
- TLS 1.3 for secure communication
- HTTP server with mTLS client certificate verification

## Architecture

### Core Components

1. **Account** (`account.go`)
   - PGP identity management using `golang.org/x/crypto/openpgp`
   - File encryption/decryption with friend-based recipients
   - Certificate generation for TLS

2. **Server** (`server.go`)
   - HTTPS server with mTLS
   - File serving with HTTP Range support (resumable downloads)
   - Friend file listing endpoint

3. **Client** (`client.go`)
   - Friend discovery and file synchronization
   - Multi-resolver support (local mDNS, DHT, static addresses)

4. **DHT/Kademlia** (`kademlia.go`)
   - Distributed peer discovery
   - K-bucket routing table
   - Peer announcement and lookup

5. **Friends** (`friend.go`)
   - PGP-based friend management
   - Follow/unfollow mechanism
   - File access control

## Development Guidelines

### Code Quality Standards

1. **Error Handling**
   - Every error must be checked
   - Use `assert.NoError(t, err)` in tests (testify/assert)
   - Never ignore errors with `_` unless explicitly justified

2. **Testing**
   - Use `github.com/stretchr/testify/assert` for all assertions
   - Avoid manual `if err != nil { t.Fatalf(...) }` - use `assert.NoError`
   - Test files should follow the same error handling patterns as production code

3. **Linting**
   - All code must pass `golangci-lint` with zero errors
   - Do not disable linters; fix the underlying issues
   - Run locally before pushing: `golangci-lint run --timeout=5m`

4. **Dependencies**
   - Go 1.26 or higher
   - Uses deprecated `golang.org/x/crypto/openpgp` (project dependency, properly marked with nolint directives)

### Common Patterns

#### Creating a New Account
```go
account, err := NewAccount(dir, "Name", "email@example.com", "password")
assert.NoError(t, err)
```

#### Adding and Following a Friend
```go
var friendPub bytes.Buffer
err := friendAccount.Export(&friendPub)
assert.NoError(t, err)

friend, err := account.AddFriend(&friendPub)
assert.NoError(t, err)

err = account.Follow(friend)
assert.NoError(t, err)
```

#### File Operations
```go
// Adding a file with recipients
file, err := account.AddFile(reader, "filename.txt", []*Friend{friend})
assert.NoError(t, err)

// Private file (no recipients)
file, err := account.AddFile(reader, "private.txt", []*Friend{})
assert.NoError(t, err)
```

## CI/CD

The project uses GitHub Actions:
- **Tests** (`.github/workflows/test.yml`): Runs test suite with coverage
- **Lint** (`.github/workflows/lint.yml`): Runs golangci-lint (installed from source for Go 1.26 compatibility)
- **Build** (`.github/workflows/xlog.yml`): Builds documentation

## Testing Commands

```bash
# Run all tests
go test ./...

# Run specific test
go test -v -run TestName

# Run linter
golangci-lint run --timeout=5m

# Run tests with coverage
go test -coverprofile=coverage.out ./...
```

## File Structure

```
mau/
├── account.go           # Account and PGP identity management
├── client.go            # Client for downloading friend files
├── server.go            # HTTPS server with mTLS
├── kademlia.go          # DHT implementation
├── friend.go            # Friend management
├── file.go              # File operations
├── fingerprint.go       # PGP fingerprint handling
├── resolvers.go         # Peer discovery resolvers
├── cmd/mau/mau.go       # CLI application
├── *_test.go            # Test files
└── .github/workflows/   # CI/CD workflows
```

## Key Concepts

### Friend-Based Security Model
- Files can be encrypted for specific friends (recipients)
- Only friends with the correct PGP key can decrypt files
- mTLS ensures peer identity verification

### Peer Discovery
- **Local**: mDNS for LAN discovery
- **Internet**: DHT (Kademlia) for global peer lookup
- **Static**: Direct IP:port addressing

### File Versioning
Files support versioning in `.versions/` subdirectories, allowing rollback and history.

## When Working on Mau

1. **Understand the security model**: Friend-based encryption and mTLS
2. **Check errors religiously**: Never ignore error returns
3. **Test thoroughly**: Add tests for new functionality
4. **Run linter locally**: Before pushing, ensure `golangci-lint` passes
5. **Maintain consistency**: Follow existing patterns (especially in tests)

## References

- [Kademlia DHT Paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)
- [OpenPGP RFC 4880](https://tools.ietf.org/html/rfc4880)
- [HTTP Range RFC 7233](https://tools.ietf.org/html/rfc7233)

---

This file is for AI assistants. For human contributors, see `README.md` and `TODO.md`.
