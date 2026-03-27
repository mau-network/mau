# Mau GUI

Browser-based GUI for the Mau P2P network. Built with React, TypeScript, and the `@mau-network/mau` library.

## Features

- **Account Management**: Create PGP-based accounts with encrypted storage
- **Friend Management**: Add/remove friends using PGP public keys
- **Status Updates**: Post and view encrypted status messages
- **WebRTC P2P**: Automatic peer-to-peer connectivity via Kademlia DHT
- **Offline-First**: All data stored locally in IndexedDB

## Quick Start

### 1. Install Dependencies

```bash
bun install
```

### 2. Start Development Bootstrap Peer

The GUI needs a bootstrap peer to join the DHT network. For development, run a local Go server:

```bash
# Terminal 1: Start the bootstrap peer
bun run dev:peer
```

The script will output a fingerprint like:
```
📌 Fingerprint: 1e7017fb8ce8865504136f718f508e19906bf729
```

### 3. Configure Bootstrap Peer

Copy the fingerprint and add it to `.env` (create from `.env.example`):

```bash
cp .env.example .env
# Edit .env and set:
VITE_DEV_PEER_FINGERPRINT=1e7017fb8ce8865504136f718f508e19906bf729
VITE_DEV_PEER_ADDRESS=localhost:8081
```

### 4. Start GUI Development Server

```bash
# Terminal 2: Start the GUI
bun run dev:gui

# Or run both together:
bun run dev
```

The GUI will be available at `http://localhost:5173`

## Architecture

### P2P Connectivity

The GUI uses a **pure DHT architecture** with no centralized signaling server:

1. **Bootstrap**: First connection via HTTP POST to bootstrap peer's `/p2p/dht/offer`
2. **Relay**: Subsequent connections use DHT relay signaling through existing peers
3. **WebRTC**: Peer-to-peer data channels for file sharing and communication

```
Browser (GUI) → HTTP Bootstrap → Go Server (dev peer)
              → WebRTC DHT Join
              → Relay Signaling → Other Peers
```

### Components

- **AccountManager** (`src/account/manager.ts`): Account lifecycle and friend management
- **ConnectionManager** (`src/network/connection-manager.ts`): WebRTC + DHT networking
- **StatusStoreManager** (`src/status/store.ts`): Encrypted status post storage
- **UI Components** (`src/ui/`): React components for auth, timeline, friends

### Network Configuration

Bootstrap peers are configured in `src/config/network.ts`:

- **Development**: Uses local Go server via environment variables
- **Production**: Configurable public bootstrap peers

## Development

### Available Scripts

```bash
# Development
bun run dev          # Run bootstrap peer + GUI concurrently
bun run dev:peer     # Run bootstrap peer only (port 8081)
bun run dev:gui      # Run GUI only (port 5173)

# Testing
bun test             # Run unit tests
bun run test:e2e     # Run E2E tests (requires GUI running)

# Code Quality
bun run lint         # Run ESLint
bun run type-check   # Run TypeScript compiler

# Build
bun run build        # Build for production
```

### Project Structure

```
gui/
├── src/
│   ├── account/          # Account management
│   ├── config/           # Configuration (network, etc.)
│   ├── network/          # WebRTC + DHT connectivity
│   ├── status/           # Status post storage
│   ├── types/            # TypeScript types
│   ├── ui/               # React components
│   ├── utils/            # Utilities
│   ├── app.tsx           # Main app component
│   ├── app-state.ts      # Global app state
│   └── main.tsx          # Entry point
├── tests/                # E2E tests
├── dev-server.sh         # Development bootstrap peer script
├── .env                  # Environment variables (git-ignored)
├── .env.example          # Environment template
└── vite.config.ts        # Vite configuration
```

### Bootstrap Peer Setup

The development bootstrap peer (`dev-server.sh`) does the following:

1. Builds the Go mau CLI binary (if not exists)
2. Creates a dev account in `.dev-peer/` (if not exists)
3. Starts the server on port 8081
4. Exposes:
   - `GET /p2p/{fingerprint}` - File list endpoint
   - `POST /p2p/dht/offer` - DHT bootstrap endpoint

The peer data (`.dev-peer/` directory and `.dev-peer-bin` binary) are git-ignored.

### Testing

```bash
# Unit tests
bun test

# E2E tests (requires GUI running)
bun run dev:gui        # Terminal 1
bun run test:e2e       # Terminal 2
```

### Environment Variables

All environment variables must be prefixed with `VITE_` to be exposed to the browser:

```bash
VITE_DEV_PEER_FINGERPRINT=<fingerprint>  # Dev bootstrap peer fingerprint
VITE_DEV_PEER_ADDRESS=localhost:8081     # Dev bootstrap peer address
```

See `.env.example` for the complete list.

## Production

### Building

```bash
bun run build
```

Output will be in `dist/` directory.

### Bootstrap Peers

For production, configure public bootstrap peers in `src/config/network.ts`:

```typescript
const PRODUCTION_BOOTSTRAP_PEERS: Peer[] = [
  { fingerprint: 'abc123...', address: 'bootstrap.mau.network:443' }
];
```

## Troubleshooting

### Bootstrap peer won't start

- **Port 8081 in use**: Change `PORT` in `dev-server.sh`
- **Build fails**: Run `go build -o gui/.dev-peer-bin ./cmd/mau` from repo root

### GUI can't connect to DHT

- Check `.env` has correct fingerprint from dev peer output
- Verify dev peer is running (`bun run dev:peer`)
- Check browser console for connection logs

### Tests fail

Some test failures are expected:
- Status store tests: Known issue with encryption
- E2E tests: Require proper setup with `bun run test:e2e`

## Technology Stack

- **Runtime**: Bun 1.3+
- **Framework**: React 18 + TypeScript
- **Build**: Vite
- **UI**: Ant Design
- **Storage**: IndexedDB (via `@mau-network/mau`)
- **Crypto**: PGP (OpenPGP.js)
- **P2P**: WebRTC + Kademlia DHT

## Contributing

See the main project [README](../README.md) and [AGENTS.md](../AGENTS.md) for development guidelines.

## License

See the main project [LICENSE](../LICENSE) file.
