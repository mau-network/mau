#!/usr/bin/env bash
# Mau Development Bootstrap Server with .env creation
# This script ensures .env is created before returning

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TYPESCRIPT_DIR="$(cd "$SCRIPT_DIR/../typescript" && pwd)"
SERVER_SCRIPT="$TYPESCRIPT_DIR/bootstrap-server.mjs"
PORT=8444
ENV_FILE="$SCRIPT_DIR/.env"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🚀 Starting Mau Development Bootstrap Peer...${NC}\n"

# Check if TypeScript is built
if [ ! -d "$TYPESCRIPT_DIR/dist" ]; then
	echo -e "${YELLOW}📦 Building TypeScript library...${NC}"
	cd "$TYPESCRIPT_DIR"
	npm run build
	echo ""
fi

# Start bootstrap server in background
cd "$TYPESCRIPT_DIR"
node "$SERVER_SCRIPT" --port "$PORT" &
SERVER_PID=$!

# Wait for .env file to be created (timeout after 10 seconds)
echo -e "${BLUE}⏳ Waiting for .env file...${NC}"
TIMEOUT=10
ELAPSED=0
while [ ! -f "$ENV_FILE" ] && [ $ELAPSED -lt $TIMEOUT ]; do
	sleep 0.5
	ELAPSED=$((ELAPSED + 1))
done

if [ -f "$ENV_FILE" ]; then
	echo -e "${GREEN}✅ .env file created${NC}"
	cat "$ENV_FILE"
	echo ""

	# Keep server running in foreground
	wait $SERVER_PID
else
	echo -e "${YELLOW}❌ Timeout waiting for .env file${NC}"
	kill $SERVER_PID 2>/dev/null || true
	exit 1
fi
