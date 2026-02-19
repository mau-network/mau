# Mau E2E Testing Framework - Comprehensive Plan

**Version:** 1.0  
**Date:** 2026-02-19  
**Status:** Design Phase  

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Research Findings](#research-findings)
3. [Architecture Overview](#architecture-overview)
4. [Technology Stack](#technology-stack)
5. [Test Scenarios](#test-scenarios)
6. [File Structure](#file-structure)
7. [Implementation Phases](#implementation-phases)
8. [Example Test Case Walkthrough](#example-test-case-walkthrough)
9. [Framework Comparison](#framework-comparison)
10. [CI/CD Integration](#cicd-integration)
11. [Debugging & Observability](#debugging--observability)
12. [Open Questions & Future Work](#open-questions--future-work)

---

## Executive Summary

This document outlines a comprehensive end-to-end (E2E) testing framework for **Mau**, a P2P file synchronization system built on Kademlia DHT. The framework provides **two complementary modes**:

1. **Interactive CLI Mode** (`mau-e2e` tool) - Manual control and exploratory testing
2. **Automated Testing Mode** (`go test`) - CI/CD integration and regression detection

Both modes share the same core `testenv` library, ensuring consistency between manual exploration and automated validation.

**Key Design Principles:**
- **Interactive-first design** - Developers can see P2P behavior, not just assert it worked
- **Deterministic by default, chaos-ready by design**
- **Easy to add new test cases** with minimal boilerplate
- **Rich observability** with comprehensive logging, tracing, and state inspection
- **Network condition simulation** (latency, packet loss, partitions)
- **CI/CD friendly** with parallelizable, isolated test execution
- **Production-grade reliability** for long-term maintenance

**Primary Goals:**
1. **[Interactive]** Enable manual exploration of P2P synchronization behavior
2. **[Automated]** Test N Mau instances discovering each other via Kademlia DHT
3. **[Automated]** Verify friend relationship establishment and maintenance
4. **[Automated]** Validate file synchronization across peers
5. **[Both]** Simulate network failures and verify recovery
6. **[Automated]** Stress test with varying peer counts (2-100+)
7. **[Automated]** Detect regressions before they reach production

> **See Also:** [CLI_DESIGN.md](CLI_DESIGN.md) for detailed interactive CLI specifications

---

## Research Findings

### Analysis of Existing P2P Testing Frameworks

#### 1. **libp2p/test-plans** 
**What it does:** Interoperability testing for libp2p implementations  
**Approach:** 
- Docker containers with different libp2p implementations
- Language-agnostic test orchestration
- Test scenarios defined in separate containers
- Results published to GitHub Pages

**Key Learnings:**
- ✅ **Language-agnostic orchestration** allows testing across implementations
- ✅ **Docker-based isolation** ensures reproducibility
- ✅ **Separate test definition from implementation** (test scenarios as standalone containers)
- ⚠️ Complex setup for simple scenarios

**Applicable to Mau:**
- Use Docker containers for peer isolation
- Define test scenarios declaratively
- Collect structured test results

---

#### 2. **Ethereum Hive**
**What it does:** End-to-end test harness for Ethereum clients  
**Approach:**
- Simulator framework in Go
- Client implementations run in Docker containers
- Simulators orchestrate multi-client scenarios
- JSON-based test result reporting

**Key Learnings:**
- ✅ **Simulator pattern** separates test logic from client runtime
- ✅ **Client-agnostic interface** allows testing different implementations
- ✅ **Structured result format** (JSON) enables trend analysis
- ✅ **Trophy list** motivates participation and showcases bugs found

**Applicable to Mau:**
- Adopt simulator pattern for orchestration
- Use Go for test harness (matches Mau's language)
- Implement structured test results with detailed traces
- Create "bug trophy list" to validate framework effectiveness

---

#### 3. **Testcontainers-Go**
**What it does:** Programmatic container lifecycle management for tests  
**Approach:**
- Go API for creating/managing Docker containers in tests
- Lifecycle hooks (startup, ready checks, teardown)
- Network management, volume mounts, log streaming
- Automatic cleanup with `defer`

**Key Learnings:**
- ✅ **Native Go integration** - no external orchestration needed
- ✅ **Type-safe API** prevents configuration errors
- ✅ **Automatic cleanup** reduces test pollution
- ✅ **Wait strategies** ensure containers are ready before testing
- ✅ **Log capture** for debugging failures
- ⚠️ Requires Docker daemon access

**Applicable to Mau:**
- **Primary orchestration tool** for E2E tests
- Use `GenericContainer` with custom Mau image
- Implement custom wait strategies for Kademlia bootstrap
- Capture logs per-container for debugging

---

#### 4. **Toxiproxy**
**What it does:** Network condition simulation proxy  
**Approach:**
- TCP proxy with HTTP API for adding "toxics"
- Toxics: latency, bandwidth limits, timeouts, connection resets
- Upstream/downstream control (client→server vs server→client)
- Probability-based application (toxicity parameter)

**Supported Toxics:**
- `latency`: Add delay ± jitter
- `bandwidth`: Rate limiting (KB/s)
- `timeout`: Drop data, optionally close connection
- `slow_close`: Delay TCP FIN
- `reset_peer`: TCP RST simulation
- `slicer`: Fragment packets
- `limit_data`: Close after N bytes

**Key Learnings:**
- ✅ **Surgical network failure injection** without container restarts
- ✅ **Directional control** (affect only requests or responses)
- ✅ **Dynamic reconfiguration** during test execution
- ✅ **Lightweight** - runs alongside services
- ⚠️ HTTP API adds complexity but enables dynamic control

**Applicable to Mau:**
- Run Toxiproxy sidecars in Docker network
- Configure each Mau peer to route through proxy
- Inject latency/partitions during synchronization tests
- Test Kademlia resilience under packet loss

---

#### 5. **Chaos Engineering Principles**
**Source:** principlesofchaos.org, Netflix Simian Army  

**Core Principles:**
1. **Define steady state** - measurable system output indicating normal behavior
2. **Hypothesize steady state continues** under perturbation
3. **Introduce real-world failure variables** (crashes, network issues, resource exhaustion)
4. **Disprove hypothesis** by detecting steady state deviation

**Advanced Principles:**
- Focus on **system behavior, not internals**
- Vary **real-world events** by impact/frequency
- **Run in production** when possible (not applicable to Mau tests)
- **Automate continuously**
- **Minimize blast radius**

**Key Learnings:**
- ✅ Define "sync success" steady state for Mau
- ✅ Test assumptions about resilience explicitly
- ✅ Automate chaos scenarios in CI
- ✅ Start with small perturbations, increase gradually

**Applicable to Mau:**
- Define steady state: "All peers have consistent file states"
- Hypothesis: "File sync completes within 5s under normal conditions"
- Variables: peer crashes, network partitions, high latency
- Validation: Check file SHA256 consistency across peers

---

### Technology Stack Evaluation

| Component | Options Considered | Selected | Rationale |
|-----------|-------------------|----------|-----------|
| **Container Orchestration** | Docker Compose, Testcontainers-Go, Kubernetes | **Testcontainers-Go** | Native Go integration, programmatic control, automatic cleanup |
| **Mau Instance Packaging** | Binary in container, Docker image | **Docker image** | Consistent environment, easy version management |
| **Network Simulation** | Toxiproxy, tc (Linux), Pumba, Comcast | **Toxiproxy** | Cross-platform, programmable, well-documented |
| **Test Framework** | Go testing, Ginkgo/Gomega, Testify | **Go testing + Testify** | Minimal dependencies, familiar to Mau contributors |
| **Logging** | Container logs, Loki, ELK | **Structured JSON logs + file export** | Simple, parseable, CI-friendly |
| **Tracing** | OpenTelemetry, Jaeger, custom | **Custom trace IDs in logs** | Lightweight, no external dependencies |
| **Metrics** | Prometheus, InfluxDB, none | **Test execution metrics in JSON** | Easy CI integration, no infra overhead |
| **Assertion Library** | Standard Go, Testify, Gomega | **Testify** | Rich assertions, already used in Mau codebase |

---

## Architecture Overview

### Dual-Mode Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                     Mau E2E Framework                          │
│                                                                │
│  ┌──────────────────────┐    ┌──────────────────────┐         │
│  │  Interactive Mode    │    │   Automated Mode     │         │
│  │  (mau-e2e CLI)       │    │   (go test)          │         │
│  │                      │    │                      │         │
│  │  - Manual control    │    │  - CI/CD testing     │         │
│  │  - Exploration       │    │  - Regression detect │         │
│  │  - Debugging         │    │  - Assertions        │         │
│  │  - Demonstrations    │    │  - Coverage tracking │         │
│  └──────────┬───────────┘    └──────────┬───────────┘         │
│             │                           │                     │
│             └────────┬──────────────────┘                     │
│                      ▼                                        │
│           ┌──────────────────────┐                           │
│           │   Shared Core        │                           │
│           │   (testenv library)  │                           │
│           │                      │                           │
│           │  - Peer management   │                           │
│           │  - Network control   │                           │
│           │  - State persistence │                           │
│           │  - Assertions        │                           │
│           └──────────┬───────────┘                           │
└──────────────────────┼───────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Docker Network (mau-test-net)               │
│                                                                 │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐    │
│  │  Mau Peer 1   │   │  Mau Peer 2   │   │  Mau Peer 3   │    │
│  │  Container    │   │  Container    │   │  Container    │    │
│  │               │   │               │   │               │    │
│  │  - Account    │   │  - Account    │   │  - Account    │    │
│  │  - Server     │   │  - Server     │   │  - Server     │    │
│  │  - DHT Node   │   │  - DHT Node   │   │  - DHT Node   │    │
│  │  - Files      │   │  - Files      │   │  - Files      │    │
│  └───────┬───────┘   └───────┬───────┘   └───────┬───────┘    │
│          │                   │                   │            │
│          └───────────────────┼───────────────────┘            │
│                              │                                │
│  ┌───────────────────────────┴────────────────────────────┐   │
│  │              Toxiproxy (Optional)                      │   │
│  │  - Latency injection                                   │   │
│  │  - Bandwidth limiting                                  │   │
│  │  - Network partitions                                  │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                │
│  ┌────────────────────────────────────────────────────────┐   │
│  │         Bootstrap Node (Optional)                      │   │
│  │  - Kademlia DHT seed                                   │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Observability Layer                         │
│                                                                 │
│  - Container logs (JSON structured)                             │
│  - Test result artifacts (JSON)                                 │
│  - File state snapshots (for debugging)                         │
│  - Network traffic traces (optional)                            │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. **Test Coordinator**
- **Language:** Go
- **Framework:** `go test` with Testcontainers-Go
- **Responsibilities:**
  - Parse test configuration (peer count, network topology, test scenario)
  - Build/pull Mau Docker image
  - Create isolated Docker network
  - Spawn Mau peer containers
  - Inject friend relationships
  - Inject files to sync
  - Wait for synchronization
  - Assert expected state
  - Collect logs and artifacts
  - Cleanup containers and networks

#### 2. **Mau Peer Container**
- **Base Image:** `golang:1.21-alpine` (multi-stage build)
- **Contents:**
  - Mau binary (server + DHT node)
  - PGP keyring initialization
  - Configuration via environment variables
  - Healthcheck endpoint
  - Structured logging to stdout

**Container Lifecycle:**
1. **Init:** Generate PGP account or import existing
2. **Bootstrap:** Connect to DHT seed nodes (if provided)
3. **Ready:** HTTP server listening, DHT routing table populated
4. **Runtime:** Accept friend additions, file synchronization
5. **Shutdown:** Graceful stop, flush logs

#### 3. **Toxiproxy Sidecar (Optional)**
- **When to use:** Chaos/resilience tests
- **Configuration:**
  - Each Mau peer routes through local Toxiproxy instance
  - Toxiproxy forwards to actual Mau server
  - Test coordinator controls toxics via HTTP API

**Example Proxy Configuration:**
```json
{
  "name": "mau_peer_1",
  "listen": "0.0.0.0:8080",
  "upstream": "mau-peer-1:8080",
  "enabled": true
}
```

#### 4. **Bootstrap Node**
- **Purpose:** Seed Kademlia DHT for peer discovery
- **Implementation:** 
  - Dedicated Mau instance with known address
  - All peers configured with bootstrap node in environment
  - Not tested, acts as infrastructure

---

## Technology Stack

### Core Technologies

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Container Runtime** | Docker | 20.10+ | Peer isolation |
| **Orchestration** | Testcontainers-Go | v0.27+ | Programmatic container management |
| **Test Framework** | Go testing | 1.21+ | Test execution |
| **Assertions** | Testify | v1.8+ | Rich assertions (already in Mau) |
| **Network Proxy** | Toxiproxy | 2.5+ | Network condition simulation |
| **Logging** | Zerolog / Zap | Latest | Structured JSON logging |

### Supporting Tools

| Tool | Purpose | When Used |
|------|---------|-----------|
| **Docker Compose** | Manual test environment setup | Local development/debugging |
| **jq** | Log parsing/filtering | Debugging failed tests |
| **Make** | Build automation | CI/CD pipeline |
| **GitHub Actions** | CI/CD runner | Automated testing |

---

## Test Scenarios

### Level 1: Basic Functionality (Deterministic)

#### TC-001: Two-Peer Discovery
**Objective:** Verify two Mau peers can discover each other via Kademlia DHT  
**Setup:**
- Peer A, Peer B
- Shared bootstrap node
- Same Docker network

**Steps:**
1. Start bootstrap node
2. Start Peer A, configure bootstrap node
3. Start Peer B, configure bootstrap node
4. Wait for DHT routing tables to populate
5. Query Peer A for Peer B's fingerprint
6. Query Peer B for Peer A's fingerprint

**Assertions:**
- Peer A finds Peer B within 5 seconds
- Peer B finds Peer A within 5 seconds
- DHT distance calculation is correct

---

#### TC-002: Two-Peer Friend Sync
**Objective:** Verify friend relationship establishment and file synchronization  
**Setup:**
- Peer A, Peer B
- Peer A adds Peer B as friend (exchange public keys)
- Peer B adds Peer A as friend

**Steps:**
1. Start both peers
2. Exchange public keys via test coordinator
3. Inject friend relationship on both sides
4. Peer A creates file `hello.txt` encrypted for Peer B
5. Wait for synchronization
6. Verify Peer B has `hello.txt`
7. Decrypt and verify content matches

**Assertions:**
- Friend relationship established (both sides)
- File appears on Peer B within 10 seconds
- Decrypted content matches original
- File permissions/metadata preserved

---

#### TC-003: Multi-Peer Sync (N=5)
**Objective:** Verify synchronization across 5 peers in a friend graph  
**Friend Graph:**
```
    A
   /|\
  B C D
   \ /
    E
```

**Setup:**
- 5 peers
- Friend relationships as shown
- Peer A creates public file

**Steps:**
1. Start all peers
2. Establish friend relationships
3. Peer A publishes public file
4. Wait for propagation
5. Verify all peers receive file

**Assertions:**
- All peers have file within 30 seconds
- File SHA256 matches across all peers
- No duplicate file fetches (check logs)

---

#### TC-004: Version Conflict Resolution
**Objective:** Test behavior when two peers edit same file concurrently  
**Setup:**
- Peer A, Peer B (mutual friends)
- Both have `shared.txt` version 1

**Steps:**
1. Network partition: isolate A and B
2. Peer A edits `shared.txt` → version 2a
3. Peer B edits `shared.txt` → version 2b
4. Restore network
5. Wait for synchronization

**Assertions:**
- Both versions exist (`.versions/` directory)
- Latest version determined by timestamp or conflict resolution rules
- No data loss

---

### Level 2: Resilience Testing (Chaos)

#### TC-101: Peer Crash During Sync
**Objective:** Verify resilience when peer crashes mid-synchronization  
**Setup:**
- Peer A, Peer B, Peer C (all friends)
- Peer A has large file (100MB)

**Steps:**
1. Peer A starts sharing file
2. Peer B starts downloading (50% complete)
3. **Kill Peer A container**
4. Wait 10 seconds
5. **Restart Peer A**
6. Verify Peer B resumes download

**Assertions:**
- Peer B resumes from last checkpoint (HTTP Range request)
- Download completes successfully
- File SHA256 matches
- Peer C unaffected

---

#### TC-102: Network Partition (Split Brain)
**Objective:** Test synchronization after network partition heals  
**Friend Graph:**
```
Partition 1: A - B
Partition 2: C - D
```

**Setup:**
- 4 peers in two groups
- Toxiproxy creates network partition

**Steps:**
1. All peers connected initially
2. Create partition: A-B can't reach C-D
3. Peer A publishes file X
4. Peer C publishes file Y
5. Wait 30 seconds
6. **Heal partition**
7. Wait for sync

**Assertions:**
- After healing, all peers have both files (X and Y)
- No file corruption
- Sync time < 60 seconds

---

#### TC-103: High Latency Network (500ms)
**Objective:** Verify synchronization under high latency  
**Setup:**
- 3 peers
- Toxiproxy adds 500ms latency ± 100ms jitter

**Steps:**
1. Start all peers with latency toxic
2. Peer A publishes 10 files (1KB each)
3. Measure sync time

**Assertions:**
- Sync completes (may be slow)
- No timeout errors
- All files synced correctly
- Test logs latency measurements

---

#### TC-104: Bandwidth Limitation (10 KB/s)
**Objective:** Test large file sync under bandwidth constraints  
**Setup:**
- Peer A, Peer B
- Toxiproxy limits bandwidth to 10 KB/s
- File size: 1 MB

**Steps:**
1. Start both peers
2. Apply bandwidth toxic
3. Peer A shares file
4. Measure sync time

**Assertions:**
- Sync time ~= 100 seconds (1MB / 10KB/s)
- No connection drops
- File SHA256 correct

---

#### TC-105: Packet Loss (10%)
**Objective:** Verify TCP retransmission handles packet loss  
**Setup:**
- Peer A, Peer B
- Toxiproxy `slicer` toxic with 10% loss simulation

**Steps:**
1. Start both peers
2. Apply packet loss toxic
3. Peer A shares 100 small files
4. Monitor sync

**Assertions:**
- All files eventually sync
- Retransmissions visible in logs
- Sync time < 5 minutes

---

### Level 3: Stress Testing

#### TC-201: 10-Peer Full Mesh
**Objective:** Test scalability with 10 peers all friends with each other  
**Setup:**
- 10 peers
- 45 friend relationships (full mesh)
- Peer 1 publishes file

**Steps:**
1. Start all peers
2. Establish all friend relationships
3. Peer 1 publishes file
4. Wait for propagation

**Assertions:**
- All peers receive file within 2 minutes
- DHT routing table sizes < 20 entries (k-bucket limit)
- No memory leaks (check container stats)

---

#### TC-202: 100-Peer Network (Sparse Graph)
**Objective:** Validate DHT performance with 100 peers  
**Friend Graph:** Random graph, average degree = 5  
**Setup:**
- 100 peers
- Random friend relationships
- Bootstrap node

**Steps:**
1. Start all peers (parallel batches)
2. Establish friend relationships
3. 10 random peers publish files
4. Wait for propagation

**Assertions:**
- DHT queries succeed for all peers
- Average lookup time < 1 second
- Sync eventually reaches all connected peers
- Test completes in < 30 minutes

---

#### TC-203: Churn Test (Peers Join/Leave)
**Objective:** Test DHT stability under peer churn  
**Setup:**
- Initial: 20 peers
- Every 30 seconds: 2 peers leave, 2 new peers join
- Duration: 10 minutes

**Steps:**
1. Start initial 20 peers
2. Start churn loop
3. Publish file every minute from random peer
4. Monitor sync success rate

**Assertions:**
- File sync success rate > 95%
- DHT routing table recovers from churn
- No peer becomes permanently isolated

---

### Level 4: Security Testing

#### TC-301: Unauthorized File Access
**Objective:** Verify encrypted files not accessible without decryption key  
**Setup:**
- Peer A (file owner)
- Peer B (authorized friend)
- Peer C (unauthorized, not a friend)

**Steps:**
1. Peer A creates file encrypted for Peer B only
2. Peer C attempts to download file (if discoverable)
3. Peer C attempts to decrypt file

**Assertions:**
- Peer C cannot decrypt file
- File content remains confidential
- No plaintext leakage in logs

---

#### TC-302: DHT Sybil Attack Resistance
**Objective:** Test DHT behavior under Sybil attack (future work)  
**Setup:**
- 10 honest peers
- 50 malicious peers with coordinated IDs

**Steps:**
1. Honest peers establish DHT
2. Malicious peers join with IDs near target peer
3. Attempt to monopolize routing table
4. Honest peer tries to find another honest peer

**Assertions:**
- Lookup success rate > 90%
- S/Kademlia defenses (if implemented) mitigate attack

---

## File Structure

```
mau/
├── e2e/                              # E2E test framework root
│   ├── PLAN.md                       # This document
│   ├── README.md                     # Quick start guide
│   ├── Makefile                      # Build and test automation
│   │
│   ├── framework/                    # Core test framework
│   │   ├── testenv/                  # Test environment setup
│   │   │   ├── testenv.go            # Testcontainers orchestration
│   │   │   ├── peer.go               # Mau peer container wrapper
│   │   │   ├── network.go            # Docker network management
│   │   │   ├── toxiproxy.go          # Toxiproxy integration
│   │   │   └── bootstrap.go          # Bootstrap node management
│   │   │
│   │   ├── assertions/               # Custom assertions
│   │   │   ├── sync.go               # File sync assertions
│   │   │   ├── dht.go                # Kademlia DHT assertions
│   │   │   └── friend.go             # Friend relationship assertions
│   │   │
│   │   ├── helpers/                  # Utility functions
│   │   │   ├── pgp.go                # PGP key generation/management
│   │   │   ├── files.go              # File creation/comparison
│   │   │   ├── logs.go               # Log collection/parsing
│   │   │   └── wait.go               # Wait strategies
│   │   │
│   │   └── types/                    # Shared types
│   │       ├── config.go             # Test configuration
│   │       ├── peer.go               # Peer metadata
│   │       └── result.go             # Test result structures
│   │
│   ├── scenarios/                    # Test scenarios
│   │   ├── basic/                    # Level 1 tests
│   │   │   ├── discovery_test.go     # TC-001: Two-peer discovery
│   │   │   ├── friend_sync_test.go   # TC-002: Two-peer friend sync
│   │   │   ├── multi_peer_test.go    # TC-003: Multi-peer sync
│   │   │   └── version_conflict_test.go # TC-004: Version conflicts
│   │   │
│   │   ├── resilience/               # Level 2 tests
│   │   │   ├── peer_crash_test.go    # TC-101: Peer crash
│   │   │   ├── partition_test.go     # TC-102: Network partition
│   │   │   ├── latency_test.go       # TC-103: High latency
│   │   │   ├── bandwidth_test.go     # TC-104: Bandwidth limits
│   │   │   └── packet_loss_test.go   # TC-105: Packet loss
│   │   │
│   │   ├── stress/                   # Level 3 tests
│   │   │   ├── full_mesh_test.go     # TC-201: 10-peer mesh
│   │   │   ├── large_network_test.go # TC-202: 100-peer network
│   │   │   └── churn_test.go         # TC-203: Peer churn
│   │   │
│   │   └── security/                 # Level 4 tests
│   │       ├── unauthorized_access_test.go # TC-301
│   │       └── sybil_attack_test.go  # TC-302 (future)
│   │
│   ├── docker/                       # Docker configurations
│   │   ├── Dockerfile.mau            # Mau peer image
│   │   ├── Dockerfile.bootstrap      # Bootstrap node image
│   │   ├── docker-compose.yml        # Manual test environment
│   │   └── entrypoint.sh             # Container entrypoint script
│   │
│   ├── configs/                      # Test configurations
│   │   ├── default.json              # Default test config
│   │   ├── ci.json                   # CI-optimized config
│   │   └── stress.json               # Stress test config
│   │
│   ├── scripts/                      # Utility scripts
│   │   ├── build-images.sh           # Build Docker images
│   │   ├── run-tests.sh              # Run test suite
│   │   ├── parse-logs.sh             # Extract logs from failed tests
│   │   └── generate-report.sh        # Generate HTML test report
│   │
│   └── docs/                         # Documentation
│       ├── writing-tests.md          # Guide for adding new tests
│       ├── debugging.md              # Debugging failed tests
│       ├── architecture.md           # Framework architecture
│       └── toxiproxy-guide.md        # Toxiproxy usage guide
│
├── go.mod                            # Add e2e dependencies
└── .github/
    └── workflows/
        └── e2e-tests.yml             # GitHub Actions workflow
```

---

## Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
**Goal:** Basic framework with simple 2-peer tests + interactive CLI foundation

**Deliverables:**
- [ ] Docker image for Mau peer (`e2e/docker/Dockerfile.mau`)
- [ ] Shared testenv library (`e2e/framework/testenv/`)
- [ ] **Interactive CLI structure** (`e2e/cmd/mau-e2e/`)
- [ ] **`mau-e2e up/down` commands**
- [ ] **`mau-e2e peer add/list` commands**
- [ ] State persistence (`~/.mau-e2e/`)
- [ ] TC-001: Two-peer discovery (automated test)
- [ ] TC-002: Two-peer friend sync (automated test)
- [ ] Makefile for building and running tests
- [ ] CI workflow (GitHub Actions)

**Success Criteria:**
- **Can start 2 peers with `mau-e2e up --peers 2`**
- **Can list peers with `mau-e2e peer list`**
- Tests run locally with `make test-e2e`
- Tests pass in CI
- Logs captured on failure
- Same `testenv` library used by both CLI and tests

**Key Files:**
```
e2e/docker/Dockerfile.mau
e2e/framework/testenv/testenv.go
e2e/framework/testenv/peer.go
e2e/scenarios/basic/discovery_test.go
e2e/scenarios/basic/friend_sync_test.go
e2e/Makefile
.github/workflows/e2e-tests.yml
```

**Example `testenv.go` structure:**
```go
package testenv

import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go"
)

type TestEnv struct {
    ctx     context.Context
    network testcontainers.Network
    peers   []*MauPeer
    t       *testing.T
}

func NewTestEnv(t *testing.T) *TestEnv {
    // Create isolated Docker network
    // Return TestEnv instance
}

func (e *TestEnv) AddPeer(name string) (*MauPeer, error) {
    // Create and start Mau peer container
    // Wait for readiness
    // Return MauPeer wrapper
}

func (e *TestEnv) Cleanup() {
    // Stop all containers
    // Remove network
    // Collect logs
}
```

---

### Phase 2: Multi-Peer & Peer Interaction (Weeks 3-4)
**Goal:** Expand to multi-peer scenarios + file/friend CLI commands

**Deliverables:**
- [ ] Custom assertion library (`e2e/framework/assertions/`)
  - `AssertFilesSynced(peers []*MauPeer, filename string, timeout time.Duration)`
  - `AssertDHTLookup(peer *MauPeer, targetFingerprint string, timeout time.Duration)`
  - `AssertFriendRelationship(peer1, peer2 *MauPeer)`
- [ ] **`mau-e2e friend add/list` commands**
- [ ] **`mau-e2e file add/list/cat` commands**
- [ ] **`mau-e2e peer inspect` command**
- [ ] TC-003: Multi-peer sync (5 peers)
- [ ] TC-004: Version conflict resolution
- [ ] Helper for complex friend graph setup
- [ ] Documentation: `docs/writing-tests.md`

**Success Criteria:**
- **Can manually test 2-peer sync via CLI**
- 5-peer test completes in < 2 minutes
- Assertions provide clear failure messages
- New test cases easy to write (< 50 lines)

---

### Phase 3: Real-time Monitoring + Chaos (Weeks 5-6)
**Goal:** Introduce Toxiproxy and real-time observability

**Deliverables:**
- [ ] Toxiproxy integration (`e2e/framework/testenv/toxiproxy.go`)
- [ ] **`mau-e2e file watch` command (real-time sync events)**
- [ ] **`mau-e2e status --watch` command (live dashboard)**
- [ ] **`mau-e2e net partition/heal` commands**
- [ ] **`mau-e2e net latency/limit` commands**
- [ ] **Color-coded CLI output**
- [ ] Proxy configuration per peer
- [ ] TC-101: Peer crash during sync
- [ ] TC-102: Network partition
- [ ] TC-103: High latency
- [ ] TC-104: Bandwidth limitation
- [ ] TC-105: Packet loss
- [ ] Documentation: `docs/toxiproxy-guide.md`

**Success Criteria:**
- **Can observe sync happening in real-time via CLI**
- **Can create network partitions interactively**
- Toxiproxy dynamically controlled during tests
- Chaos tests reproducible (same seed → same result)
- Tests detect real bugs (validate against known issues)

**Example Toxiproxy usage:**
```go
func TestNetworkPartition(t *testing.T) {
    env := testenv.NewTestEnv(t)
    defer env.Cleanup()

    // Create 4 peers
    peers := env.AddPeers(4)
    
    // Establish friend relationships
    env.MakeFriends(peers[0], peers[1])
    env.MakeFriends(peers[2], peers[3])
    
    // Create network partition: {0,1} vs {2,3}
    partition := env.CreatePartition([]int{0, 1}, []int{2, 3})
    
    // Publish files on both sides
    env.AddFile(peers[0], "fileA.txt", "content A")
    env.AddFile(peers[2], "fileB.txt", "content B")
    
    time.Sleep(5 * time.Second)
    
    // Assert files don't cross partition
    assert.NoFile(t, peers[2], "fileA.txt")
    assert.NoFile(t, peers[0], "fileB.txt")
    
    // Heal partition
    partition.Heal()
    
    // Assert files eventually sync
    assertions.AssertFilesSynced(t, peers, "fileA.txt", 60*time.Second)
    assertions.AssertFilesSynced(t, peers, "fileB.txt", 60*time.Second)
}
```

---

### Phase 4: Stress Testing (Weeks 7-8)
**Goal:** Validate scalability and performance

**Deliverables:**
- [ ] TC-201: 10-peer full mesh
- [ ] TC-202: 100-peer network (if CI resources allow)
- [ ] TC-203: Peer churn
- [ ] Performance metrics collection
- [ ] Memory/CPU usage monitoring
- [ ] Test result trending (store results in Git)

**Success Criteria:**
- 10-peer test completes in < 5 minutes
- 100-peer test completes in < 30 minutes (optional)
- No memory leaks detected
- Performance baselines established

**Resource Considerations:**
- 100-peer test may require dedicated CI runners
- Consider matrix testing: run 100-peer test weekly, not on every PR
- Implement early exit if resource exhaustion detected

---

### Phase 5: Advanced CLI Features (Weeks 9-10)
**Goal:** Complete interactive feature set

**Deliverables:**
- [ ] **Interactive shell mode (`mau-e2e shell`)**
- [ ] **Predefined scenarios (`mau-e2e scenario <name>`)**
- [ ] **Snapshot/restore (`mau-e2e snapshot/restore`)**
- [ ] **DHT commands (`dht lookup/table`)**
- [ ] Structured logging with trace IDs
- [ ] Log aggregation script (`scripts/parse-logs.sh`)
- [ ] State snapshot capture on failure (peer file trees, DHT tables)
- [ ] HTML test report generation (`scripts/generate-report.sh`)
- [ ] Documentation: `docs/debugging.md`
- [ ] Automatic log upload to CI artifacts

**Success Criteria:**
- **Interactive shell provides seamless workflow**
- **Can prototype test scenarios interactively**
- Failed test produces:
  - Full logs for all peers
  - File system state snapshots
  - DHT routing table dumps
  - Network traffic summary (if available)
- Debugging time reduced by 80%

**Example Log Format:**
```json
{
  "timestamp": "2026-02-19T14:30:00Z",
  "level": "info",
  "peer": "peer-1",
  "fingerprint": "ABAF11C65A2970B130ABE3C479BE3E4300411886",
  "trace_id": "test-tc002-abc123",
  "component": "sync",
  "event": "file_download_started",
  "file": "hello.txt",
  "source_peer": "peer-2",
  "source_fingerprint": "BBAF11C65A2970B130ABE3C479BE3E4300411887"
}
```

---

### Phase 6: Polish & Documentation (Weeks 11-12)
**Goal:** Production-ready framework with excellent docs

**Deliverables:**
- [ ] TC-301: Unauthorized file access
- [ ] TC-302: DHT Sybil attack (basic version)
- [ ] **Comprehensive CLI documentation**
- [ ] **Video tutorial (screencast of interactive usage)**
- [ ] **Example demo scripts**
- [ ] Parallel test execution in CI
- [ ] Test result caching (skip unchanged tests)
- [ ] Nightly stress test runs
- [ ] Security test suite in separate workflow
- [ ] Badge generation (test pass rate, coverage)
- [ ] Integration verification (ensure CLI + tests share code)

**Success Criteria:**
- **New developer can use CLI productively in < 15 minutes**
- **Video tutorial demonstrates P2P sync visually**
- CI pipeline completes in < 15 minutes (basic tests)
- Nightly stress tests run without supervision
- Security tests detect unauthorized access attempts
- Test failures block PR merges

**GitHub Actions Workflow Structure:**
```yaml
name: E2E Tests

on:
  pull_request:
  push:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # Nightly at 2 AM

jobs:
  basic-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: docker/setup-buildx-action@v2
      - name: Build Mau image
        run: make -C e2e build-image
      - name: Run basic tests
        run: make -C e2e test-basic
      - name: Upload logs
        if: failure()
        uses: actions/upload-artifact@v3
        with:
          name: test-logs-basic
          path: e2e/test-results/

  chaos-tests:
    runs-on: ubuntu-latest
    steps:
      # Similar structure
      
  stress-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'schedule'  # Only nightly
    steps:
      # Run TC-202 (100-peer test)
```

---

## Example Test Case Walkthrough

### Test: TC-002 - Two-Peer Friend Sync

**File:** `e2e/scenarios/basic/friend_sync_test.go`

```go
package basic

import (
    "strings"
    "testing"
    "time"
    
    "github.com/mau-network/mau/e2e/framework/assertions"
    "github.com/mau-network/mau/e2e/framework/testenv"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestTwoP eerFriendSync(t *testing.T) {
    // Step 1: Create test environment
    env := testenv.NewTestEnv(t)
    defer env.Cleanup()  // Ensures cleanup even on test failure
    
    // Step 2: Start two Mau peers
    peerA, err := env.AddPeer("peer-a")
    require.NoError(t, err, "Failed to create peer A")
    
    peerB, err := env.AddPeer("peer-b")
    require.NoError(t, err, "Failed to create peer B")
    
    // Step 3: Exchange public keys and establish friend relationship
    err = env.MakeFriends(peerA, peerB)
    require.NoError(t, err, "Failed to establish friendship")
    
    // Step 4: Verify friend relationship from both sides
    assertions.AssertFriendRelationship(t, peerA, peerB)
    assertions.AssertFriendRelationship(t, peerB, peerA)
    
    // Step 5: Peer A creates a file encrypted for Peer B
    fileContent := "Hello from Peer A!"
    err = peerA.AddFile("hello.txt", strings.NewReader(fileContent), []string{peerB.Fingerprint()})
    require.NoError(t, err, "Failed to create file on peer A")
    
    // Step 6: Wait for synchronization (with timeout)
    syncTimeout := 30 * time.Second
    err = assertions.WaitForFile(t, peerB, "hello.txt", syncTimeout)
    require.NoError(t, err, "File did not sync to peer B within timeout")
    
    // Step 7: Verify file content matches
    content, err := peerB.ReadFile("hello.txt")
    require.NoError(t, err, "Failed to read file from peer B")
    assert.Equal(t, fileContent, content, "File content mismatch")
    
    // Step 8: Verify file is encrypted (PGP format)
    rawContent, err := peerB.ReadFileRaw("hello.txt")
    require.NoError(t, err, "Failed to read raw file")
    assert.Contains(t, rawContent, "-----BEGIN PGP MESSAGE-----", "File not encrypted")
    
    // Step 9: Check synchronization logs for debugging
    logs := peerB.GetLogs()
    assert.Contains(t, logs, "file_download_completed", "Sync event not logged")
}
```

**How This Test Executes:**

1. **Test Environment Creation:**
   - `testenv.NewTestEnv(t)` creates isolated Docker network `mau-test-<uuid>`
   - Initializes cleanup handlers

2. **Peer A Container Startup:**
   - Pulls/uses `mau-e2e:latest` image
   - Generates PGP account or uses pre-generated
   - Starts HTTP server on random port (mapped to host)
   - Joins DHT with bootstrap node (if configured)
   - Exposes health endpoint: `GET /health`
   - Testcontainers waits for healthy status (max 30s)

3. **Peer B Container Startup:**
   - Same process as Peer A
   - Different fingerprint, different port

4. **Friend Relationship Setup:**
   - Test coordinator extracts Peer B's public key via API: `GET /p2p/<peer-b-fpr>/account.pgp`
   - Injects into Peer A's keyring: `POST /admin/friends` (test-only endpoint)
   - Repeats in reverse direction
   - Verifies keyring files created: `.mau/<peer-fpr>.pgp`

5. **File Creation:**
   - Test coordinator calls Peer A API: `POST /admin/files`
     ```json
     {
       "name": "hello.txt",
       "content": "SGVsbG8gZnJvbSBQZWVyIEEh",  // base64
       "encrypt_for": ["BBAF..."] // Peer B fingerprint
     }
     ```
   - Peer A encrypts file with Peer B's public key
   - Writes to `<peer-a-fpr>/hello.txt.pgp`

6. **Synchronization:**
   - Peer B periodically polls Peer A: `GET /p2p/<peer-a-fpr>` (If-Modified-Since header)
   - Response includes `hello.txt` metadata
   - Peer B downloads: `GET /p2p/<peer-a-fpr>/hello.txt`
   - Verifies signature, decrypts, writes to local storage

7. **Assertion:**
   - Test coordinator calls Peer B: `GET /admin/files/hello.txt`
   - Decrypts and returns plaintext
   - Compares with original content

8. **Cleanup:**
   - `defer env.Cleanup()` triggers
   - Stops containers
   - Collects logs to `e2e/test-results/<test-name>/`
   - Removes Docker network
   - On failure: preserves container state for debugging

**Execution Time:** ~15 seconds (including container startup)

---

## Framework Comparison

### Approach 1: Pure Docker Compose
**How it works:**
- Define all peers in `docker-compose.yml`
- Use shell scripts to orchestrate (docker-compose up/down)
- Manual assertion via `docker exec` commands

**Pros:**
- ✅ Simple to understand
- ✅ Easy to run manually for debugging
- ✅ No Go dependencies

**Cons:**
- ❌ Not programmatic - hard to parameterize (N peers)
- ❌ Poor test isolation (shared Docker Compose project)
- ❌ Manual cleanup prone to errors
- ❌ Difficult to integrate with `go test`
- ❌ No automatic log collection on failure

**Verdict:** ❌ **Not Recommended** - Good for manual exploration, bad for automated testing

---

### Approach 2: Testcontainers-Go (Recommended)
**How it works:**
- Go test code creates containers programmatically
- Full control over lifecycle, networking, configuration
- Native integration with `go test`

**Pros:**
- ✅ **Type-safe, programmatic** control
- ✅ **Automatic cleanup** with `defer`
- ✅ **Parameterized tests** (easy to vary peer count)
- ✅ **Test isolation** (each test gets unique network)
- ✅ **Rich ecosystem** (wait strategies, log streaming)
- ✅ **CI-friendly** (integrates with GitHub Actions)

**Cons:**
- ⚠️ Requires Docker daemon (already needed for Mau development)
- ⚠️ Learning curve for Testcontainers API (well-documented)

**Verdict:** ✅ **Recommended** - Best balance of control and maintainability

---

### Approach 3: Kubernetes-based (e.g., kind, k3d)
**How it works:**
- Deploy Mau peers as Kubernetes pods
- Use Kubernetes CRDs for test orchestration
- Tools: Kubetest2, Chainsaw, Sonobuoy

**Pros:**
- ✅ Production-like environment
- ✅ Advanced networking (NetworkPolicies for partitions)
- ✅ Resource management (CPU/memory limits)

**Cons:**
- ❌ **Massive overkill** for Mau's scope
- ❌ Slow startup time (k8s cluster initialization)
- ❌ Complex debugging
- ❌ CI resource intensive

**Verdict:** ❌ **Not Recommended** - Overkill, stick with Docker

---

### Approach 4: Custom Test Harness (like Ethereum Hive)
**How it works:**
- Build custom orchestration tool in Go
- Test scenarios as separate binaries
- Client implementations containerized

**Pros:**
- ✅ **Maximum flexibility**
- ✅ **Client-agnostic** (could test Rust/Python Mau implementations)
- ✅ **Reusable across projects**

**Cons:**
- ❌ **Huge development effort** (weeks to build harness)
- ❌ **Maintenance burden**
- ❌ Not justified for single-implementation project (Mau only has Go impl)

**Verdict:** ⚠️ **Overkill Now, Revisit Later** - Good if Mau gets multiple implementations

---

### Recommendation Matrix

| Criterion | Docker Compose | Testcontainers-Go | Kubernetes | Custom Harness |
|-----------|---------------|-------------------|------------|----------------|
| **Ease of Use** | ★★★★☆ | ★★★☆☆ | ★☆☆☆☆ | ★★☆☆☆ |
| **Programmatic Control** | ★☆☆☆☆ | ★★★★★ | ★★★☆☆ | ★★★★★ |
| **Test Isolation** | ★★☆☆☆ | ★★★★★ | ★★★★★ | ★★★★☆ |
| **CI/CD Integration** | ★★☆☆☆ | ★★★★★ | ★★★☆☆ | ★★★☆☆ |
| **Debugging** | ★★★★☆ | ★★★★☆ | ★★☆☆☆ | ★★★☆☆ |
| **Maintenance** | ★★★☆☆ | ★★★★☆ | ★☆☆☆☆ | ★★☆☆☆ |
| **Scalability (100+ peers)** | ★★☆☆☆ | ★★★★☆ | ★★★★★ | ★★★★☆ |
| **Setup Time** | 5 min | 15 min | 60 min | 120 min |

**Final Recommendation:** **Testcontainers-Go** with Docker Compose for manual debugging

---

## CI/CD Integration

### GitHub Actions Workflow Design

**File:** `.github/workflows/e2e-tests.yml`

```yaml
name: E2E Tests

on:
  pull_request:
    paths:
      - '**.go'
      - 'e2e/**'
      - '.github/workflows/e2e-tests.yml'
  push:
    branches: [main, develop]
  schedule:
    - cron: '0 2 * * *'  # Nightly stress tests

env:
  GO_VERSION: '1.21'
  DOCKER_BUILDKIT: 1

jobs:
  build-image:
    name: Build Mau E2E Image
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Build and export
        uses: docker/build-push-action@v5
        with:
          context: .
          file: e2e/docker/Dockerfile.mau
          tags: mau-e2e:${{ github.sha }}
          outputs: type=docker,dest=/tmp/mau-e2e.tar
      
      - name: Upload image artifact
        uses: actions/upload-artifact@v4
        with:
          name: mau-e2e-image
          path: /tmp/mau-e2e.tar
          retention-days: 1

  test-basic:
    name: Basic Tests (Level 1)
    needs: build-image
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
      
      - name: Download image
        uses: actions/download-artifact@v4
        with:
          name: mau-e2e-image
          path: /tmp
      
      - name: Load image
        run: docker load --input /tmp/mau-e2e.tar
      
      - name: Run basic tests
        run: |
          cd e2e
          go test -v -timeout 10m ./scenarios/basic/...
        env:
          MAU_E2E_IMAGE: mau-e2e:${{ github.sha }}
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: test-results-basic
          path: e2e/test-results/

  test-resilience:
    name: Resilience Tests (Level 2)
    needs: build-image
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      # Similar to test-basic
      - name: Run resilience tests
        run: |
          cd e2e
          go test -v -timeout 25m ./scenarios/resilience/...

  test-stress:
    name: Stress Tests (Level 3)
    needs: build-image
    runs-on: ubuntu-latest-8-cores  # Larger runner
    if: github.event_name == 'schedule' || contains(github.event.head_commit.message, '[stress]')
    timeout-minutes: 60
    steps:
      # Similar to test-basic
      - name: Run stress tests
        run: |
          cd e2e
          go test -v -timeout 50m ./scenarios/stress/...

  test-security:
    name: Security Tests (Level 4)
    needs: build-image
    runs-on: ubuntu-latest
    timeout-minutes: 20
    steps:
      - name: Run security tests
        run: |
          cd e2e
          go test -v -timeout 15m ./scenarios/security/...

  report:
    name: Generate Test Report
    needs: [test-basic, test-resilience, test-security]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Download all results
        uses: actions/download-artifact@v4
        with:
          path: all-results
      
      - name: Generate HTML report
        run: |
          cd e2e
          ./scripts/generate-report.sh ../all-results
      
      - name: Upload report
        uses: actions/upload-artifact@v4
        with:
          name: test-report
          path: e2e/report.html
      
      - name: Comment PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            // Parse test results and post summary comment
```

### Optimization Strategies

1. **Parallel Test Execution:**
   - Use `t.Parallel()` in Go tests where safe
   - Run test levels (basic/resilience/stress) in parallel jobs
   - Resource limits: max 4 parallel stress tests

2. **Image Caching:**
   - Cache Mau Docker image layers in GitHub Actions
   - Only rebuild on source changes
   - Use Docker Buildx cache export

3. **Test Result Caching:**
   - Hash test inputs (code + config)
   - Skip tests if hash matches previous run
   - Stored in GitHub Actions cache

4. **Fast Failure:**
   - Run basic tests first
   - Fail fast on basic test failures
   - Stress tests only on nightly or manual trigger

5. **Resource Management:**
   - Limit concurrent containers per test (max 20)
   - Use Docker resource limits (CPU/memory)
   - Clean up orphaned containers with reaper

---

## Debugging & Observability

### Log Collection Strategy

**Structured Logs:**
All Mau peers emit JSON logs to stdout:

```json
{
  "timestamp": "2026-02-19T14:30:00Z",
  "level": "info",
  "peer_id": "peer-a",
  "fingerprint": "ABAF11C65A2970B130ABE3C479BE3E4300411886",
  "trace_id": "tc002-run42",
  "component": "sync",
  "event": "file_download_started",
  "file": "hello.txt",
  "source_peer": "peer-b",
  "bytes": 1024
}
```

**Fields:**
- `trace_id`: Links all logs from one test execution
- `peer_id`: Container name (e.g., `peer-a`)
- `component`: `dht`, `sync`, `server`, `keyring`
- `event`: Structured event name

**Collection:**
- Testcontainers auto-captures stdout/stderr
- On test failure: dump to `e2e/test-results/<test-name>/logs/<peer-id>.json`
- Use `jq` for filtering: `jq '.component == "sync"' peer-a.json`

---

### State Snapshots

**What to capture on test failure:**

1. **File System State:**
   - Peer directory trees (`.mau/`, `<fpr>/`)
   - `tar -czf peer-a-files.tar.gz /data`

2. **DHT Routing Tables:**
   - Admin API: `GET /admin/dht/routing-table`
   - Save as JSON

3. **Friend Lists:**
   - Admin API: `GET /admin/friends`
   - Shows keyring state

4. **Container Stats:**
   - `docker stats` snapshot (CPU/memory usage)
   - Helps detect resource exhaustion

5. **Network State:**
   - Active Toxiproxy toxics
   - Container connectivity matrix

**Automated Snapshot Script:**
```go
func (e *TestEnv) CaptureSnapshot(testName string) error {
    snapshotDir := filepath.Join("test-results", testName, "snapshots")
    os.MkdirAll(snapshotDir, 0755)
    
    for _, peer := range e.peers {
        // Capture file tree
        peer.ExecTar("/data", filepath.Join(snapshotDir, peer.Name+"-files.tar.gz"))
        
        // Capture DHT state
        dht, _ := peer.GetDHTState()
        writeJSON(filepath.Join(snapshotDir, peer.Name+"-dht.json"), dht)
        
        // Capture logs
        logs := peer.GetLogs()
        os.WriteFile(filepath.Join(snapshotDir, peer.Name+".log"), logs, 0644)
    }
    return nil
}
```

---

### Debugging Workflow

**When a test fails:**

1. **Check CI Artifacts:**
   - Download `test-results-<level>.zip`
   - Extract to local machine

2. **Read Test Summary:**
   - `test-results/<test-name>/summary.json`
   - Shows which assertion failed

3. **Filter Logs by Trace ID:**
   ```bash
   cd test-results/TestTwoPeerFriendSync/logs
   jq '. | select(.trace_id == "tc002-run42")' peer-*.json | less
   ```

4. **Inspect File State:**
   ```bash
   tar -xzf snapshots/peer-a-files.tar.gz
   tree data/
   ```

5. **Reproduce Locally:**
   ```bash
   # Use Docker Compose for manual control
   cd e2e
   docker-compose -f docker/docker-compose.yml up
   # Manually trigger actions via API
   curl -X POST http://localhost:8080/admin/files -d '...'
   ```

6. **Enable Verbose Logging:**
   ```go
   // In test file
   env.SetLogLevel("debug")  // Enables DEBUG level logs
   ```

7. **Pause Test on Failure:**
   ```go
   if t.Failed() {
       fmt.Println("Test failed, containers still running. Press enter to cleanup...")
       bufio.NewReader(os.Stdin).ReadString('\n')
   }
   ```

---

### Observability Tools

| Tool | Purpose | Integration |
|------|---------|-------------|
| **jq** | Log filtering/analysis | Manual, CI scripts |
| **Docker logs** | Real-time log tailing | `docker logs -f <container>` |
| **Docker stats** | Resource monitoring | `docker stats` during test |
| **Wireshark/tcpdump** | Network traffic capture (advanced) | Manual debugging |
| **Grafana/Loki** | Log aggregation (future) | Optional for large test suites |

---

## Open Questions & Future Work

### Open Questions

1. **DHT Bootstrap Strategy:**
   - Should tests use a dedicated bootstrap node or peer-to-peer discovery?
   - Trade-off: Bootstrap node simplifies setup but adds dependency

2. **Test Data Persistence:**
   - Should test results be stored in Git for trend analysis?
   - Or use external service (TestRail, Allure)?

3. **Performance Baselines:**
   - What is acceptable sync time for 10 peers? 100 peers?
   - Need empirical data to set thresholds

4. **Chaos Test Reproducibility:**
   - How to ensure random failures are reproducible?
   - Solution: Seed-based randomness with seed in test name

5. **Security Test Scope:**
   - How deep should Sybil attack testing go?
   - May require S/Kademlia implementation first

6. **Test Environment Variables:**
   - Should tests read config from env vars (for CI tuning)?
   - Or strictly use code-defined configs?

### Future Enhancements

#### Phase 7+: Advanced Features

1. **Visual Test Reports:**
   - HTML dashboard with pass/fail trends
   - Peer graph visualization (D3.js)
   - Timeline view of peer interactions

2. **Mutation Testing:**
   - Inject bugs into Mau code
   - Verify E2E tests catch them
   - Measures test effectiveness

3. **Fuzz Testing Integration:**
   - Use `go-fuzz` to generate file content
   - Test PGP encryption with malformed keys
   - Kademlia message fuzzing

4. **Performance Regression Detection:**
   - Store sync time metrics in database
   - Alert on >20% slowdown
   - Integration with GitHub Status Checks

5. **Multi-Platform Testing:**
   - Test on ARM64 (e.g., Raspberry Pi simulation)
   - Windows containers (if Mau supports)

6. **Record/Replay:**
   - Record network interactions during test
   - Replay for deterministic debugging
   - Tools: VCR, go-replay

7. **Chaos Mesh Integration:**
   - More advanced chaos scenarios
   - CPU/memory pressure testing
   - Clock skew simulation (important for PGP timestamp validation)

8. **Contract Testing:**
   - Verify HTTP API backwards compatibility
   - Pact or OpenAPI validation

---

## Conclusion

This E2E testing framework is designed to:

✅ **Validate Mau's core P2P functionality** (discovery, sync, friend management)  
✅ **Detect regressions early** via automated CI/CD integration  
✅ **Simulate real-world conditions** (network failures, high latency, peer churn)  
✅ **Scale from 2 to 100+ peers** with minimal test code changes  
✅ **Provide excellent debugging** with rich logs, state snapshots, and artifacts  
✅ **Remain maintainable** with clear structure and comprehensive documentation  

### Next Steps

1. **Review this plan** with the team
2. **Approve technology choices** (Testcontainers-Go, Toxiproxy)
3. **Prioritize test scenarios** (start with TC-001, TC-002)
4. **Begin Phase 1 implementation**
5. **Iterate based on real-world bugs found**

### Success Metrics

After 6 months of use:
- **Test coverage:** >80% of P2P scenarios
- **Bug detection:** >10 bugs caught before production
- **Developer adoption:** New contributors can add tests in <1 hour
- **CI reliability:** <1% flaky test rate
- **Debugging time:** <30 minutes to root-cause failures

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-19  
**Reviewers:** [To be assigned]  
**Status:** Awaiting Review
