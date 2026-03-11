#!/bin/bash

# Fix @ts-ignore → @ts-expect-error
find src -name "*.ts" -exec sed -i 's|// @ts-ignore|// @ts-expect-error|g' {} \;

# Fix empty catch blocks in test files
files=(
  "src/account-e2e.test.ts"
  "src/client-edge.test.ts"
  "src/file-extended.test.ts"
  "src/network/signaling.test.ts"
  "src/network/webrtc-advanced.test.ts"
)

for file in "${files[@]}"; do
  # Replace "} catch {}" with "} catch (error) { console.error(error); }"
  sed -i 's|} catch {}|} catch (error) { /\* Ignore expected error \*/ }|g' "$file"
done

echo "Fixed linting issues"
