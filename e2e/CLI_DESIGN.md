# Mau E2E Interactive CLI - Design Document

**Version:** 1.0  
**Date:** 2026-02-19  
**Critical Requirement:** Interactive CLI for manual control and exploratory testing  

---

## Executive Summary

The Mau E2E framework provides **two modes of operation**:

1. **Automated Testing Mode:** Run via `go test` for CI/CD (as described in PLAN.md)
2. **Interactive Mode:** CLI-driven test environment for manual exploration, debugging, and demonstrations

**Key Insight:** Developers need to **see** P2P synchronization happening, not just assert it worked in tests. The interactive CLI makes the invisible visible.

---

## Architecture: Dual-Mode Design

```
┌─────────────────────────────────────────────────────────────┐
│                    Mau E2E Framework                        │
│                                                             │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │  Automated Mode  │         │ Interactive Mode │         │
│  │                  │         │                  │         │
│  │  - go test       │         │  - mau-e2e CLI   │         │
│  │  - CI/CD         │         │  - Docker Compose│         │
│  │  - Assertions    │         │  - Live Control  │         │
│  └────────┬─────────┘         └────────┬─────────┘         │
│           │                            │                   │
│           └──────────┬─────────────────┘                   │
│                      │                                     │
│              ┌───────▼────────┐                            │
│              │  Shared Core   │                            │
│              │  - testenv lib │                            │
│              │  - peer mgmt   │                            │
│              │  - assertions  │                            │
│              └────────────────┘                            │
└─────────────────────────────────────────────────────────────┘
```

**Shared Core:** Both modes use the same underlying `testenv` library, ensuring consistency.

---

## Interactive CLI: `mau-e2e`

### Installation

```bash
cd e2e
go install ./cmd/mau-e2e
mau-e2e --help
```

### Command Structure

```
mau-e2e <command> [options]

ENVIRONMENT MANAGEMENT:
  up          Start a new test environment
  down        Tear down the environment
  status      Show environment status
  logs        Stream logs from peers

PEER MANAGEMENT:
  peer add <name>              Add a new peer
  peer list                    List all peers
  peer rm <name>               Remove a peer
  peer inspect <name>          Show detailed peer state
  peer restart <name>          Restart a peer
  peer exec <name> <cmd>       Execute command in peer container

FRIEND RELATIONSHIPS:
  friend add <peer1> <peer2>   Make peer1 and peer2 friends
  friend list <peer>           Show peer's friend list
  friend rm <peer1> <peer2>    Remove friendship

FILE OPERATIONS:
  file add <peer> <path>       Add file to peer
  file list <peer>             List peer's files
  file get <peer> <file>       Download file from peer
  file cat <peer> <file>       Show decrypted file content
  file rm <peer> <file>        Remove file from peer
  file watch                   Watch file sync events in real-time

DHT OPERATIONS:
  dht lookup <peer> <target>   Lookup target fingerprint from peer
  dht table <peer>             Show peer's routing table
  dht bootstrap <peer>         Trigger bootstrap on peer

NETWORK SIMULATION:
  net partition <group1> <group2>   Create network partition
  net heal                          Heal all partitions
  net latency <peer> <ms>           Add latency to peer
  net limit <peer> <kb/s>           Limit peer bandwidth
  net reset                         Remove all network toxics

UTILITIES:
  snapshot <dir>               Capture full environment snapshot
  restore <snapshot-dir>       Restore environment from snapshot
  scenario <name>              Run predefined scenario
  shell                        Interactive shell mode
```

---

## Example Workflows

### Workflow 1: Basic Sync Exploration

