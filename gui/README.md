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
# Install dependencies (from project root)
npm install

# Start both bootstrap peer and GUI
npm run dev -w gui
```

This automatically:
1. Builds TypeScript library if needed
2. Starts Node.js bootstrap peer on port 8444
3. Starts Vite dev server on http://localhost:3000
4. Runs both concurrently with colored output

The bootstrap peer uses the TypeScript library and runs in Node.js - no Go compilation needed!

### Manual Setup (Advanced)

If you want to run components separately:

**Terminal 1: Bootstrap Peer**
```bash
npm run dev:peer -w gui
```

**Terminal 2: GUI**
```bash
npm run dev:gui -w gui
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
npm run dev -w gui          # Run bootstrap peer + GUI concurrently
npm run dev:peer -w gui     # Run bootstrap peer only (port 8444)
npm run dev:gui -w gui      # Run GUI only (port 3000)

# Building
npm run build -w gui        # Build GUI for production
npm run build:ts -w gui     # Build TypeScript library only

# Testing
npm run test:e2e -w gui     # Run E2E tests (Playwright)

# Code Quality
npm run lint -w gui         # Run ESLint
npm run typecheck -w gui    # Run TypeScript compiler
npm run format -w gui       # Format with Prettier

# Production
npm run build -w gui        # Build for production
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
├── .env                  # Environment variables (git-ignored)
├── .env.example          # Environment template
└── vite.config.ts        # Vite configuration
```

### Bootstrap Peer Setup

The development bootstrap peer (Node.js) provides:

1. **WebSocket Signaling** (port 8444): Browser-compatible signaling for DHT bootstrap
2. **WebRTC Data Channels**: Peer-to-peer communication after connection
3. **Auto-configuration**: Creates `.env` file automatically with peer fingerprint

The bootstrap server is implemented using the TypeScript library (`../typescript/bootstrap-server.mjs`).

### Testing

Unit tests use bun:test runtime and require bun to run. E2E tests use Playwright and work with npm.

```bash
# E2E tests (Playwright - works with npm)
npm run dev:gui -w gui    # Terminal 1
npm run test:e2e -w gui   # Terminal 2

# Unit tests (requires bun)
bun test
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
npm run build -w gui
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

- **Build fails**: Ensure TypeScript library is built: `npm run build:ts -w gui`
- **Port in use**: Change port in package.json or stop other services on port 8444

### GUI can't connect to DHT

- Check `.env` has correct fingerprint and WebSocket address
- Verify dev peer is running: `npm run dev:peer -w gui`
- Check browser console for WebSocket connection logs
- Look for: `[DHT] WebSocket connected, registering fingerprint...`
- **WebSocket connection failed**: Ensure port 8444 is not blocked by firewall
- **CORS errors**: The signaling server allows all origins in development

### Tests fail

- Unit tests require bun runtime (use `bun test` instead of npm)
- E2E tests require GUI running: `npm run dev:gui -w gui` first

## Technology Stack

- **Package Manager**: npm (workspace)
- **Runtime**: Node.js 18+ (server), Browser (GUI)
- **Framework**: React 19 + TypeScript
- **Build**: Vite 8
- **UI**: Ant Design 6
- **Storage**: IndexedDB (via `@mau-network/mau`)
- **Crypto**: PGP (OpenPGP.js)
- **P2P**: WebRTC + Kademlia DHT
- **Testing**: Playwright (E2E), bun:test (unit)

## Contributing

See the main project [README](../README.md) and [AGENTS.md](../AGENTS.md) for development guidelines.

## License

See the main project [LICENSE](../LICENSE) file.
