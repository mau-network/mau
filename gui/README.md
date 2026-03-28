# Mau GUI

Browser-based GUI for the Mau P2P network. Built with React, TypeScript, and the `@mau-network/mau` library.

## Features

- **Account Management**: Create PGP-based accounts with encrypted storage
- **Friend Management**: Add/remove friends using PGP public keys
- **Status Updates**: Post and view encrypted status messages
- **WebRTC P2P**: Automatic peer-to-peer connectivity via Kademlia DHT
- **Offline-First**: All data stored locally in IndexedDB

## Quick Start

### Start Development Server

The easiest way to get started:

```bash
# Install dependencies
bun install

# Start both bootstrap peer and GUI
npm run dev
```

This automatically:
1. Starts Node.js bootstrap peer (WebRTC server)
2. Starts GUI dev server
3. Opens browser at http://localhost:5173

The bootstrap peer uses the TypeScript library and runs in Node.js - no Go compilation needed!

### Manual Setup (Advanced)

If you want to run components separately:

**Terminal 1: Bootstrap Peer**
```bash
cd ../typescript
node bootstrap-server.mjs --port 8444
```

**Terminal 2: GUI**
```bash
cd gui
bun run dev:gui
```

## Architecture

### P2P Connectivity

The GUI uses **Node.js WebRTC bootstrap server** (pure TypeScript) for DHT network join:

1. **Bootstrap**: Browser connects to Node.js server via WebSocket signaling
2. **WebRTC**: Establishes data channel with server
3. **DHT**: Server adds browser to DHT routing table
4. **Discovery**: Browser queries server for other peers
5. **Relay**: Browser connects to other peers via DHT relay signaling

```
Browser A ──┐
            ├──> WebSocket ──> Node.js Bootstrap Server
Browser B ──┘                  (TypeScript Library)
                               ├─ WebRTCServer
                               ├─ KademliaDHT
                               └─ Tracks all peers
```

**Why Node.js?** Instead of implementing WebRTC in Go, we reuse the existing TypeScript library. The bootstrap server is literally just another Mau peer running in Node.js!

### Components

- **AccountManager** (`src/account/manager.ts`): Account lifecycle and friend management
- **ConnectionManager** (`src/network/connection-manager.ts`): WebRTC + DHT networking
- **StatusStoreManager** (`src/status/store.ts`): Encrypted status post storage
- **UI Components** (`src/ui/`): React components for auth, timeline, friends

### Network Configuration

Bootstrap peers are configured in `src/config/network.ts`:

- **Development**: Automatically configured by `npm run dev`
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
3. Starts two servers:
   - **HTTPS Server** (port 8443): mTLS file serving for Go clients
   - **WebSocket Signaling** (port 8444): Browser-compatible signaling
4. Exposes:
   - `GET /p2p/{fingerprint}` - File list endpoint (HTTPS, mTLS)
   - `WS ws://localhost:8444` - WebSocket signaling for DHT bootstrap

**Why two servers?**
- Go clients use HTTPS with mTLS for authenticated connections
- Browser clients use WebSocket for signaling (browsers can't do mTLS programmatically)
- After bootstrap, all clients use WebRTC data channels for P2P communication

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
VITE_DEV_PEER_FINGERPRINT=<fingerprint>      # Dev bootstrap peer fingerprint
VITE_DEV_PEER_WS_ADDRESS=localhost:8444      # WebSocket signaling URL
```

See `.env.example` for the complete list.

## Production

### Building

```bash
bun run build
```

Output will be in `dist/` directory.

### Bootstrap Peers

For production, configure public bootstrap peers in `src/config/network.ts` using WebSocket URLs:

```typescript
const PRODUCTION_BOOTSTRAP_PEERS: Peer[] = [
  { fingerprint: 'abc123...', address: 'wss://bootstrap.mau.network:443' }
];
```

**Important:** Use `wss://` (WebSocket Secure) for production to ensure encrypted signaling.

```typescript
const PRODUCTION_BOOTSTRAP_PEERS: Peer[] = [
  { fingerprint: 'abc123...', address: 'bootstrap.mau.network:443' }
];
```

## Troubleshooting

### Bootstrap peer won't start

- **Ports in use**: Change `HTTPS_PORT` or `WS_PORT` in `dev-server.sh` (defaults: 8443, 8444)
- **Build fails**: Run `go build -o gui/.dev-peer-bin ./cmd/mau` from repo root
- **Permission denied**: Run `chmod +x dev-server.sh`

### GUI can't connect to DHT

- Check `.env` has correct fingerprint and WebSocket address from dev peer output
- Verify dev peer is running (`bun run dev:peer`)
- Check browser console for WebSocket connection logs
- Look for: `[DHT] WebSocket connected, registering fingerprint...`
- **WebSocket connection failed**: Ensure port 8444 is not blocked by firewall
- **CORS errors**: The signaling server allows all origins in development

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