```bash
# 1. Start environment with 2 peers
$ mau-e2e up --peers 2
✓ Created Docker network: mau-test-7a3f
✓ Started bootstrap node
✓ Started peer alice (fingerprint: ABAF...)
✓ Started peer bob (fingerprint: BBCF...)
Environment ready. Use 'mau-e2e shell' for interactive mode.

# 2. Make them friends
$ mau-e2e friend add alice bob
✓ Exchanged public keys
✓ alice added bob to keyring
✓ bob added alice to keyring
Friendship established.

# 3. Add a file to alice
$ mau-e2e file add alice hello.txt
Content (Ctrl-D to finish):
Hello from Alice!
^D
✓ File encrypted for [bob]
✓ File created: alice/hello.txt.pgp
File SHA256: a3f8b2c...

# 4. Watch sync in real-time
$ mau-e2e file watch
[14:30:02] bob: Polling alice for updates...
[14:30:03] bob: Found new file: hello.txt (1.2 KB)
[14:30:03] bob: Downloading alice/hello.txt...
[14:30:04] bob: Download complete, verifying signature...
[14:30:04] bob: Signature valid, decrypting...
[14:30:04] bob: ✓ Sync complete: hello.txt

# 5. Verify bob has the file
$ mau-e2e file cat bob hello.txt
Hello from Alice!

# 6. Check detailed state
$ mau-e2e peer inspect bob
Peer: bob
Fingerprint: BBCF11C65A2970B130ABE3C479BE3E4300411887
Status: running
Uptime: 2m 34s
Friends: 1 (alice)
Files: 1 local, 0 pending sync
DHT peers: 2
Recent activity:
  - 10s ago: Downloaded hello.txt from alice
  - 2m ago: Added friend alice
  - 2m ago: Joined DHT

# 7. Tear down when done
$ mau-e2e down
✓ Stopped 2 peers
✓ Removed network
Environment cleaned up.
```

---

### Workflow 2: Network Partition Simulation

```bash
# Start 4 peers in two groups
$ mau-e2e up --peers 4
✓ Started peers: alice, bob, carol, dave

# Make friends: alice-bob, carol-dave
$ mau-e2e friend add alice bob
$ mau-e2e friend add carol dave

# Create network partition: {alice, bob} vs {carol, dave}
$ mau-e2e net partition alice,bob carol,dave
✓ Created partition via Toxiproxy
✓ alice and bob isolated from carol and dave

# Add files on both sides
$ mau-e2e file add alice fileA.txt <<< "From Group 1"
$ mau-e2e file add carol fileC.txt <<< "From Group 2"

# Wait a bit, then check sync status
$ sleep 10
$ mau-e2e file list bob
Files on bob:
  - fileA.txt (synced from alice) ✓

$ mau-e2e file list dave
Files on dave:
  - fileC.txt (synced from carol) ✓

# Heal partition
$ mau-e2e net heal
✓ Partition healed

# Watch convergence (if they're friends)
$ mau-e2e file watch
# (No sync because alice-carol are not friends)

# Make cross-group friendship
$ mau-e2e friend add alice carol
[14:35:12] alice: Polling carol for updates...
[14:35:13] alice: Found new file: fileC.txt
[14:35:14] carol: Found new file: fileA.txt
# Files sync across partition boundary
```

---

### Workflow 3: Interactive Shell Mode

```bash
$ mau-e2e shell
Welcome to Mau E2E Interactive Shell
Type 'help' for commands, 'exit' to quit

mau> up --peers 3
✓ Started peers: peer-0, peer-1, peer-2

mau> peer list
NAME     STATUS    FINGERPRINT                                  FRIENDS  FILES
peer-0   running   ABAF11C65A2970B130ABE3C479BE3E4300411886   0        0
peer-1   running   BBCF11C65A2970B130ABE3C479BE3E4300411887   0        0
peer-2   running   CCDF11C65A2970B130ABE3C479BE3E4300411888   0        0

mau> friend add peer-0 peer-1
✓ Friendship established

mau> friend add peer-1 peer-2
✓ Friendship established

mau> file add peer-0 test.txt
Content: This is a test file
✓ File created

mau> file watch &
# Background file watcher started

mau> # Wait for sync...
[14:40:05] peer-1: Sync complete: test.txt

mau> file cat peer-1 test.txt
This is a test file

mau> peer add peer-3
✓ Started peer peer-3

mau> friend add peer-2 peer-3
✓ Friendship established
[14:41:10] peer-3: Sync complete: test.txt (via peer-2)

mau> snapshot demo-snapshot
✓ Captured snapshot to demo-snapshot/
  - 4 peer states
  - Network topology
  - Logs

mau> exit
Environment still running. Tear down? [y/N]: y
✓ Cleaned up
```

---

### Workflow 4: Chaos Testing Demo

