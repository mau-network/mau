/**
 * Node.js integration test
 */

import { createAccount, File } from './dist/index.js';
import { FilesystemStorage } from './dist/storage/filesystem.js';
import { promises as fs } from 'fs';

const TEST_DIR = './test-integration';

async function cleanup() {
  try {
    await fs.rm(TEST_DIR, { recursive: true, force: true });
  } catch {}
}

async function test() {
  console.log('🧪 Mau Integration Test\n');

  try {
    // Cleanup
    await cleanup();
    console.log('✓ Cleanup complete');

    // Create account
    console.log('\n1. Creating account...');
    const account = await createAccount(
      TEST_DIR,
      'Test User',
      'test@example.com',
      'test-passphrase'
    );
    console.log(`✓ Account created`);
    console.log(`  Name: ${account.getName()}`);
    console.log(`  Email: ${account.getEmail()}`);
    console.log(`  Fingerprint: ${account.getFingerprint()}`);

    // Write a file
    console.log('\n2. Writing test file...');
    const storage = new FilesystemStorage();
    const file = File.create(account, storage, 'test.json');
    await file.writeJSON({
      '@type': 'SocialMediaPosting',
      headline: 'Test Post',
      articleBody: 'This is a test post',
      datePublished: new Date().toISOString(),
    });
    console.log('✓ File written');

    // Read it back
    console.log('\n3. Reading file back...');
    const content = await file.readJSON();
    console.log('✓ File read successfully');
    console.log(`  Headline: ${content.headline}`);

    // List files
    console.log('\n4. Listing files...');
    const files = await File.list(account, storage);
    console.log(`✓ Found ${files.length} file(s)`);
    for (const f of files) {
      console.log(`  - ${f.getName()}`);
    }

    // Write another file (test versioning)
    console.log('\n5. Testing versioning...');
    await file.writeJSON({
      '@type': 'SocialMediaPosting',
      headline: 'Test Post (Updated)',
      articleBody: 'This is an updated test post',
      datePublished: new Date().toISOString(),
    });
    console.log('✓ File updated');

    const versions = await file.getVersions();
    console.log(`✓ Found ${versions.length} previous version(s)`);

    // Cleanup
    console.log('\n6. Cleaning up...');
    await cleanup();
    console.log('✓ Cleanup complete');

    console.log('\n✅ All tests passed!');
    process.exit(0);
  } catch (err) {
    console.error('\n❌ Test failed:', err.message);
    console.error(err.stack);
    await cleanup();
    process.exit(1);
  }
}

test();
