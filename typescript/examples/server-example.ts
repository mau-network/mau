/**
 * Example: Express Server
 */

import express from 'express';
import { loadAccount, createAccount, Server } from '../src/index.js';

async function main() {
  const app = express();
  const port = 8080;

  // Create or load account
  let account;
  try {
    account = await loadAccount('./server-data', 'server-passphrase');
    console.log('Loaded existing account');
  } catch {
    account = await createAccount(
      './server-data',
      'Mau Server',
      'server@mau.network',
      'server-passphrase'
    );
    console.log('Created new account');
  }

  console.log('Fingerprint:', account.getFingerprint());

  // Create Mau server
  const server = new Server(account, account.storage);

  // Mount Mau routes
  app.use(server.expressMiddleware());

  // Health check endpoint
  app.get('/', (req, res) => {
    res.json({
      service: 'Mau P2P Server',
      fingerprint: account.getFingerprint(),
      name: account.getName(),
    });
  });

  app.listen(port, () => {
    console.log(`Mau server running at http://localhost:${port}`);
    console.log(`File list: http://localhost:${port}/p2p/${account.getFingerprint()}`);
  });
}

main().catch(console.error);
