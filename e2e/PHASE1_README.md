# Mau E2E Framework - Phase 1 Implementation

**Status:** Phase 1 Foundation Complete ✓  
**Date:** 2026-02-19  

## What's Implemented

### 1. Docker Infrastructure

- **Dockerfile** (`e2e/docker/Dockerfile`)
  - Multi-stage build (builder + runtime)
  - Based on Mau's existing build process
  - Exposes HTTP (8080) and P2P (9090) ports
  - Automatic Go 1.26 toolchain download
  - Configurable via environment variables

- **Entrypoint Script** (`e2e/docker/entrypoint.sh`)
  - Automatic account initialization
  - Non-interactive passphrase handling
  - Serves Mau on startup

### 2. CLI Tool (`mau-e2e`)

Basic command structure with 4 core commands:

- `mau-e2e start [--peers N]` - Start test environment with N peers
- `mau-e2e stop` - Stop and cleanup environment
- `mau-e2e status` - Show running peers and environment info
- `mau-e2e logs <peer>` - View peer container logs

**State Management:**
- Environment state saved to `~/.mau-e2e/current-env.json`
- Persists across command invocations
- Cleanup on environment stop

### 3. Shared testenv Library

Core library (`e2e/internal/testenv/`) used by both CLI and automated tests:

- `Environment` - Manages Docker network and peers
- `Peer` - Represents a Mau peer container
- State persistence and loading
- Testcontainers-Go integration

### 4. Integration Tests

Basic tests (`e2e/tests/basic_test.go`):

- `TestSinglePeerHealthCheck` - Verify single peer starts successfully
- `TestTwoPeerDiscovery` - Verify two peers can be created on same network

**Test Infrastructure:**
- Uses `testing` package + Testify assertions
- Automatic cleanup with defer
- Respects `-short` flag

### 5. Makefile Targets

```bash
make docker-build  # Build Mau E2E Docker image
make cli           # Build mau-e2e CLI binary
make test          # Run E2E tests
make clean         # Cleanup Docker resources
```

## Quick Start

### Build Everything

```bash
cd e2e

# Build Docker image
make docker-build

# Build CLI
make cli
```

### Interactive Mode

```bash
# Start environment with 2 peers
./bin/mau-e2e start --peers 2

# Check status
./bin/mau-e2e status

# View logs
./bin/mau-e2e logs peer-0

# Stop environment
./bin/mau-e2e stop
```

### Automated Testing

```bash
# Run all tests
make test

# Or run directly
go test -v ./tests/...
```

## File Structure

```
e2e/
├── Makefile                       # Build automation
├── go.mod                         # E2E-specific Go module
├── go.sum                         # Go dependencies
│
├── docker/                        # Docker configuration
│   ├── Dockerfile                 # Mau test image
│   └── entrypoint.sh              # Container startup script
│
├── cmd/
│   └── mau-e2e/                   # CLI tool
│       ├── main.go                # Entry point
│       └── commands/              # Subcommands
│           ├── start.go           # Start environment
│           ├── stop.go            # Stop environment
│           ├── status.go          # Show status
│           └── logs.go            # View logs
│
├── internal/
│   └── testenv/                   # Shared test environment library
│       └── environment.go         # Environment management
│
└── tests/                         # Integration tests
    └── basic_test.go              # Basic functionality tests
```

## Known Limitations (Phase 1)

These are intentional - to be addressed in later phases:

1. **No DHT Discovery Testing** - Peers start but don't verify DHT connectivity yet
2. **No Friend Relationships** - CLI commands not yet implemented
3. **No File Operations** - File add/sync commands coming in Phase 2
4. **No Toxiproxy** - Network simulation coming in Phase 3
5. **Fingerprint Extraction** - Currently returns placeholder value
6. **Logs Command** - Placeholder implementation (suggests using `docker logs`)

## Phase 1 Deliverables Checklist

✅ **Dockerfile for Mau Test Image**
   - Multi-stage build
   - Environment variable configuration
   - Working entrypoint

✅ **Basic CLI Tool (`mau-e2e`)**
   - Cobra framework
   - `start`, `stop`, `status`, `logs` commands
   - Testcontainers-Go integration
   - State persistence in ~/.mau-e2e/

✅ **First Integration Test**
   - Single peer health check
   - Two peers basic setup
   - Automatic cleanup

✅ **Makefile Integration**
   - `make docker-build` - Build test image
   - `make cli` - Build CLI tool
   - `make test` - Run tests
   - `make clean` - Cleanup

## Testing Phase 1

### Docker Image Test

```bash
# Build image
make docker-build

# Run container manually
docker run -it --rm \
  -e MAU_PEER_NAME=test-peer \
  -e MAU_PASSPHRASE=testpass \
  -p 8080:8080 \
  mau-e2e:latest

# Should see:
# [entrypoint] Initializing new account...
# [entrypoint] Starting Mau server...
# Account: test-peer <fingerprint>
```

### CLI Test

```bash
# Build CLI
make cli

# Start environment
./bin/mau-e2e start --peers 2

# Expected output:
# Starting Mau E2E environment with 2 peers...
# Starting peer peer-0...
#   ✓ peer-0 started (HTTP: <port>, container: <id>)
# Starting peer peer-1...
#   ✓ peer-1 started (HTTP: <port>, container: <id>)
# 
# ✓ Environment 'mau-test-<timestamp>' started successfully!

# Check status
./bin/mau-e2e status

# Expected: Table showing 2 running peers

# Cleanup
./bin/mau-e2e stop --force
```

### Automated Test

```bash
# Run tests
make test

# Expected:
# === RUN   TestSinglePeerHealthCheck
# --- PASS: TestSinglePeerHealthCheck (15.23s)
# === RUN   TestTwoPeerDiscovery
# --- PASS: TestTwoPeerDiscovery (18.45s)
# PASS
```

## Troubleshooting

### Docker Build Fails

**Problem:** `go: requires go >= 1.26`

**Solution:** Ensure `GOTOOLCHAIN=auto` is set in Dockerfile (already included)

### Container Won't Start

**Problem:** Entrypoint fails

**Solution:**
```bash
# Check container logs
docker logs <container-id>

# Common issues:
# - Passphrase prompt hanging: Fixed by printf in entrypoint
# - Account init fails: Check /data/.mau directory permissions
```

### Tests Timeout

**Problem:** Containers don't become "ready"

**Solution:**
```bash
# Check wait strategy in environment.go
# Current: wait.ForLog("Account:")
# Adjust timeout if needed (default: 30s)
```

### State File Corruption

**Problem:** `mau-e2e status` shows wrong info

**Solution:**
```bash
# Delete state and start fresh
rm ~/.mau-e2e/current-env.json
./bin/mau-e2e start
```

## Next Steps: Phase 2

**Focus:** Multi-peer interactions + file/friend CLI commands

**Planned:**
1. Implement fingerprint extraction from containers
2. Add `mau-e2e friend add/list` commands
3. Add `mau-e2e file add/list/cat` commands
4. Implement `peer inspect` command
5. Add real DHT discovery assertions to tests
6. Test 2-peer file synchronization end-to-end

**Timeline:** Week 3-4

## Success Criteria Met

✅ Can build Mau E2E Docker image  
✅ Can start N containers with one command  
✅ CLI persists state across commands  
✅ Tests run and pass (basic infrastructure validation)  
✅ Automatic cleanup works  
✅ Makefile targets all functional  

**Phase 1 is foundation-complete and ready for Phase 2 work!**

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-19  
**Implementation Status:** Complete
