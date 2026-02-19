# Mau E2E Testing Framework

> **End-to-end testing infrastructure for the Mau P2P file synchronization system**

## Quick Start

### Prerequisites

- Docker 20.10+ and Docker Compose
- Go 1.21+
- Make

### Run Tests Locally

```bash
# Build the Mau E2E Docker image
make -C e2e build-image

# Run basic tests (2-5 peers)
make -C e2e test-basic

# Run all tests (includes chaos and stress tests)
make -C e2e test-all
```

### Run Manual Test Environment

```bash
# Start 3 Mau peers with Docker Compose
cd e2e
docker-compose up

# In another terminal, interact with peers
curl http://localhost:8081/p2p/<fingerprint>
```

## Project Status

**Current Phase:** Design & Planning ✅  
**Next Phase:** Foundation Implementation (Weeks 1-2)

## Documentation

- **[📋 PLAN.md](PLAN.md)** - Comprehensive framework design (read this first!)
- **[📝 Writing Tests Guide](docs/writing-tests.md)** - How to add new test cases *(coming soon)*
- **[🐛 Debugging Guide](docs/debugging.md)** - Troubleshooting failed tests *(coming soon)*
- **[🏗️ Architecture](docs/architecture.md)** - Framework internals *(coming soon)*

## Test Scenarios

### Level 1: Basic Functionality
- ✅ **TC-001:** Two-peer discovery via Kademlia DHT
- ✅ **TC-002:** Two-peer friend sync
- ✅ **TC-003:** Multi-peer sync (5 peers)
- ✅ **TC-004:** Version conflict resolution

### Level 2: Resilience Testing
- ✅ **TC-101:** Peer crash during sync
- ✅ **TC-102:** Network partition (split brain)
- ✅ **TC-103:** High latency network (500ms)
- ✅ **TC-104:** Bandwidth limitation (10 KB/s)
- ✅ **TC-105:** Packet loss (10%)

### Level 3: Stress Testing
- ✅ **TC-201:** 10-peer full mesh
- ✅ **TC-202:** 100-peer network (sparse graph)
- ✅ **TC-203:** Peer churn (join/leave)

### Level 4: Security Testing
- ✅ **TC-301:** Unauthorized file access
- ✅ **TC-302:** DHT Sybil attack resistance *(future)*

**Status Legend:**  
✅ Designed | 🚧 In Progress | ✔️ Implemented

## Architecture Overview

```
Test Coordinator (Go + Testcontainers)
    │
    ├── Mau Peer Containers (Docker)
    │   ├── Account + PGP Keyring
    │   ├── HTTP Server
    │   ├── DHT Node
    │   └── File Storage
    │
    ├── Toxiproxy (Network Simulation)
    │   ├── Latency Injection
    │   ├── Bandwidth Limiting
    │   └── Network Partitions
    │
    └── Observability
        ├── Structured JSON Logs
        ├── State Snapshots
        └── Test Result Artifacts
```

## Technology Stack

| Component | Technology | Why? |
|-----------|-----------|------|
| **Orchestration** | Testcontainers-Go | Programmatic container control, automatic cleanup |
| **Network Simulation** | Toxiproxy | Dynamic failure injection without container restarts |
| **Test Framework** | Go testing + Testify | Minimal dependencies, familiar to contributors |
| **Logging** | Structured JSON | Easy parsing, CI-friendly |

## Contributing

### Adding a New Test

1. Choose the appropriate level: `basic/`, `resilience/`, `stress/`, or `security/`
2. Create test file: `scenarios/<level>/<test_name>_test.go`
3. Follow the example test structure in [PLAN.md](PLAN.md#example-test-case-walkthrough)
4. Run locally to validate
5. Submit PR

**Example:**

```go
func TestMyScenario(t *testing.T) {
    env := testenv.NewTestEnv(t)
    defer env.Cleanup()
    
    peers := env.AddPeers(3)
    env.MakeFriends(peers[0], peers[1])
    
    // Your test logic here
    
    assertions.AssertFilesSynced(t, peers, "test.txt", 30*time.Second)
}
```

See [PLAN.md § Example Test Case Walkthrough](PLAN.md#example-test-case-walkthrough) for detailed walkthrough.

## CI/CD Integration

Tests run automatically on:
- **Every PR** (basic + resilience tests)
- **Main branch push** (all tests)
- **Nightly** (stress tests with 100+ peers)

**GitHub Actions Workflow:** [`.github/workflows/e2e-tests.yml`](../.github/workflows/e2e-tests.yml)

## Debugging Failed Tests

When a test fails in CI:

1. Download artifacts: `test-results-<level>.zip`
2. Extract and check `summary.json` for failure reason
3. Read peer logs with `jq`:
   ```bash
   jq '.component == "sync"' peer-a.json
   ```
4. Inspect file state snapshots in `snapshots/`
5. Reproduce locally with Docker Compose

Full guide: [docs/debugging.md](docs/debugging.md) *(coming soon)*

## Performance Benchmarks

*To be established during Phase 4 implementation*

| Scenario | Target | Current | Status |
|----------|--------|---------|--------|
| 2-peer sync (1MB) | <5s | TBD | 🚧 |
| 10-peer mesh sync | <2min | TBD | 🚧 |
| 100-peer DHT lookup | <1s avg | TBD | 🚧 |

## Roadmap

- [x] **Phase 0:** Design & Planning (Week 0) ← **YOU ARE HERE**
- [ ] **Phase 1:** Foundation (Weeks 1-2)
- [ ] **Phase 2:** Multi-Peer & Assertions (Weeks 3-4)
- [ ] **Phase 3:** Chaos Engineering (Weeks 5-6)
- [ ] **Phase 4:** Stress Testing (Weeks 7-8)
- [ ] **Phase 5:** Observability & Debugging (Weeks 9-10)
- [ ] **Phase 6:** CI/CD Integration & Security (Weeks 11-12)

See [PLAN.md § Implementation Phases](PLAN.md#implementation-phases) for details.

## Questions or Issues?

- **Design Questions:** Review [PLAN.md § Open Questions](PLAN.md#open-questions--future-work)
- **Implementation Issues:** Open a GitHub issue with `e2e` label
- **Framework Bugs:** Include test logs and snapshots

## License

This testing framework inherits Mau's GPLv3 license.

---

**Maintained by:** Mau Contributors  
**Last Updated:** 2026-02-19  
**Framework Version:** 1.0-design
