#!/bin/bash
# Remove console.log/warn from production code (keep console.error)

# webrtc-server.ts
sed -i '/console.log.*WebRTCServer.*Data channel opened/d' src/network/webrtc-server.ts
sed -i '/console.log.*WebRTCServer.*Data channel closed/d' src/network/webrtc-server.ts
sed -i '/console.log.*WebRTCServer.*mTLS authenticated/d' src/network/webrtc-server.ts
sed -i 's|console.warn(\`\[WebRTCServer\] Unknown message type|// Unknown message type|' src/network/webrtc-server.ts

# webrtc.ts
sed -i "/console.log('Data channel opened');/d" src/network/webrtc.ts
sed -i "/console.log('Data channel closed');/d" src/network/webrtc.ts

# signaling.ts
sed -i "/console.log('\[Signaling\] Disconnected');/d" src/network/signaling.ts

# resolvers.ts - Keep console.warn for browser warning (it's important)

echo "Removed console.log/warn statements"
