#!/usr/bin/env bash
# Manual verification script for DHT peer connections
# This script helps verify that peers actually connect to each other

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     DHT PEER CONNECTION MANUAL VERIFICATION           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${YELLOW}This script will:${NC}"
echo "  1. Start the bootstrap server"
echo "  2. Guide you through testing with multiple browser tabs"
echo "  3. Verify peer connections are established"
echo ""
echo -e "${YELLOW}Prerequisites:${NC}"
echo "  - Node.js ≥18"
echo "  - npm dependencies installed"
echo "  - Ports 8444 (bootstrap) and 5173 (GUI) available"
echo ""

read -p "Press ENTER to continue or Ctrl+C to abort..."

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  STEP 1: Build TypeScript library${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

cd "$(dirname "$0")/../typescript"
npm run build
echo -e "${GREEN}✅ TypeScript library built${NC}"

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  STEP 2: Start development servers${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

cd ../gui

echo -e "${YELLOW}Starting bootstrap server and GUI dev server...${NC}"
echo -e "${YELLOW}This will open in a new terminal. Keep it running.${NC}"
echo ""

# Start in background and capture PIDs
npm run dev &
DEV_PID=$!

echo -e "${GREEN}✅ Servers starting (PID: $DEV_PID)${NC}"
echo ""
echo -e "${YELLOW}Waiting 5 seconds for servers to start...${NC}"
sleep 5

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  STEP 3: Open browser tabs${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${YELLOW}📝 Manual steps:${NC}"
echo ""
echo "  1. Open THREE browser tabs to: ${GREEN}http://localhost:5173${NC}"
echo ""
echo "  2. In Tab 1 (Alice):"
echo "     - Create account: Alice / alice@test.com / password123456"
echo "     - Wait 3 seconds"
echo ""
echo "  3. In Tab 2 (Bob):"
echo "     - Create account: Bob / bob@test.com / password234567"
echo "     - Wait 3 seconds"
echo ""
echo "  4. In Tab 3 (Charlie):"
echo "     - Create account: Charlie / charlie@test.com / password345678"
echo ""
echo "  5. Wait 20 seconds for DHT discovery"
echo ""
echo "  6. Open browser DevTools console in each tab (F12)"
echo ""
echo "  7. In EACH tab's console, run:"
echo -e "     ${GREEN}window.testGetDHTState()${NC}"
echo ""
echo "  8. Check the output:"
echo "     - ${GREEN}isActive: true${NC} (DHT is running)"
echo "     - ${GREEN}peerCount: <number>${NC} (should be > 0)"
echo "     - ${GREEN}connectedPeers: [...]${NC} (list of peer fingerprints)"
echo ""

read -p "Press ENTER when you've completed the steps above..."

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  STEP 4: Verification checklist${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

echo -e "${YELLOW}Answer the following questions:${NC}"
echo ""

read -p "Did Alice's console show peerCount > 0? (y/n): " alice_peers
read -p "Did Bob's console show peerCount > 0? (y/n): " bob_peers
read -p "Did Charlie's console show peerCount > 0? (y/n): " charlie_peers
read -p "Did you see 'Bootstrap discovery' logs in any console? (y/n): " discovery_logs
read -p "Did you see 'relay' related logs in any console? (y/n): " relay_logs

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  VERIFICATION RESULTS${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

PASSED=true

if [[ "$alice_peers" == "y" ]]; then
	echo -e "${GREEN}✅ Alice has peer connections${NC}"
else
	echo -e "${RED}❌ Alice has NO peer connections${NC}"
	PASSED=false
fi

if [[ "$bob_peers" == "y" ]]; then
	echo -e "${GREEN}✅ Bob has peer connections${NC}"
else
	echo -e "${RED}❌ Bob has NO peer connections${NC}"
	PASSED=false
fi

if [[ "$charlie_peers" == "y" ]]; then
	echo -e "${GREEN}✅ Charlie has peer connections${NC}"
else
	echo -e "${RED}❌ Charlie has NO peer connections${NC}"
	PASSED=false
fi

if [[ "$discovery_logs" == "y" ]]; then
	echo -e "${GREEN}✅ DHT discovery is running${NC}"
else
	echo -e "${RED}❌ DHT discovery NOT detected${NC}"
	PASSED=false
fi

if [[ "$relay_logs" == "y" ]]; then
	echo -e "${GREEN}✅ Relay signaling attempted${NC}"
else
	echo -e "${YELLOW}⚠️  No relay signaling logs (may be expected if using bootstrap only)${NC}"
fi

echo ""

if [[ "$PASSED" == "true" ]]; then
	echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
	echo -e "${GREEN}║              ✅ VERIFICATION PASSED                    ║${NC}"
	echo -e "${GREEN}║                                                        ║${NC}"
	echo -e "${GREEN}║  Peers are successfully connecting through DHT!       ║${NC}"
	echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
else
	echo -e "${RED}╔════════════════════════════════════════════════════════╗${NC}"
	echo -e "${RED}║              ❌ VERIFICATION FAILED                    ║${NC}"
	echo -e "${RED}║                                                        ║${NC}"
	echo -e "${RED}║  Some peers failed to establish connections           ║${NC}"
	echo -e "${RED}╚════════════════════════════════════════════════════════╝${NC}"
	echo ""
	echo -e "${YELLOW}Debugging steps:${NC}"
	echo "  1. Check bootstrap server logs for 'Peer registered in DHT'"
	echo "  2. Verify browser console shows '[Bootstrap] Peer connected'"
	echo "  3. Check for JavaScript errors in browser DevTools"
	echo "  4. Ensure WebSocket connection to ws://localhost:8444 succeeds"
	echo "  5. Run automated tests: npm run test:e2e"
fi

echo ""
echo -e "${YELLOW}Cleaning up...${NC}"
kill $DEV_PID 2>/dev/null || true
echo -e "${GREEN}✅ Servers stopped${NC}"
echo ""
