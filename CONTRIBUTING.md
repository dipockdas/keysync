# Contributing to keysync

Thank you for your interest in contributing to keysync! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [your email].

## Ways to Contribute

- **Report bugs** - File detailed bug reports with reproduction steps
- **Suggest features** - Propose new features or improvements
- **Submit pull requests** - Fix bugs, add features, improve documentation
- **Write client libraries** - Add support for new programming languages
- **Improve documentation** - Fix typos, add examples, write tutorials
- **Test on different platforms** - Verify functionality on Windows/Linux/macOS

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- `gh` CLI (for testing GitHub integration)
- Platform-specific keychain tools:
  - **macOS**: Built-in (Keychain Access)
  - **Linux**: `libsecret-tools` (`apt install libsecret-tools`)
  - **Windows**: Built-in (Credential Manager)

### Getting Started

1. **Fork and clone the repository**

   ```bash
   git clone https://github.com/YOUR_USERNAME/keysync.git
   cd keysync
   ```

2. **Build the project**

   ```bash
   make build
   ```

   This creates `./bin/keysync`.

3. **Run tests**

   ```bash
   make test              # Unit tests (all platforms)
   make test-platform     # Platform-specific client tests
   ```

4. **Run the CLI**

   ```bash
   ./bin/keysync --help
   ```

## Project Structure

```
cmd/keysync/           # CLI entry point
internal/commands/     # Cobra commands (set, get, sync, etc.)
internal/config/       # .keysync.json configuration
internal/crypto/       # NaCl encryption for fallback store
internal/github/       # GitHub Secrets client (gh CLI wrapper)
internal/platforms/    # Platform integrations (Vercel, Railway, Supabase)
internal/store/        # OS keychain backends
  ├── darwin.go        # macOS Keychain (security CLI)
  ├── linux.go         # Linux libsecret (secret-tool CLI)
  ├── windows.go       # Windows Credential Manager (wincred library)
  └── fallback.go      # Encrypted file store (CI/testing)
clients/               # Runtime client libraries
  ├── go/              # Go client
  ├── node/            # TypeScript/Node.js client
  ├── python/          # Python client
  ├── swift/           # Swift client
  ├── java/            # Java client
  ├── csharp/          # C# client
  ├── rust/            # Rust client
  ├── cpp/             # C++ client
  └── ruby/            # Ruby client
docs/                  # Tutorials and guides
```

## Making Changes

### 1. Create a branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Make your changes

- Follow Go conventions (run `gofmt`, `go vet`)
- Add tests for new functionality
- Update documentation if needed
- Keep commits focused and atomic

### 3. Test your changes

```bash
# Run all tests
make test

# Test specific package
go test ./internal/commands -v

# Test with race detector
go test -race ./...

# Test platform-specific code
make test-platform
```

### 4. Commit your changes

Write clear commit messages following this format:

```
type: short description

Longer explanation if needed. Wrap at 72 characters.

Fixes #123
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test changes
- `refactor`: Code refactoring
- `chore`: Build/tooling changes

Examples:
```
feat: add GitLab sync support

Implements GitLab CI/CD Variables sync via glab CLI.
Follows the same pattern as GitHub sync.

Closes #45

---

fix: handle spaces in secret values

Secret values with spaces were being truncated.
Now properly quoted in shell export.

Fixes #67
```

## Pull Request Process

1. **Update documentation** if you've changed functionality
2. **Add tests** for new code
3. **Ensure all tests pass** (`make test`)
4. **Update CHANGELOG.md** if applicable
5. **Submit the PR** with a clear description

### PR Title Format

Use the same format as commit messages:

```
feat: add support for BitBucket sync
fix: correct scope precedence in export command
docs: update Windows setup tutorial
```

### PR Description Template

```markdown
## Description
Brief description of what this PR does.

## Motivation
Why is this change needed?

## Changes
- List of changes made
- Another change

## Testing
How was this tested?

## Screenshots (if applicable)
Add screenshots for UI changes.

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] All tests pass
- [ ] Commit messages follow conventions
```

## Writing Client Libraries

To add support for a new programming language:

1. **Review the recipe**: [docs/new-client-library-recipe.md](docs/new-client-library-recipe.md)
2. **Study existing clients**: See `clients/go/` or `clients/python/` for reference
3. **Follow the pattern**:
   - Direct OS keychain access (no keysync binary dependency)
   - Support all three scopes (global, project, environment)
   - Provide in-memory store for testing
   - Include comprehensive README with examples
4. **Test on all platforms**: macOS, Linux, Windows
5. **Document platform support**: Note which OSs are fully tested

## Adding Platform Integrations

To add a new deployment platform (e.g., Netlify, Fly.io):

1. **Implement the `Platform` interface** (see `internal/platforms/platform.go`)
2. **Add configuration struct** to `internal/config/config.go`
3. **Write tests** following `internal/platforms/vercel_test.go` pattern
4. **Document token requirements** in README
5. **Add example config** to README and tutorials

Template available at `internal/platforms/example_test.go`.

## Coding Guidelines

### Go Code Style

- Use `gofmt` for formatting
- Run `go vet` and `golangci-lint`
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Keep functions small and focused
- Write tests for all public functions

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to get secret: %w", err)
}

// Good: Return specific errors
if secret == "" {
    return ErrSecretNotFound
}
```

### Testing

```go
// Use table-driven tests
func TestGetSecret(t *testing.T) {
    tests := []struct {
        name    string
        project string
        key     string
        want    string
        wantErr bool
    }{
        {"global secret", "", "API_KEY", "value", false},
        {"project secret", "myapp", "DB_URL", "postgres://", false},
        {"not found", "myapp", "MISSING", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := GetSecret(tt.project, tt.key)
            if (err != nil) != tt.wantErr {
                t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("GetSecret() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Documentation

- Document all exported functions with godoc comments
- Include examples in documentation
- Update README.md for user-facing changes
- Add tutorials for complex features

## Release Process

Releases are managed by maintainers:

1. Version bump in `VERSION` file (semantic versioning)
2. Update `CHANGELOG.md`
3. Tag release: `git tag v1.x.x`
4. Push tag: `git push origin v1.x.x`
5. GitHub Actions builds and publishes release

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: File an issue with reproduction steps
- **Features**: Open an issue to discuss before implementing
- **Security**: See [SECURITY.md](SECURITY.md) for vulnerability reporting

## Recognition

Contributors will be:
- Listed in release notes
- Credited in commit history
- Mentioned in documentation (for significant contributions)

Thank you for contributing to keysync! 🎉
