#!/usr/bin/env bash
# Mau Development Server Orchestrator
# Ensures .env exists before starting Vite, then runs both servers

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TYPESCRIPT_DIR="$(cd "$SCRIPT_DIR/../typescript" && pwd)"
SERVER_SCRIPT="$TYPESCRIPT_DIR/bootstrap-server.mjs"
PORT=8444
ENV_FILE="$SCRIPT_DIR/.env"

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${BLUE}🚀 Starting Mau Development Environment...${NC}\n"

# Build TypeScript if needed
if [ ! -d "$TYPESCRIPT_DIR/dist" ]; then
	echo "📦 Building TypeScript..."
	cd "$TYPESCRIPT_DIR"
	npm run build >/dev/null 2>&1
fi

# Start bootstrap server in background
cd "$TYPESCRIPT_DIR"
node "$SERVER_SCRIPT" --port "$PORT" >/tmp/bootstrap-server.log 2>&1 &
SERVER_PID=$!

# Wait for .env to be created
echo "⏳ Waiting for bootstrap server to create .env..."
for i in {1..20}; do
	if [ -f "$ENV_FILE" ]; then
		echo -e "${GREEN}✅ Bootstrap server ready${NC}"
		cat "$ENV_FILE"
		echo ""
		break
	fi
	sleep 0.5
done

if [ ! -f "$ENV_FILE" ]; then
	echo "❌ Bootstrap server failed to create .env"
	kill $SERVER_PID 2>/dev/null || true
	exit 1
fi

# Start Vite dev server
cd "$SCRIPT_DIR"
echo "🎨 Starting Vite..."
bun run vite &
VITE_PID=$!

# Cleanup on exit
cleanup() {
	echo "\n\n🛑 Shutting down..."
	kill $SERVER_PID 2>/dev/null || true
	kill $VITE_PID 2>/dev/null || true
	exit 0
}

trap cleanup SIGINT SIGTERM

# Wait for both processes
wait
