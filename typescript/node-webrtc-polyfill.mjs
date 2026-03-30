/**
 * WebRTC and IndexedDB Polyfills for Node.js
 * 
 * This sets up WebRTC and IndexedDB APIs in Node.js.
 * Import this before using any WebRTC or storage functionality.
 */

import wrtc from '@roamhq/wrtc';
import 'fake-indexeddb/auto';

// Only polyfill if not already defined (i.e., in Node.js, not browser)
if (typeof global.RTCPeerConnection === 'undefined') {
  console.log('[Polyfill] Setting up WebRTC for Node.js using @roamhq/wrtc...');
  
  global.RTCPeerConnection = wrtc.RTCPeerConnection;
  global.RTCSessionDescription = wrtc.RTCSessionDescription;
  global.RTCIceCandidate = wrtc.RTCIceCandidate;
  
  console.log('[Polyfill] ✅ WebRTC polyfill ready');
}

// Export for convenience
export const RTCPeerConnection = global.RTCPeerConnection;
