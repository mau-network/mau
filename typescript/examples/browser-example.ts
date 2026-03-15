/**
 * Browser Example - Complete P2P Flow with WebRTC
 * 
 * This example demonstrates how to use Mau in the browser with WebRTC
 * for peer-to-peer communication without traditional HTTP servers.
 */

import { createAccount } from '../src/account.js';
import { WebRTCServer } from '../src/network/webrtc-server.js';
import { WebRTCClient } from '../src/network/webrtc.js';
import { LocalSignalingServer, SignaledConnection } from '../src/network/signaling.js';
import { createStorage } from '../src/storage/index.js';

// ====================
// Scenario: Two Peers Connecting
// ====================

async function exampleTwoPeers() {
  console.log('=== Mau Browser P2P Example ===\n');

  // Create signaling server (in production, this would be a real server)
  const signaling = new LocalSignalingServer();

  // ====================
  // Peer A: Alice
  // ====================
  console.log('Setting up Alice...');
  const aliceStorage = await createStorage('mau-alice');
  const alice = await createAccount(
    'mau-alice',
    'Alice',
    'alice@mau.network',
    'alice-secret'
  );
  const aliceFpr = alice.getFingerprint();
  console.log(`Alice fingerprint: ${aliceFpr.slice(0, 16)}...`);

  // Start Alice's WebRTC server
  const aliceServer = new WebRTCServer(alice, alice.getStorage());

  // Alice creates some content
  const alicePost = await alice.createFile('hello.json');
  await alicePost.writeJSON({
    '@type': 'SocialMediaPosting',
    headline: 'Hello from Alice!',
    articleBody: 'This is my first Mau post',
    datePublished: new Date().toISOString(),
  });
  console.log('Alice created: hello.json\n');

  // ====================
  // Peer B: Bob
  // ====================
  console.log('Setting up Bob...');
  const bobStorage = await createStorage('mau-bob');
  const bob = await createAccount('mau-bob', 'Bob', 'bob@mau.network', 'bob-secret');
  const bobFpr = bob.getFingerprint();
  console.log(`Bob fingerprint: ${bobFpr.slice(0, 16)}...`);

  // Bob follows Alice
  await bob.followFriend(aliceFpr, alice.getPublicKey());
  console.log('Bob now follows Alice\n');

  // ====================
  // WebRTC Connection: Bob -> Alice
  // ====================
  console.log('Establishing WebRTC connection...');

  // Bob creates WebRTC client to connect to Alice
  const bobClient = new WebRTCClient(bob, bob.getStorage(), aliceFpr);

  // Step 1: Bob creates offer
  const offer = await bobClient.createOffer();
  console.log('Bob created WebRTC offer');

  // Signaling: Bob sends offer to Alice via signaling server
  await signaling.post({
    from: bobFpr,
    to: aliceFpr,
    type: 'offer',
    data: offer,
  });

  // Alice polls signaling server and gets Bob's offer
  const aliceMessages = await signaling.poll(aliceFpr);
  const bobOffer = aliceMessages.find((m) => m.type === 'offer');

  if (!bobOffer) {
    throw new Error('No offer received');
  }

  // Step 2: Alice accepts offer and creates answer
  const connectionId = `conn-${Date.now()}`;
  const answer = await aliceServer.acceptConnection(connectionId, bobOffer.data);
  console.log('Alice accepted offer and created answer');

  // Signaling: Alice sends answer back to Bob
  await signaling.post({
    from: aliceFpr,
    to: bobFpr,
    type: 'answer',
    data: answer,
  });

  // Bob polls and gets Alice's answer
  const bobMessages = await signaling.poll(bobFpr);
  const aliceAnswer = bobMessages.find((m) => m.type === 'answer');

  if (!aliceAnswer) {
    throw new Error('No answer received');
  }

  // Step 3: Bob completes connection with answer
  await bobClient.completeConnection(aliceAnswer.data);
  console.log('WebRTC connection established!\n');

  // Wait for data channel to open
  await new Promise((resolve) => setTimeout(resolve, 1000));

  // Step 4: Bob performs mTLS handshake
  console.log('Performing mTLS handshake...');
  const authenticated = await bobClient.performMTLS();

  if (!authenticated) {
    throw new Error('mTLS authentication failed');
  }
  console.log('mTLS authentication successful!\n');

  // ====================
  // File Transfer
  // ====================
  console.log('Fetching file list from Alice...');
  const fileList = await bobClient.fetchFileList();
  console.log(`Received ${fileList.files.length} file(s):`);
  fileList.files.forEach((file: any) => {
    console.log(`  - ${file.path} (${file.size} bytes)`);
  });

  console.log('\nDownloading hello.json...');
  const fileData = await bobClient.downloadFile('hello.json');
  console.log(`Downloaded ${fileData.length} bytes`);

  // Bob saves Alice's file
  const bobStorage = bob.getStorage();
  await bobStorage.writeFile(bobStorage.join(bob.getFriendContentDir(aliceFpr), 'hello.json'), fileData);
  console.log('Saved to Bob\'s content directory');

  // Bob reads the content (decrypts automatically)
  const savedAliceFiles = await bob.listFriendFiles(aliceFpr);
  const aliceFile = savedAliceFiles.find(f => f.getName() === 'hello.json');
  if (!aliceFile) {
    throw new Error('File not found after save');
  }
  const content = await aliceFile.readJSON();
  console.log('\nDecrypted content:');
  console.log(JSON.stringify(content, null, 2));

  // ====================
  // Cleanup
  // ====================
  bobClient.close();
  aliceServer.stop();

  console.log('\n=== Example Complete ===');
}