```bash
# Start 5-peer mesh
$ mau-e2e scenario 5-peer-mesh
✓ Started 5 peers with full mesh friendship
✓ Injected 10 test files

# Add random latency to all peers (100-300ms)
$ mau-e2e net latency --all --random 100-300
✓ Applied latency toxic to 5 peers

# Start monitoring sync
$ mau-e2e file watch &

# Kill random peer every 30s
$ mau-e2e scenario chaos-kill --interval 30s
[14:45:00] Killing peer-2...
[14:45:30] Killing peer-4...
[14:46:00] Killing peer-1...
# Sync continues via remaining peers

# Check sync health
$ mau-e2e status
Environment: mau-test-7a3f
Peers: 5 (2 running, 3 stopped)
Sync status: 8/10 files synced to all running peers
Network health: Degraded (latency toxics active)

# Restart all stopped peers
$ mau-e2e peer restart --all
✓ Restarted 3 peers

# Remove network chaos
$ mau-e2e net reset
✓ Removed all toxics

# Verify eventual consistency
$ sleep 30
$ mau-e2e file list --all
All 5 peers have 10 files ✓
```

---

## Implementation Details

### File Structure

```
e2e/
├── cmd/
│   └── mau-e2e/               # CLI tool
│       ├── main.go
│       ├── commands/
│       │   ├── up.go          # Environment creation
│       │   ├── peer.go        # Peer management
│       │   ├── friend.go      # Friend operations
│       │   ├── file.go        # File operations
│       │   ├── dht.go         # DHT commands
│       │   ├── net.go         # Network simulation
│       │   ├── scenario.go    # Predefined scenarios
│       │   └── shell.go       # Interactive shell
│       └── ui/
│           ├── table.go       # Table formatting
│           ├── progress.go    # Progress bars
│           └── colors.go      # Color output
│
├── framework/
│   ├── testenv/
│   │   ├── environment.go     # Environment lifecycle
│   │   ├── state.go           # Persistent state (for CLI)
│   │   └── api.go             # HTTP API for peer control
│   └── ...
│
├── scenarios/                 # Predefined scenarios
│   ├── 5-peer-mesh.json
│   ├── chaos-kill.json
│   └── partition-test.json
│
└── docker/
    ├── docker-compose.yml     # Base compose file
    └── docker-compose.override.example.yml
```

---

### State Persistence

The CLI needs to remember environment state between commands:

**State File:** `~/.mau-e2e/current-env.json`

```json
{
  "env_id": "mau-test-7a3f",
  "created_at": "2026-02-19T14:30:00Z",
  "network": "mau-test-7a3f",
  "bootstrap_node": "bootstrap-7a3f",
  "peers": [
    {
      "name": "alice",
      "container_id": "a3f8b2c...",
      "fingerprint": "ABAF11C65A2970B130ABE3C479BE3E4300411886",
      "port": 8001,
      "status": "running"
    },
    {
      "name": "bob",
      "container_id": "b4e9c3d...",
      "fingerprint": "BBCF11C65A2970B130ABE3C479BE3E4300411887",
      "port": 8002,
      "status": "running"
    }
  ],
  "friendships": [
    ["alice", "bob"]
  ],
  "toxiproxy": {
    "enabled": true,
    "proxies": {
      "alice": "proxy-alice-7a3f",
      "bob": "proxy-bob-7a3f"
    }
  }
}
```

**Operations:**
- `mau-e2e up` → Creates state file
- All commands read state to find containers
- `mau-e2e down` → Deletes state file
- Multiple environments: `--env <name>` flag

---

### Docker Compose Integration

**File:** `e2e/docker/docker-compose.yml`

```yaml
version: '3.8'

services:
  bootstrap:
    image: mau-e2e:latest
    container_name: bootstrap-${ENV_ID:-default}
    networks:
      - mau-test
    environment:
      - MAU_MODE=bootstrap
      - MAU_LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 5

  toxiproxy:
    image: ghcr.io/shopify/toxiproxy:latest
    container_name: toxiproxy-${ENV_ID:-default}
    networks:
      - mau-test
    ports:
      - "8474:8474"  # API
      - "8000-8100:8000-8100"  # Peer ports

networks:
  mau-test:
    name: mau-test-${ENV_ID:-default}
    driver: bridge
```

