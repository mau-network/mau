# Contributing to Mau

Thank you for your interest in contributing to **Mau**! This guide will help you get started with development, testing, and submitting contributions.

## Table of Contents

- [Quick Start](#quick-start)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Community Guidelines](#community-guidelines)

## Quick Start

### Prerequisites

**For Go development:**
```bash
# Go 1.23 or later
go version

# Install build dependencies
sudo apt-get install -y libgpgme-dev gpg
```

**For GUI development:**
```bash
# GTK4 + Libadwaita
sudo apt-get install -y \
    libgtk-4-dev \
    libadwaita-1-dev \
    libgpgme-dev \
    pkg-config
```

**For TypeScript/Node development:**
```bash
# Node.js 18+ and npm
node --version
npm --version
```

### Clone and Setup

```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/mau.git
cd mau

# Add upstream remote
git remote add upstream https://github.com/mau-network/mau.git

# Fetch upstream
git fetch upstream

# Install Go dependencies
go mod download

# Run tests to verify setup
go test ./...
```

## Development Workflow

### 1. Sync with Upstream

**Always start by syncing with upstream master:**

```bash
git checkout master
git fetch upstream
git pull upstream master
git push origin master
```

### 2. Create a Feature Branch

Use descriptive branch names following these conventions:

```bash
# New features
git checkout -b feature/add-profile-pictures

# Bug fixes
git checkout -b fix/race-condition-in-sync

# Documentation improvements
git checkout -b docs/improve-api-reference

# Improvements/enhancements
git checkout -b improvement/optimize-dht-lookup
```

### 3. Make Your Changes

- Write clear, focused commits
- Follow code style guidelines (see below)
- Add tests for new functionality
- Update documentation if needed

### 4. Test Your Changes

```bash
# Run Go tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test -v ./cmd/mau

# Build GUI (if applicable)
cd gui && make clean && make
```

### 5. Commit Your Changes

Write meaningful commit messages:

```bash
git add .
git commit -m "feat: Add support for profile pictures

- Add ProfilePicture field to Account struct
- Implement image upload and storage
- Add tests for image validation
- Update API documentation"
```

**Commit message format:**
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Test additions/improvements
- `refactor:` Code refactoring
- `style:` Code formatting/style changes
- `chore:` Build/tooling changes

### 6. Push and Create Pull Request

```bash
# Push to your fork
git push origin feature/add-profile-pictures

# Create PR using GitHub CLI
gh pr create --repo mau-network/mau \
  --title "feat: Add support for profile pictures" \
  --body "Implements profile picture support with upload, storage, and display."
```

## Code Standards

### Go Code Style

**Follow standard Go conventions:**

```bash
# Format code
go fmt ./...

# Run linter (if installed)
golangci-lint run
```

**Key guidelines:**
- Use `gofmt` for formatting
- Keep functions small and focused (<50 lines when possible)
- Write descriptive variable names
- Add comments for exported functions/types
- Use error wrapping: `fmt.Errorf("context: %w", err)`

**Example:**

```go
// AddFile adds a new encrypted file to the account directory.
// The filename must not contain path separators (enforces flat structure).
// Files are automatically encrypted with PGP if not already .pgp files.
func (a *Account) AddFile(filename string, content []byte) error {
    if err := validateFlatFileName(filename); err != nil {
        return fmt.Errorf("invalid filename: %w", err)
    }
    // ... implementation
}
```

### TypeScript Code Style

**Use ESLint and Prettier:**

```bash
cd typescript
npm run lint
npm run format
```

**Key guidelines:**
- Use TypeScript strict mode
- Prefer `const` over `let`
- Use descriptive function/variable names
- Add JSDoc comments for public APIs
- Use async/await over promises

### C Code Style (GUI)

**Follow Linux kernel style:**

```bash
# Use indent for formatting
indent -linux gui/*.c
```

**Key guidelines:**
- Use tabs for indentation
- Keep functions under 100 lines
- Use `snake_case` for functions
- Add comments for complex logic
- Handle all error cases

## Testing

### Writing Tests

**Go tests:**

```go
func TestAddFile(t *testing.T) {
    // Setup
    account, cleanup := setupTestAccount(t)
    defer cleanup()
    
    // Test
    err := account.AddFile("test.json", []byte(`{"test": true}`))
    if err != nil {
        t.Fatalf("AddFile failed: %v", err)
    }
    
    // Verify
    files := account.ListFiles()
    if len(files) != 1 {
        t.Errorf("Expected 1 file, got %d", len(files))
    }
}
```

**Test coverage requirements:**
- New features: >80% coverage
- Bug fixes: Add regression test
- Critical paths: 100% coverage

### Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...

# Specific package
go test -v ./cmd/mau

# Run single test
go test -v -run TestAddFile ./
```

## Documentation

### When to Update Documentation

**Always update docs when:**
- Adding new features
- Changing APIs or behavior
- Fixing bugs that affect usage
- Adding new configuration options

### Documentation Structure

```
docs/
├── 01-introduction.md          # What is Mau?
├── 02-core-concepts.md         # Architecture overview
├── 03a-quickstart-gpg.md       # GPG tutorial
├── 03b-quickstart-cli.md       # CLI tutorial
├── 03c-quickstart-package.md   # Go package tutorial
├── 04-storage-and-data.md      # File format, JSON-LD
├── 05-authentication.md        # PGP, signing, encryption
├── 06-networking.md            # Kademlia, discovery, sync
├── 07-http-api.md              # HTTP endpoints
├── 08-building-social-apps.md  # Practical patterns
├── 09-privacy-security.md      # Security best practices
├── 10-performance.md           # Optimization guide
├── 11-api-reference.md         # Go package reference
├── 12-schema-types.md          # Schema.org types
└── 13-troubleshooting.md       # Common issues
```

### Writing Good Documentation

**Guidelines:**
- Use clear, simple language
- Include code examples
- Add diagrams where helpful
- Link to related sections
- Test all code examples

**Example structure:**

```markdown
## Feature Name

### Overview

Brief description of what the feature does and why it exists.

### Usage

```go
// Code example showing basic usage
account := mau.NewAccount(keyring)
err := account.AddFile("post.json", content)
```

### Advanced Usage

More complex examples and edge cases.

### Common Issues

Known problems and their solutions.

### Related

- Link to [relevant section](#)
- Link to [API reference](#)
```

## Pull Request Process

### Before Submitting

**Checklist:**
- [ ] Code follows style guidelines
- [ ] Tests pass locally (`go test ./...`)
- [ ] New tests added for new functionality
- [ ] Documentation updated
- [ ] Commits are clear and descriptive
- [ ] Branch is up to date with upstream master

### PR Description Template

```markdown
## Description

Brief description of what this PR does.

## Motivation

Why is this change needed? What problem does it solve?

## Changes

- List of specific changes made
- Group related changes together

## Testing

How was this tested?
- Unit tests added/updated
- Manual testing performed
- Edge cases considered

## Breaking Changes

Are there any breaking changes? If yes, describe migration path.

## Related Issues

Fixes #123
Relates to #456
```

### Review Process

1. **Automated checks** run (tests, linting)
2. **Maintainer review** (usually within 1-3 days)
3. **Discussion** on any issues or suggestions
4. **Revisions** if needed
5. **Approval and merge**

### After Merge

Your changes will be:
- Merged into master branch
- Included in next release
- Credited in changelog

## Community Guidelines

### Code of Conduct

Be respectful and inclusive:
- Welcome newcomers
- Be patient with questions
- Provide constructive feedback
- Respect different perspectives

### Getting Help

**Stuck? Need guidance?**
- Check [documentation](docs/README.md)
- Review [existing issues](https://github.com/mau-network/mau/issues)
- Ask in pull request comments
- Reach out to maintainers

### Communication

**Keep communication:**
- Clear and concise
- Focused on technical merits
- Professional and respectful

## Common Contribution Areas

### Good First Issues

**Easy starting points:**
- Fix typos in documentation
- Add code examples to docs
- Improve error messages
- Add unit tests for existing code
- Optimize performance in specific functions

### Areas Needing Help

**Current priorities:**
- Documentation improvements
- Test coverage expansion
- Performance optimization
- GUI enhancements
- TypeScript library features
- Example applications

## Development Tips

### Debugging

```bash
# Enable debug logging
MAU_DEBUG=1 go test -v ./...

# Run with race detector
go test -race ./...

# Profile performance
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
```

### Local Development

```bash
# Quick development iteration
while true; do
    clear
    go test -v ./... || break
    inotifywait -e modify -r .
done
```

### Testing P2P Functionality

```bash
# Create test accounts
mkdir -p /tmp/alice /tmp/bob

# Start Alice's server
MAU_DIR=/tmp/alice mau server --port 8001 &

# Start Bob's server
MAU_DIR=/tmp/bob mau server --port 8002 &

# Test sync
MAU_DIR=/tmp/alice mau sync --peer localhost:8002
```

## License

By contributing to Mau, you agree that your contributions will be licensed under the **GNU General Public License v3.0** (GPL-3.0).

---

**Thank you for contributing to Mau!** 🐾

Your contributions help build a decentralized, user-owned social network. Every improvement, no matter how small, makes Mau better for everyone.