// ====================
// Advanced: Multiple Peers with Signaling
// ====================

async function exampleSignaledNetwork() {
  console.log('=== WebSocket Signaling Example ===\n');

  // In a real app, this would be: wss://signaling.mau.network
  const signalingUrl = 'ws://localhost:8080/signaling';

  const storage = await createStorage('mau-data');
  const account = await createAccount(
    'mau-data',
    'Charlie',
    'charlie@mau.network',
    'charlie-secret'
  );

  const fingerprint = account.getFingerprint();

  // Create WebRTC server
  const server = new WebRTCServer(account, account.getStorage());

  // Connect to signaling server
  const { WebSocketSignaling } = await import('../src/network/signaling.js');
  const signaling = new WebSocketSignaling(signalingUrl, fingerprint);

  // Handle incoming connection offers
  signaling.onMessage(async (message) => {
    if (message.type === 'offer' && message.to === fingerprint) {
      console.log(`Received offer from ${message.from.slice(0, 8)}...`);

      // Accept connection
      const connectionId = `conn-${Date.now()}`;
      const answer = await server.acceptConnection(connectionId, message.data);

      // Send answer back
      await signaling.send({
        from: fingerprint,
        to: message.from,
        type: 'answer',
        data: answer,
      });

      console.log('Sent answer');
    }
  });

  console.log('Server ready and listening for connections');
  console.log(`Fingerprint: ${fingerprint}`);
  console.log('(Press Ctrl+C to stop)');
}

// ====================
// Connect to Remote Peer
// ====================

async function exampleConnectToPeer(peerFingerprint: string) {
  console.log(`=== Connecting to ${peerFingerprint.slice(0, 16)}... ===\n`);

  const signalingUrl = 'ws://localhost:8080/signaling';

  const storage = await createStorage('mau-data');
  const account = await createAccount(
    'mau-data',
    'Dave',
    'dave@mau.network',
    'dave-secret'
  );

  const fingerprint = account.getFingerprint();

  // Connect to signaling server
  const { WebSocketSignaling, SignaledConnection } = await import(
    '../src/network/signaling.js'
  );
  const signaling = new WebSocketSignaling(signalingUrl, fingerprint);

  // Create signaled connection helper
  const connection = new SignaledConnection(signaling, fingerprint, peerFingerprint);

  // Create WebRTC client
  const client = new WebRTCClient(account, account.getStorage(), peerFingerprint);

  // Handle answer from peer
  connection.onAnswer(async (answer) => {
    console.log('Received answer, completing connection...');
    await client.completeConnection(answer);

    // Wait for channel to open
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // Perform mTLS
    console.log('Authenticating...');
    const authenticated = await client.performMTLS();

    if (!authenticated) {
      console.error('Authentication failed');
      return;
    }

    console.log('Connected and authenticated!\n');

    // Fetch file list
    const fileList = await client.fetchFileList();
    console.log(`Files available: ${fileList.files.length}`);
    fileList.files.forEach((file: any) => {
      console.log(`  - ${file.path}`);
    });
  });

  // Create and send offer
  console.log('Creating offer...');
  const offer = await client.createOffer();
  await connection.sendOffer(offer);
  console.log('Offer sent, waiting for answer...');
}

// Run example
if (typeof window !== 'undefined') {
  // Browser environment
  (window as any).mauExample = {
    twoPeers: exampleTwoPeers,
    signaledNetwork: exampleSignaledNetwork,
    connectToPeer: exampleConnectToPeer,
  };

  console.log('Mau examples loaded!');
  console.log('Run: mauExample.twoPeers()');
  console.log('Or: mauExample.signaledNetwork()');
  console.log('Or: mauExample.connectToPeer("<fingerprint>")');
} else {
  // Node.js environment - run simple example
  exampleTwoPeers().catch(console.error);
}
