#!/bin/bash
# Fix ESLint errors in Mau TypeScript implementation

set -e

echo "🔧 Fixing ESLint errors..."

# Fix unused imports in account.ts
sed -i '/IncorrectPassphraseError,/d' src/account.ts

# Fix unused imports in client.ts  
sed -i '/import { File }/d' src/client.ts

# Fix unused imports in crypto/pgp.ts
sed -i '/PassphraseRequiredError,/d' src/crypto/pgp.ts
sed -i '/IncorrectPassphraseError,/d' src/crypto/pgp.ts

# Fix unused variable dnsNames in crypto/pgp.ts (line 264)
sed -i 's/const dnsNames/\/\/ const dnsNames/' src/crypto/pgp.ts

# Fix unused imports in server.ts
sed -i 's/, Fingerprint//' src/server.ts
sed -i '/import { sha256 }/d' src/server.ts

# Fix unused variable afterTime in server.ts (line 91)
sed -i 's/const afterTime =/\/\/ const afterTime =/' src/server.ts

# Fix unused parameters in network/resolvers.ts
# These are placeholder functions, mark params as unused
sed -i 's/domain: string/\_domain: string/' src/network/resolvers.ts
sed -i 's/fingerprint: Fingerprint/\_fingerprint: Fingerprint/g' src/network/resolvers.ts  
sed -i 's/timeout?: number/\_timeout?: number/g' src/network/resolvers.ts
sed -i 's/bootstrapNodes: Peer\[\]/\_bootstrapNodes: Peer\[\]/' src/network/resolvers.ts

# Fix unused imports in network/webrtc-server.ts
sed -i 's/, ServerResponse//' src/network/webrtc-server.ts
sed -i 's/verify, //' src/network/webrtc-server.ts

# Fix unused imports in network/webrtc.ts
sed -i 's/signAndEncrypt, //' src/network/webrtc.ts
sed -i 's/decryptAndVerify//' src/network/webrtc.ts
sed -i 's/, decryptAndVerify//' src/network/webrtc.ts

# Fix empty catch blocks in test files
sed -i 's/} catch (err) {}/} catch (err) { \/* expected *\/ }/' src/account.test.ts
sed -i 's/} catch (err) {}/} catch (err) { \/* expected *\/ }/' src/file.test.ts

echo "✅ ESLint errors fixed!"
echo ""
echo "Running eslint to verify..."
npm run lint
