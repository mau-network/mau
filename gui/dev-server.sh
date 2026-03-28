#!/usr/bin/env bash
# Mau Development Bootstrap Server (Node.js)
# This script starts the Node.js WebRTC bootstrap server for development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TYPESCRIPT_DIR="$(cd "$SCRIPT_DIR/../typescript" && pwd)"
SERVER_SCRIPT="$TYPESCRIPT_DIR/bootstrap-server.mjs"
PORT=8444

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting Mau Development Bootstrap Peer (Node.js)...${NC}\n"

# Check if TypeScript is built
if [ ! -d "$TYPESCRIPT_DIR/dist" ]; then
	echo -e "${YELLOW}📦 TypeScript library not built, building now...${NC}"
	cd "$TYPESCRIPT_DIR"
	npm run build
	echo ""
fi

# Check if Node.js is available
if ! command -v node &>/dev/null; then
	echo -e "${YELLOW}❌ Node.js not found. Please install Node.js ≥18${NC}"
	exit 1
fi

# Check Node.js version
NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
	echo -e "${YELLOW}❌ Node.js version 18 or higher required (found: $(node --version))${NC}"
	exit 1
fi

# Start the bootstrap server
cd "$TYPESCRIPT_DIR"
exec node "$SERVER_SCRIPT" --port "$PORT"
