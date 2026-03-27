#!/usr/bin/env bash
# Development Bootstrap Peer
#
# Starts a Mau Go server that acts as a bootstrap peer for the GUI during development.
# This server exposes /p2p/<fingerprint> endpoints and handles DHT bootstrap via /p2p/dht/offer.

set -e

PEER_DIR=".dev-peer"
PEER_BIN=".dev-peer-bin"
PASSPHRASE="dev-server-pass-12345"
PORT="8081"

echo "🚀 Starting Mau Development Bootstrap Peer..."
echo ""

# Build the mau binary if it doesn't exist
if [ ! -f "$PEER_BIN" ]; then
	echo "📦 Building mau CLI..."
	cd .. && go build -o gui/$PEER_BIN ./cmd/mau && cd gui
	echo "✅ Built mau CLI"
	echo ""
fi

# Create account if it doesn't exist
if [ ! -d "$PEER_DIR" ]; then
	echo "🔑 Creating new peer account..."
	mkdir -p "$PEER_DIR"
	cd "$PEER_DIR"
	../$PEER_BIN init -name "Dev Bootstrap Peer" -email "dev@mau.local" -passphrase "$PASSPHRASE"
	cd ..
	echo "✅ Created peer account"
	echo ""
fi

# Get fingerprint
cd "$PEER_DIR"
FINGERPRINT=$(../$PEER_BIN show -passphrase "$PASSPHRASE" | grep "Fingerprint:" | awk '{print $2}')
cd ..

echo "✨ Development bootstrap peer starting!"
echo ""
echo "📌 Fingerprint: $FINGERPRINT"
echo ""
echo "📍 Endpoints:"
echo "   http://localhost:$PORT/p2p/$FINGERPRINT - File list"
echo "   POST http://localhost:$PORT/p2p/dht/offer - DHT bootstrap"
echo ""
echo "🔗 To configure the GUI, add this to .env:"
echo "   VITE_DEV_PEER_FINGERPRINT=$FINGERPRINT"
echo "   VITE_DEV_PEER_ADDRESS=localhost:$PORT"
echo ""
echo "💡 The GUI will automatically use this peer for DHT bootstrap"
echo "⏳ Starting server..."
echo ""

# Start the server (runs in the account directory, which is PEER_DIR)
cd "$PEER_DIR"
exec ../$PEER_BIN serve -passphrase "$PASSPHRASE" -port "$PORT"
