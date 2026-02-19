# Mau E2E Testing Framework

End-to-end testing framework for Mau P2P file synchronization system with **interactive CLI** and **automated testing**.

## Quick Start

### Interactive Mode (Explore & Debug)

```bash
# Install CLI tool
cd e2e
go install ./cmd/mau-e2e

# Start 3 Mau peers
mau-e2e up --peers 3

# Make them friends
mau-e2e friend add peer-0 peer-1
mau-e2e friend add peer-1 peer-2

# Add a file
mau-e2e file add peer-0 hello.txt
# Type content, Ctrl-D to finish

# Watch sync happen in real-time
mau-e2e file watch

# Inspect peer state
mau-e2e peer inspect peer-1

# Clean up
mau-e2e down
```

**Why Interactive Mode?**
- 👁️ **See** P2P synchronization happening in real-time
- 🔬 **Explore** edge cases manually
- 🐛 **Debug** test failures by recreating environment
- 🎓 **Learn** how Mau P2P works through experimentation

### Automated Testing Mode (CI/CD)

```bash
# Run all E2E tests
make test-e2e

# Run specific test suite
cd e2e
go test -v ./scenarios/basic/...

# Run with race detector
go test -race ./scenarios/...

# CI mode (optimized for GitHub Actions)
make test-e2e-ci
```

## Documentation

| Document | Purpose |
|----------|---------|
| [PLAN.md](PLAN.md) | Comprehensive testing framework design (automated tests) |
| [CLI_DESIGN.md](CLI_DESIGN.md) | Interactive CLI architecture and usage |
| [docs/writing-tests.md](docs/writing-tests.md) | How to add new test scenarios |
| [docs/debugging.md](docs/debugging.md) | Troubleshooting failed tests |
| [docs/toxiproxy-guide.md](docs/toxiproxy-guide.md) | Network simulation with Toxiproxy |

## Architecture

The framework operates in **two complementary modes** sharing the same core library:

```
┌─────────────────────────────────────────┐
│         Interactive CLI Mode            │
│    (mau-e2e command-line tool)         │
│                                         │
│  • Manual control                       │
│  • Real-time monitoring                 │
│  • Network simulation                   │
│  • Exploratory testing                  │
└──────────────┬──────────────────────────┘
               │
               │   Shared testenv library
               │
┌──────────────┴──────────────────────────┐
│       Automated Testing Mode            │
│           (go test)                     │
│                                         │
│  • CI/CD integration                    │
│  • Regression detection                 │
│  • Performance benchmarks               │
│  • Chaos engineering                    │
└─────────────────────────────────────────┘
```

## Features

### Interactive CLI (`mau-e2e`)

- ✅ **Environment Management:** Start/stop multi-peer test environments
- ✅ **Peer Control:** Add/remove peers, inspect state, execute commands
- ✅ **Friend Relationships:** Establish/list/remove friendships
- ✅ **File Operations:** Add/list/download files, watch sync in real-time
- ✅ **Network Simulation:** Inject latency, partition networks, limit bandwidth
- ✅ **DHT Operations:** Query routing tables, lookup peers
- ✅ **Snapshots:** Capture and restore environment state
- ✅ **Interactive Shell:** Persistent session with command history
- ✅ **Scenarios:** Run predefined test scenarios

### Automated Tests

- ✅ **Basic Functionality:** 2-peer discovery, friend sync, multi-peer propagation
- ✅ **Resilience:** Peer crashes, network partitions, high latency, packet loss
- ✅ **Stress Testing:** 10-100 peer networks, peer churn, concurrent operations
- ✅ **Security:** Unauthorized access prevention, encryption validation
- ✅ **Chaos Engineering:** Random failures, resource exhaustion

## Example Workflows

### Workflow 1: Basic Sync Test (Interactive)

```bash
$ mau-e2e up --peers 2
✓ Started peers: alice, bob

$ mau-e2e friend add alice bob
✓ Friendship established

$ mau-e2e file add alice test.txt <<< "Hello, Bob!"
✓ File created and encrypted

$ mau-e2e file watch
[15:30:12] bob: Downloading test.txt from alice...
[15:30:13] bob: ✓ Sync complete: test.txt

$ mau-e2e file cat bob test.txt
Hello, Bob!
```

### Workflow 2: Network Partition (Interactive)

```bash
$ mau-e2e up --peers 4

# Create two groups
$ mau-e2e friend add peer-0 peer-1
$ mau-e2e friend add peer-2 peer-3

# Partition network: {0,1} vs {2,3}
$ mau-e2e net partition peer-0,peer-1 peer-2,peer-3

# Add files on both sides
$ mau-e2e file add peer-0 fileA.txt <<< "Group A"
$ mau-e2e file add peer-2 fileB.txt <<< "Group B"

# Verify isolation
$ mau-e2e file list peer-1  # Has fileA only
$ mau-e2e file list peer-3  # Has fileB only

# Heal partition
$ mau-e2e net heal

# Verify convergence (if cross-group friends exist)
$ mau-e2e file watch
```

### Workflow 3: Automated Test

```go
// e2e/scenarios/basic/friend_sync_test.go
func TestTwoP eerFriendSync(t *testing.T) {
    env := testenv.NewTestEnvironment(t)
    defer env.Cleanup()
    
    alice, _ := env.AddPeer("alice")
    bob, _ := env.AddPeer("bob")
    env.MakeFriends(alice, bob)
    
    alice.AddFile("hello.txt", strings.NewReader("Hello!"))
    
    assert.Eventually(t, func() bool {
        return bob.HasFile("hello.txt")
    }, 30*time.Second, 1*time.Second)
    
    content, _ := bob.ReadFile("hello.txt")
    assert.Equal(t, "Hello!", content)
}
```

