# Publishing Guide: @mau-network/mau

This document outlines the complete strategy for publishing the Mau TypeScript library to npm and providing public documentation for library users.

## Table of Contents

1. [Current Status](#current-status)
2. [Publishing Phases](#publishing-phases)
3. [Documentation Strategy](#documentation-strategy)
4. [npm Publishing Setup](#npm-publishing-setup)
5. [Documentation Hosting](#documentation-hosting)
6. [Release Process](#release-process)
7. [Maintenance](#maintenance)

---

## Current Status

**Package Name:** `@mau-network/mau`  
**Current Version:** 0.2.0  
**License:** GPL-3.0  
**Repository:** https://github.com/mau-network/mau (monorepo, TypeScript in `typescript/` directory)

### Pre-Publishing Checklist

- ✅ TypeScript strict mode enabled
- ✅ Zero native dependencies (pure JavaScript)
- ✅ Works in Node.js and Browser environments
- ✅ Package.json properly configured with main/types entries
- ⚠️ ESLint: 20 warnings (mostly `any` types) - **MUST FIX BEFORE v1.0**
- ⚠️ Missing `.npmignore` - **MUST ADD BEFORE PUBLISHING**
- ⚠️ JSDoc gaps in network modules - **RECOMMENDED FOR v0.3+**
- ✅ Test suite present (19 test suites)
- ✅ README.md with examples
- ✅ AGENTS.md with development guidelines

---

## Publishing Phases

### Phase 1: Pre-Release (v0.2.x - Current)

**Goals:** Stabilize API, gather feedback, build community

**Timeline:** 2-4 weeks

**Deliverables:**
1. Create `.npmignore` to reduce package size
2. Fix critical `any` type issues (7 instances)
3. Add JSDoc to public APIs
4. Generate API documentation
5. Deploy hosted documentation

**Action Items:**
- [ ] Create `.npmignore` file
- [ ] Replace `any` types with specific types (files listed below)
- [ ] Add JSDoc comments to WebRTC modules
- [ ] Set up TypeDoc generation
- [ ] Deploy docs to GitHub Pages

**Estimated Effort:** 4-6 hours

---

### Phase 2: Beta Release (v0.3.0)

**Goals:** Release stable API, improve documentation, expand examples

**Timeline:** 4-6 weeks after v0.2.x stabilization

**Deliverables:**
1. v0.3.0 release on npm
2. API documentation (TypeDoc)
3. Advanced examples (WebRTC, DHT, sync)
4. Getting started guide
5. Troubleshooting guide

**Requirements:**
- All `any` types replaced
- JSDoc 100% coverage on public APIs
- Test coverage >45% branches
- No ESLint warnings

**Publishing Steps:**
```bash
npm version minor  # 0.2.0 → 0.3.0
npm publish        # Push to npm registry
```

---

### Phase 3: v1.0.0 Release

**Goals:** Production-ready, stable API

**Timeline:** 6-12 months

**Deliverables:**
1. Stable API freeze
2. Complete documentation
3. Migration guides from v0.x
4. Security audit completed
5. Performance benchmarks

**Requirements:**
- All features documented
- Test coverage >70%
- Zero critical issues
- Community feedback integrated

---

## Documentation Strategy

### 1. API Documentation (Auto-Generated)

**Tool:** TypeDoc

**What it generates:**
- HTML documentation from TypeScript definitions and JSDoc comments
- Type signatures for all public APIs
- Examples embedded in JSDoc
- Search functionality

**Setup:**

```bash
npm install --save-dev typedoc typedoc-plugin-markdown
```

**Configuration (package.json):**

```json
{
  "scripts": {
    "docs": "typedoc --out docs/api --plugin typedoc-plugin-markdown src/index.ts"
  }
}
```

**Implementation:**
1. Add JSDoc comments to all public classes/functions
2. Include `@example` blocks in JSDoc
3. Add `@throws` and `@returns` annotations
4. Generate HTML docs via TypeDoc

---

### 2. User Documentation

**Location:** `typescript/docs/` directory

**Files to Create:**

| File | Purpose | Audience |
|------|---------|----------|
| `docs/getting-started.md` | Installation & first steps | Beginners |
| `docs/core-concepts.md` | Architecture overview | All users |
| `docs/account.md` | Account management | All users |
| `docs/files.md` | File operations | All users |
| `docs/p2p-sync.md` | Peer discovery & sync | Intermediate |
| `docs/webrtc.md` | WebRTC connections | Advanced |
| `docs/storage.md` | Storage backends | Advanced |
| `docs/examples.md` | Complete examples | All users |
| `docs/troubleshooting.md` | Common issues | Support |

**Narrative Documentation Outline:**

```
docs/
├── getting-started.md
│   ├── Installation
│   ├── Hello World (Node.js & Browser)
│   ├── Creating an Account
│   └── Writing Your First File
│
├── core-concepts.md
│   ├── Architecture Overview
│   ├── Friend-Based Security Model
│   ├── Storage Abstraction
│   ├── P2P Networking
│   └── File Encryption Flow
│
├── account.md
│   ├── Creating Accounts
│   ├── Loading Accounts
│   ├── Friend Management
│   ├── PGP Keys & Fingerprints
│   └── Account Backup/Recovery
│
├── files.md
│   ├── Writing Files
│   ├── Reading Files
│   ├── File Operations
│   ├── Versioning
│   └── Sharing with Friends
│
├── p2p-sync.md
│   ├── Peer Discovery
│   ├── Resolvers (Static, DHT, DNS, mDNS)
│   ├── Client Synchronization
│   ├── Conflict Resolution
│   └── Error Handling
│
├── webrtc.md
│   ├── WebRTC Architecture
│   ├── Signaling Servers
│   ├── Data Channels
│   ├── mTLS Authentication
│   └── Building P2P Apps
│
├── storage.md
│   ├── Storage Abstraction
│   ├── Filesystem Storage (Node.js)
│   ├── IndexedDB Storage (Browser)
│   └── Custom Storage Implementation
│
├── examples.md
│   ├── Basic Node.js Example
│   ├── Browser Single-Page App
│   ├── WebRTC P2P Chat
│   ├── Server with Express
│   └── Advanced: Custom Resolvers
│
└── troubleshooting.md
    ├── Common Errors & Solutions
    ├── Browser Compatibility
    ├── Performance Tuning
    ├── Debugging Tips
    └── Getting Help
```

---

### 3. Code Examples

**Location:** `typescript/examples/`

**Examples to Add/Improve:**

1. **`basic-node.ts`** - Simple Node.js usage
   ```typescript
   import { createAccount, File } from '@mau-network/mau';
   
   // Create account → Write file → List files
   ```

2. **`basic-browser.ts`** - Simple browser usage
   ```typescript
   // Same as Node.js but uses IndexedDB
   ```

3. **`webrtc-p2p.ts`** - Full WebRTC example
   ```typescript
   // WebRTCServer + WebRTCClient + Signaling
   ```

4. **`server-express.ts`** - Express.js server
   ```typescript
   // Mount Mau server as Express middleware
   ```

5. **`peer-discovery.ts`** - Using resolvers
   ```typescript
   // Static, DHT, DNS, mDNS resolver examples
   ```

6. **`custom-storage.ts`** - Custom storage backend
   ```typescript
   // Implement custom Storage interface
   ```

---

### 4. README Improvements

**Current README:** ✅ Good (631 lines, comprehensive)

**Recommended Additions:**
- [ ] Add "Why Mau?" section
- [ ] Add table of contents with links
- [ ] Add badges (npm, tests, coverage, license)
- [ ] Add architecture diagram
- [ ] Add comparison with alternatives
- [ ] Separate README for monorepo root and `/typescript`

---

## npm Publishing Setup

### 1. Create `.npmignore`

**Purpose:** Exclude unnecessary files from npm package

**File Content:**

```
# Build artifacts
src/
*.test.ts
jest.config.ts
jest.setup.ts

# Development files
.eslintrc.json
.prettierrc.json
tsconfig.json
vite.config.ts
playwright.config.ts

# CI/CD
.github/

# Documentation
docs/
*.md

# Testing
test-*.mjs
test-*.cjs
test-*.html
e2e/
coverage/
jest.config.js

# Node
node_modules/
npm-debug.log*

# IDE
.vscode/
.idea/

# OS
.DS_Store
.env
```

**Benefits:**
- Reduces package size from ~50MB to ~500KB
- Faster npm install times
- Cleaner package contents

**Implementation:**
```bash
# Add to typescript/.npmignore
cat > typescript/.npmignore << 'EOF'
src/
*.test.ts
jest.config.ts
jest.setup.ts
.eslintrc.json
.prettierrc.json
tsconfig.json
vite.config.ts
playwright.config.ts
.github/
docs/
test-*.mjs
test-*.cjs
test-*.html
e2e/
coverage/
.vscode/
.idea/
.DS_Store
EOF
```

---

### 2. Verify package.json

**Current State:** ✅ Mostly correct

**Recommended Updates:**

```json
{
  "name": "@mau-network/mau",
  "version": "0.2.0",
  "description": "TypeScript implementation of Mau P2P social network protocol",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "type": "module",
  
  "files": [
    "dist/",
    "package.json",
    "README.md",
    "LICENSE"
  ],
  
  "exports": {
    ".": {
      "import": "./dist/index.js",
      "types": "./dist/index.d.ts"
    },
    "./examples": "./examples/"
  },
  
  "repository": {
    "type": "git",
    "url": "https://github.com/mau-network/mau.git",
    "directory": "typescript"
  },
  
  "homepage": "https://mau-network.github.io/docs",
  "bugs": "https://github.com/mau-network/mau/issues",
  
  "keywords": [
    "p2p",
    "social-network",
    "decentralized",
    "pgp",
    "kademlia",
    "webrtc",
    "encryption"
  ],
  
  "author": "Mau Network",
  "license": "GPL-3.0",
  
  "engines": {
    "node": ">=18.0.0"
  },
  
  "publishConfig": {
    "access": "public",
    "registry": "https://registry.npmjs.org/"
  }
}
```

**New Fields Added:**
- `files`: Explicitly whitelist what gets published
- `exports`: Modern ESM exports with types
- `homepage`: Link to documentation
- `bugs`: Issue tracker
- `publishConfig`: Ensure public access

---

### 3. GitHub Actions for npm Publishing

**Create `.github/workflows/publish.yml`:**

```yaml
name: Publish to npm

on:
  push:
    tags:
      - 'v*'

jobs:
  publish:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-node@v4
        with:
          node-version: '18'
          registry-url: 'https://registry.npmjs.org'
      
      - name: Install dependencies
        working-directory: typescript
        run: npm ci
      
      - name: Run linter
        working-directory: typescript
        run: npm run lint
      
      - name: Run tests
        working-directory: typescript
        run: npm test
      
      - name: Build
        working-directory: typescript
        run: npm run build
      
      - name: Publish to npm
        working-directory: typescript
        run: npm publish
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

---

## Documentation Hosting

### Option 1: GitHub Pages (Recommended)

**Setup:**

1. Create docs folder structure:
   ```bash
   mkdir -p typescript/docs-site/api
   mkdir -p typescript/docs-site/guides
   ```

2. Generate TypeDoc HTML:
   ```bash
   npm run docs
   ```

3. Deploy via GitHub Pages workflow:
   ```yaml
   # .github/workflows/docs.yml
   - uses: peaceiris/actions-gh-pages@v3
     with:
       github_token: ${{ secrets.GITHUB_TOKEN }}
       publish_dir: ./typescript/docs-site
   ```

**Result:** https://mau-network.github.io/mau/

---

### Option 2: Vercel (Modern Alternative)

**Benefits:**
- Fast CDN delivery
- Preview deployments
- Analytics included

**Setup:**
```bash
npm install -g vercel

# In typescript/ directory
vercel deploy --prod
```

---

### Option 3: npm Package Docs

**Automatic:** Documentation appears on npmjs.com when README is rendered

**Additional:** Add `"documentation"` link in package.json:
```json
{
  "documentation": "https://mau-network.github.io/mau/api"
}
```

---

## Release Process

### Semver Versioning

```
MAJOR.MINOR.PATCH
1.2.3

MAJOR: Breaking changes
MINOR: New features (backward compatible)
PATCH: Bug fixes
```

**Current:** 0.2.0
- Pre-release phase (0.x)
- Breaking changes allowed
- Increment MINOR for features, PATCH for fixes

### Release Checklist

**Before Each Release:**

- [ ] Update CHANGELOG.md
  ```markdown
  ## [0.3.0] - 2024-03-15
  ### Added
  - New WebRTC signaling server
  - DTOs for serialization
  
  ### Fixed
  - Browser DHT resolver issue
  
  ### Breaking Changes
  - None
  ```

- [ ] Run full test suite
  ```bash
  npm test -- --coverage
  ```

- [ ] Run linter with no warnings
  ```bash
  npm run lint
  ```

- [ ] Verify build succeeds
  ```bash
  npm run build
  npm run build:browser
  ```

- [ ] Update version
  ```bash
  npm version minor  # or patch/major
  ```

- [ ] Tag and push
  ```bash
  git push origin main --tags
  ```

- [ ] npm publishes automatically via GitHub Actions

---

## Maintenance

### Long-Term Strategy

#### Quarterly Reviews

- [ ] Update dependencies (`npm outdated`)
- [ ] Review security advisories (`npm audit`)
- [ ] Analyze GitHub issues and PRs
- [ ] Plan next release features

#### Annual Tasks

- [ ] Review code organization
- [ ] Refactor large modules
- [ ] Performance optimization
- [ ] Documentation refresh

### Feedback Loop

**Monitoring:**
- GitHub Issues (bug reports, feature requests)
- GitHub Discussions (questions, ideas)
- npm package page (user ratings)
- npm downloads analytics

**Community Engagement:**
- Respond to issues within 48 hours
- Tag similar issues/discussions
- Create templates for better reports
- Monthly digest of feedback

---

## Summary: Timeline

```
Week 1-2:     Prepare for v0.2.x (fix linting, add .npmignore)
Week 2-3:     Documentation setup (TypeDoc, guides)
Week 3-4:     v0.2.0+ releases (bug fixes, minor features)
Week 5-8:     Gather community feedback
Month 2:      v0.3.0 release (beta feature freeze)
Month 3-6:    Documentation expansion, example repository
Month 6-12:   Path to v1.0.0 (production release)
```

---

## Quick Commands Reference

```bash
# Development
npm install
npm run build
npm run lint
npm test

# Documentation
npm run docs              # Generate TypeDoc HTML
npm run docs -- --serve  # Serve docs locally

# Publishing (GitHub Actions automated)
npm version patch        # v0.2.0 → v0.2.1
git push origin main --tags

# Manual publishing (if needed)
npm publish --access public
```

---

## Contacts & Resources

**Repository:** https://github.com/mau-network/mau

**npm Package:** https://www.npmjs.com/package/@mau-network/mau

**Documentation:**
- GitHub Pages: https://mau-network.github.io/mau/
- GitHub Discussions: https://github.com/mau-network/mau/discussions

**Maintenance Team:** @emad-elsaid
