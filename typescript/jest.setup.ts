/**
 * Test Environment Setup
 * - WebRTC Polyfill for Node.js using @roamhq/wrtc (fallback from node-datachannel)
 * - IndexedDB Polyfill for Node.js using fake-indexeddb
 */

import 'fake-indexeddb/auto';

// Polyfill RTCPeerConnection for Node.js environment
if (typeof global.RTCPeerConnection === 'undefined') {
  try {
    // Try @roamhq/wrtc (browser-compatible API)
    const wrtc = require('@roamhq/wrtc');
    (global as any).RTCPeerConnection = wrtc.RTCPeerConnection;
    (global as any).RTCSessionDescription = wrtc.RTCSessionDescription;
    (global as any).RTCIceCandidate = wrtc.RTCIceCandidate;
    console.log('[jest.setup] Using @roamhq/wrtc polyfill');
  } catch (error) {
    console.warn('[jest.setup] WebRTC polyfill not available - WebRTC tests will be skipped');
    console.warn('[jest.setup] Install @roamhq/wrtc to run WebRTC tests');
  }
}