**Dynamic Peer Addition:**

The CLI uses `docker run` to add peers dynamically:

```bash
docker run -d \
  --name alice-7a3f \
  --network mau-test-7a3f \
  -e MAU_PEER_NAME=alice \
  -e MAU_BOOTSTRAP=bootstrap-7a3f:8080 \
  -l mau-e2e-env=7a3f \
  mau-e2e:latest
```

**Why not pure Docker Compose?**
- Compose requires predefining all services
- CLI needs dynamic peer count
- Hybrid approach: Compose for infra (bootstrap, toxiproxy), Docker API for peers

---

### CLI Implementation: Key Commands

#### `mau-e2e up`

```go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/mau-network/mau/e2e/framework/testenv"
)

var upCmd = &cobra.Command{
    Use:   "up",
    Short: "Start a new test environment",
    RunE: func(cmd *cobra.Command, args []string) error {
        peers, _ := cmd.Flags().GetInt("peers")
        envID := generateEnvID()
        
        // Create environment
        env := testenv.NewEnvironment(envID)
        
        // Start bootstrap node
        if err := env.StartBootstrap(); err != nil {
            return err
        }
        
        // Start Toxiproxy
        if err := env.StartToxiproxy(); err != nil {
            return err
        }
        
        // Start N peers
        for i := 0; i < peers; i++ {
            name := generatePeerName(i)
            peer, err := env.AddPeer(name)
            if err != nil {
                return err
            }
            fmt.Printf("✓ Started peer %s (fingerprint: %s)\n", 
                peer.Name, peer.Fingerprint[:8]+"...")
        }
        
        // Save state
        if err := env.SaveState(); err != nil {
            return err
        }
        
        fmt.Println("Environment ready.")
        return nil
    },
}

func init() {
    upCmd.Flags().IntP("peers", "n", 2, "Number of peers to start")
    upCmd.Flags().Bool("no-toxiproxy", false, "Disable Toxiproxy")
    upCmd.Flags().StringP("name", "N", "", "Custom environment name")
}
```

#### `mau-e2e friend add`

```go
var friendAddCmd = &cobra.Command{
    Use:   "add <peer1> <peer2>",
    Short: "Make two peers friends",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        env, err := testenv.LoadCurrentEnvironment()
        if err != nil {
            return err
        }
        
        peer1 := env.GetPeer(args[0])
        peer2 := env.GetPeer(args[1])
        
        // Exchange public keys
        fmt.Print("Exchanging public keys... ")
        key1, _ := peer1.GetPublicKey()
        key2, _ := peer2.GetPublicKey()
        
        // Add to each other's keyrings
        peer1.AddFriend(peer2.Fingerprint, key2)
        peer2.AddFriend(peer1.Fingerprint, key1)
        
        fmt.Println("✓")
        fmt.Printf("Friendship established: %s ↔ %s\n", args[0], args[1])
        
        // Update state
        env.AddFriendship(args[0], args[1])
        env.SaveState()
        
        return nil
    },
}
```

#### `mau-e2e file watch`

```go
var fileWatchCmd = &cobra.Command{
    Use:   "watch",
    Short: "Watch file sync events in real-time",
    RunE: func(cmd *cobra.Command, args []string) error {
        env, err := testenv.LoadCurrentEnvironment()
        if err != nil {
            return err
        }
        
        // Stream logs from all peers, filter for sync events
        streams := make([]io.Reader, len(env.Peers))
        for i, peer := range env.Peers {
            streams[i] = peer.StreamLogs()
        }
        
        // Multiplex streams
        combined := io.MultiReader(streams...)
        scanner := bufio.NewScanner(combined)
        
        for scanner.Scan() {
            line := scanner.Text()
            
            // Parse JSON log
            var log LogEntry
            json.Unmarshal([]byte(line), &log)
            
            // Filter for sync events
            if log.Component == "sync" {
                printSyncEvent(log)
            }
        }
        
        return nil
    },
}

func printSyncEvent(log LogEntry) {
    timestamp := log.Timestamp.Format("15:04:05")
    
    switch log.Event {
    case "file_download_started":
        fmt.Printf("[%s] %s: Downloading %s from %s...\n",
            timestamp, log.PeerID, log.File, log.SourcePeer)
    case "file_download_completed":
        fmt.Printf("[%s] %s: ✓ Sync complete: %s\n",
            timestamp, log.PeerID, log.File)
    case "file_verification_failed":
        fmt.Printf("[%s] %s: ✗ Verification failed: %s\n",
            timestamp, log.PeerID, log.File)
    }
}
```