## Installation

### Prerequisites

- Go 1.21+
- Docker 20.10+
- Docker Compose 2.0+ (optional, for manual setups)

### Build

```bash
# Clone repository
git clone https://github.com/mau-network/mau
cd mau

# Switch to E2E branch
git checkout e2e-tests-framework

# Build Mau E2E Docker image
cd e2e
make build-image

# Install CLI tool
go install ./cmd/mau-e2e

# Verify installation
mau-e2e --version
```

## Usage

### CLI Commands

```bash
# Environment
mau-e2e up [--peers N]           # Start environment
mau-e2e down                     # Tear down
mau-e2e status [--watch]         # Show status

# Peers
mau-e2e peer add <name>          # Add peer
mau-e2e peer list                # List peers
mau-e2e peer inspect <name>      # Show peer details
mau-e2e peer restart <name>      # Restart peer

# Friends
mau-e2e friend add <p1> <p2>     # Make friends
mau-e2e friend list <peer>       # List friends

# Files
mau-e2e file add <peer> <file>   # Add file
mau-e2e file list <peer>         # List files
mau-e2e file cat <peer> <file>   # Show content
mau-e2e file watch               # Watch sync events

# Network
mau-e2e net partition <g1> <g2>  # Create partition
mau-e2e net heal                 # Heal partition
mau-e2e net latency <peer> <ms>  # Add latency
mau-e2e net reset                # Remove all toxics

# Utilities
mau-e2e scenario <name>          # Run scenario
mau-e2e snapshot <dir>           # Capture snapshot
mau-e2e shell                    # Interactive shell
```

### Test Commands

```bash
# Run all tests
make test-e2e

# Run specific suite
make test-e2e-basic              # Basic functionality
make test-e2e-resilience         # Resilience tests
make test-e2e-stress             # Stress tests (slow)

# Run individual test
cd e2e
go test -v -run TestTwoPeerFriendSync ./scenarios/basic/
```

## Configuration

### CLI Configuration

**File:** `~/.mau-e2e/config.json`

```json
{
  "default_peers": 3,
  "docker_image": "mau-e2e:latest",
  "enable_toxiproxy": true,
  "log_level": "info",
  "auto_cleanup": true
}
```

### Test Configuration

**File:** `e2e/configs/default.json`

```json
{
  "timeout": "30s",
  "sync_timeout": "60s",
  "docker_image": "mau-e2e:latest",
  "network_name_prefix": "mau-test",
  "log_level": "debug"
}
```

## CI Integration

The framework runs automatically in GitHub Actions on every PR:

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests

on: [pull_request]

jobs:
  test-basic:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build image
        run: make -C e2e build-image
      - name: Run tests
        run: make -C e2e test-basic
```

## Troubleshooting

### Tests Fail in CI

```bash
# Download test artifacts from GitHub Actions
# Extract logs:
cd test-results/TestTwoPeerFriendSync/logs
jq '. | select(.component == "sync")' peer-*.json

# Reproduce locally
mau-e2e up --peers 2
# ... manually trigger the failing scenario
```

### CLI Can't Find Environment

```bash
# Check state file
cat ~/.mau-e2e/current-env.json

# List Docker containers
docker ps -a --filter label=mau-e2e

# Force cleanup
docker rm -f $(docker ps -aq --filter label=mau-e2e)
rm ~/.mau-e2e/current-env.json
```

### Sync Not Happening

```bash
# Watch logs in real-time
mau-e2e logs peer-0 --follow

# Check DHT connectivity
mau-e2e dht table peer-0

# Verify friendship
mau-e2e friend list peer-0
```

## Development

### Adding a New Test

1. Create test file: `e2e/scenarios/<category>/<name>_test.go`
2. Use `testenv.NewTestEnvironment(t)`
3. Add peers, establish friendships, inject files
4. Assert expected behavior
5. Run: `go test -v ./<category>/`

**Example:**

```go
func TestMyScenario(t *testing.T) {
    env := testenv.NewTestEnvironment(t)
    defer env.Cleanup()
    
    // Your test logic here
}
```

See [docs/writing-tests.md](docs/writing-tests.md) for details.

### Adding a CLI Command

1. Create command file: `e2e/cmd/mau-e2e/commands/<name>.go`
2. Implement using `cobra.Command`
3. Register in `main.go`
4. Add tests in `commands/<name>_test.go`

## Roadmap

- [x] Phase 1: CLI foundation + basic tests (Weeks 1-2)
- [ ] Phase 2: Multi-peer + file/friend commands (Weeks 3-4)
- [ ] Phase 3: Real-time monitoring + chaos (Weeks 5-6)
- [ ] Phase 4: Stress testing (Weeks 7-8)
- [ ] Phase 5: Advanced CLI features (Weeks 9-10)
- [ ] Phase 6: Polish & documentation (Weeks 11-12)

See [PLAN.md](PLAN.md) for detailed roadmap.

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for general contribution guidelines.

**E2E-specific guidelines:**
- Interactive CLI changes should maintain automated test compatibility
- New test scenarios should have CLI equivalents where applicable
- Always test both modes (CLI + automated) before submitting PR

## License

Same as Mau project: GPLv3

## Support

- **Issues:** https://github.com/mau-network/mau/issues
- **Discussions:** https://github.com/mau-network/mau/discussions
- **Documentation:** https://mau.network/docs

---

**Status:** 🚧 Under Development  
**Version:** 0.1.0 (Design Phase)  
**Last Updated:** 2026-02-19
