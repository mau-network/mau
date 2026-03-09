/**
 * Example: Node.js Usage
 */

import { createAccount, loadAccount, File, Client } from '../src/index.js';

async function main() {
  console.log('=== Creating Account ===');
  
  // Create a new account
  const account = await createAccount(
    './example-data',
    'Alice',
    'alice@mau.network',
    'my-strong-passphrase',
    { algorithm: 'ed25519' }
  );

  console.log('Account created!');
  console.log('Fingerprint:', account.getFingerprint());
  console.log('Name:', account.getName());
  console.log('Email:', account.getEmail());

  console.log('\n=== Writing Files ===');

  // Create and write a post
  const post1 = File.create(account, account.storage, 'hello.json');
  await post1.writeJSON({
    '@type': 'SocialMediaPosting',
    headline: 'Hello, Mau!',
    articleBody: 'This is my first post on the Mau network.',
    datePublished: new Date().toISOString(),
  });
  console.log('Created: hello.json');

  // Create another post
  const post2 = File.create(account, account.storage, 'second-post.json');
  await post2.writeJSON({
    '@type': 'SocialMediaPosting',
    headline: 'Second Post',
    articleBody: 'Getting the hang of this!',
    datePublished: new Date().toISOString(),
  });
  console.log('Created: second-post.json');

  console.log('\n=== Reading Files ===');

  // List all files
  const files = await File.list(account, account.storage);
  console.log(`Found ${files.length} files:`);
  
  for (const file of files) {
    const content = await file.readJSON();
    console.log(`- ${file.getName()}: ${(content as any).headline}`);
  }

  console.log('\n=== Modifying File (Creates Version) ===');

  // Modify a file (creates version)
  await post1.writeJSON({
    '@type': 'SocialMediaPosting',
    headline: 'Hello, Mau! (Updated)',
    articleBody: 'This is my first post on the Mau network. I updated it!',
    datePublished: new Date().toISOString(),
  });
  console.log('Updated: hello.json');

  // Check versions
  const versions = await post1.getVersions();
  console.log(`File has ${versions.length} previous version(s)`);

  console.log('\n=== Public Key Export ===');

  // Export public key for sharing
  const publicKey = account.getPublicKey();
  console.log('Public key (first 100 chars):');
  console.log(publicKey.substring(0, 100) + '...');

  console.log('\n=== Complete! ===');
  console.log('Your Mau data is stored in: ./example-data/.mau');
}

main().catch(console.error);