#### `mau-e2e net partition`

```go
var netPartitionCmd = &cobra.Command{
    Use:   "partition <group1> <group2>",
    Short: "Create network partition",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        env, _ := testenv.LoadCurrentEnvironment()
        
        group1 := strings.Split(args[0], ",")
        group2 := strings.Split(args[1], ",")
        
        // For each peer in group1, block traffic to group2
        for _, p1 := range group1 {
            peer := env.GetPeer(p1)
            proxy := env.GetToxiproxy(p1)
            
            for _, p2 := range group2 {
                targetPeer := env.GetPeer(p2)
                
                // Add "timeout" toxic for traffic to p2
                proxy.AddToxic(ToxicConfig{
                    Name:       fmt.Sprintf("partition-%s-%s", p1, p2),
                    Type:       "timeout",
                    Stream:     "downstream",
                    Toxicity:   1.0,
                    Attributes: map[string]int{"timeout": 0},
                    Match:      targetPeer.IP,
                })
            }
        }
        
        fmt.Printf("✓ Created partition: {%s} vs {%s}\n", 
            strings.Join(group1, ","), strings.Join(group2, ","))
        
        env.SavePartitionState(group1, group2)
        return nil
    },
}
```

---

### Interactive Shell Implementation

**Using:** [github.com/c-bata/go-prompt](https://github.com/c-bata/go-prompt)

```go
package commands

import (
    "github.com/c-bata/go-prompt"
    "github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
    Use:   "shell",
    Short: "Interactive shell mode",
    RunE: func(cmd *cobra.Command, args []string) error {
        p := prompt.New(
            executor,
            completer,
            prompt.OptionPrefix("mau> "),
            prompt.OptionTitle("Mau E2E Shell"),
        )
        p.Run()
        return nil
    },
}

func executor(line string) {
    // Parse line as command
    args := strings.Fields(line)
    if len(args) == 0 {
        return
    }
    
    // Execute via cobra
    rootCmd.SetArgs(args)
    rootCmd.Execute()
}

func completer(d prompt.Document) []prompt.Suggest {
    s := []prompt.Suggest{
        {Text: "up", Description: "Start environment"},
        {Text: "down", Description: "Tear down environment"},
        {Text: "peer", Description: "Peer management"},
        {Text: "friend", Description: "Friend operations"},
        {Text: "file", Description: "File operations"},
        {Text: "net", Description: "Network simulation"},
        {Text: "status", Description: "Show status"},
        {Text: "exit", Description: "Exit shell"},
    }
    return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
```

---

### Predefined Scenarios

**File:** `e2e/scenarios/5-peer-mesh.json`

```json
{
  "name": "5-peer-mesh",
  "description": "5 peers in full mesh topology",
  "steps": [
    {
      "action": "create_peers",
      "count": 5,
      "names": ["alice", "bob", "carol", "dave", "eve"]
    },
    {
      "action": "full_mesh_friends",
      "peers": ["alice", "bob", "carol", "dave", "eve"]
    },
    {
      "action": "inject_files",
      "peer": "alice",
      "files": [
        {"name": "file1.txt", "content": "Test file 1"},
        {"name": "file2.txt", "content": "Test file 2"}
      ]
    },
    {
      "action": "wait",
      "duration": "30s",
      "condition": "all_files_synced"
    }
  ]
}
```

**Usage:**

```bash
$ mau-e2e scenario 5-peer-mesh
✓ Created 5 peers
✓ Established 10 friendships (full mesh)
✓ Injected 2 files into alice
⏳ Waiting for sync (max 30s)...
✓ All peers have 2 files
Scenario complete in 12.3s
```

---

## UI/UX Enhancements

### Progress Bars

```bash
$ mau-e2e file add alice large-file.bin < /dev/urandom | head -c 100M
Encrypting file... ████████████████████ 100% (100 MB)
Uploading to alice... ████████████████████ 100% (102 MB encrypted)
✓ File added: large-file.bin
```

### Real-time Tables

```bash
$ mau-e2e status --watch
Environment: mau-test-7a3f                         Uptime: 5m 23s

PEERS (5)
NAME     STATUS    FILES   FRIENDS   DHT PEERS   CPU    MEMORY
alice    running   10/10   4         4           2.1%   45 MB
bob      running   10/10   4         4           1.8%   43 MB
carol    running   8/10    4         4           3.2%   48 MB  ⚠ Syncing
dave     stopped   -       -         -           -      -      ✗
eve      running   10/10   4         3           1.5%   42 MB

NETWORK
Toxiproxy: enabled
Active Toxics: latency (alice: 120ms, bob: 98ms)

RECENT ACTIVITY (last 30s)
14:52:10  carol   Downloaded file8.txt from alice
14:52:15  carol   Downloaded file9.txt from bob
14:52:18  dave    Peer stopped (manual)

Press Ctrl-C to exit watch mode
```

### Color-Coded Output

```bash
$ mau-e2e file list alice
Files on alice (10):
  ✓ file1.txt        (synced)       1.2 KB
  ✓ file2.txt        (synced)       3.4 KB
  ⏳ file3.txt       (syncing)      5.6 KB  [████░░░░] 50%
  ✗ file4.txt        (failed)       2.1 KB  Signature invalid
  ✓ file5.txt        (synced)       8.9 KB
```

---

## Integration with Automated Tests

**Shared Testenv Library:**

```go
// e2e/framework/testenv/environment.go

type Environment struct {
    ID       string
    Network  *DockerNetwork
    Peers    []*MauPeer
    state    *State
}

// Used by CLI
func NewEnvironment(id string) *Environment {
    return &Environment{
        ID:    id,
        state: NewState(id),
    }
}

// Used by go test
func NewTestEnvironment(t *testing.T) *Environment {
    id := fmt.Sprintf("test-%s", t.Name())
    env := NewEnvironment(id)
    
    // Auto-cleanup on test end
    t.Cleanup(func() {
        env.Cleanup()
    })
    
    return env
}
```

**Example Test Using CLI-Compatible Env:**

```go
func TestFileSync(t *testing.T) {
    env := testenv.NewTestEnvironment(t)
    
    // Same API as CLI uses
    alice, _ := env.AddPeer("alice")
    bob, _ := env.AddPeer("bob")
    env.MakeFriends(alice, bob)
    
    alice.AddFile("test.txt", strings.NewReader("content"))
    
    // Automated assertion
    assert.Eventually(t, func() bool {
        return bob.HasFile("test.txt")
    }, 30*time.Second, 1*time.Second)
}
```

**Benefit:** Tests can be converted to CLI scenarios and vice versa.

---

## Documentation Structure

### Quick Start Guide

**File:** `e2e/README.md`

```markdown
# Mau E2E Framework

## Quick Start

### Interactive Mode (Recommended for exploration)

```bash
# Install CLI
cd e2e
go install ./cmd/mau-e2e

# Start environment
mau-e2e up --peers 3

# Make friends
mau-e2e friend add peer-0 peer-1

# Add file
mau-e2e file add peer-0 test.txt
# (Enter content, Ctrl-D to finish)

# Watch sync
mau-e2e file watch

# Inspect state
mau-e2e peer inspect peer-1

# Clean up
mau-e2e down
```

### Automated Testing Mode

```bash
# Run all tests
make test-e2e

# Run specific suite
go test ./scenarios/basic/...

# Run in CI
make test-e2e-ci
```

## Learn More

- [CLI Design](CLI_DESIGN.md) - Interactive CLI details
- [Test Plan](PLAN.md) - Automated testing framework
- [Writing Tests](docs/writing-tests.md) - Add new test scenarios
- [Debugging](docs/debugging.md) - Troubleshoot failures
```

---

## Implementation Roadmap

### Phase 1: CLI Foundation (Week 1)
- [ ] Basic `mau-e2e` CLI structure (cobra)
- [ ] `up` command (start environment)
- [ ] `down` command (cleanup)
- [ ] `peer add/list` commands
- [ ] State persistence (`~/.mau-e2e/`)
- [ ] Docker Compose base setup

**Deliverable:** Can start/stop environment with N peers

---

### Phase 2: Peer Interaction (Week 2)
- [ ] `friend add/list` commands
- [ ] `file add/list/cat` commands
- [ ] Peer inspection (`peer inspect`)
- [ ] Log streaming (`logs` command)

**Deliverable:** Can manually test 2-peer sync

---

### Phase 3: Real-time Monitoring (Week 3)
- [ ] `file watch` command (real-time sync events)
- [ ] `status --watch` command (live dashboard)
- [ ] Color-coded output
- [ ] Progress bars for long operations

**Deliverable:** Can observe sync happening live

---

### Phase 4: Network Simulation (Week 4)
- [ ] Toxiproxy integration
- [ ] `net partition/heal` commands
- [ ] `net latency/limit` commands
- [ ] Partition state management

**Deliverable:** Can create network partitions interactively

---

### Phase 5: Advanced Features (Week 5)
- [ ] Interactive shell mode
- [ ] Predefined scenarios (`scenario` command)
- [ ] Snapshot/restore functionality
- [ ] DHT commands (`dht lookup/table`)

**Deliverable:** Full-featured interactive environment

---

### Phase 6: Integration & Polish (Week 6)
- [ ] Integrate with automated tests (shared `testenv`)
- [ ] Documentation (guides, examples)
- [ ] Video tutorial (screencast)
- [ ] CI integration (ensure compatibility)

**Deliverable:** Production-ready CLI + tests

---

## Success Criteria

After implementation:

✅ **Emad can spin up 5 Mau instances with one command**  
✅ **Can manually trigger friend relationships via CLI**  
✅ **Can inject files and watch them sync in real-time**  
✅ **Can inspect any peer's state at any time**  
✅ **Can simulate network failures interactively**  
✅ **Can capture snapshots for later analysis**  
✅ **Same environment code powers both CLI and automated tests**  
✅ **New developers can explore Mau P2P behavior without reading code**  

---

## Example Demo Script (For Video Tutorial)

```bash
# Demo: Mau P2P File Sync in Action

# 1. Start environment
$ mau-e2e up --peers 3
# (Shows progress bars, peer fingerprints)

# 2. Enter shell mode
$ mau-e2e shell

mau> peer list
# (Shows 3 peers, all running)

# 3. Create friend triangle: A-B, B-C, C-A
mau> friend add peer-0 peer-1
mau> friend add peer-1 peer-2
mau> friend add peer-2 peer-0

# 4. Start watching sync events
mau> file watch &

# 5. Add file to peer-0
mau> file add peer-0 secret.txt
Content (Ctrl-D to finish):
This is a secret message!
^D
# (Watch shows sync events appearing)

# 6. Verify all peers have it
mau> file list --all
# (Table shows all 3 peers have secret.txt)

# 7. Create network partition
mau> net partition peer-0 peer-1,peer-2
# (Watch shows sync stops)

# 8. Add files on both sides
mau> file add peer-0 isolated.txt <<< "Only peer-0 sees this"
mau> file add peer-1 shared.txt <<< "Peers 1 and 2 see this"
# (Watch confirms no cross-partition sync)

# 9. Heal partition
mau> net heal
# (Watch shows files flooding across)

# 10. Verify eventual consistency
mau> sleep 10
mau> file list --all
# (All peers have all files)

mau> snapshot demo-run
mau> exit
```

**Demo Duration:** 3-4 minutes  
**Impact:** Developers **see** P2P sync working, builds intuition for distributed systems

---

## Conclusion

The interactive CLI transforms the E2E framework from **"test infrastructure"** into **"P2P development playground."**

**Key Benefits:**
1. **Exploratory Testing:** Manually probe edge cases
2. **Debugging:** Inspect live peer state when tests fail
3. **Demonstrations:** Show Mau P2P behavior to stakeholders
4. **Education:** New contributors learn by doing
5. **Scenario Development:** Prototype test scenarios interactively before automating

**Next Steps:**
1. Review this CLI design
2. Approve tech stack (Cobra, go-prompt)
3. Implement Phase 1 (CLI foundation)
4. Iterate based on real usage

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-19  
**Status:** Awaiting Approval
