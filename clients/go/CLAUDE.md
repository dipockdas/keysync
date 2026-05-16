# keysync Go Client — Claude Instructions

## Build & Test

```bash
cd clients/go
go build ./...      # Build all packages
go test ./... -v    # Run all tests
```

## Key files

```
keysync.go          # Public API (GetSecret, ListSecrets), service name helpers
darwin.go           # macOS: exec security CLI (build tag: darwin)
linux.go            # Linux: exec secret-tool CLI (build tag: linux)
windows.go          # Windows: wincred Go library (build tag: windows)
unsupported.go      # Other platforms: returns error (build tag: !darwin,!linux,!windows)
store.go            # Store interface for testing
memory_store.go     # In-memory test store
keysync_test.go     # Tests
```

## Conventions

- Build tags for platform selection (`//go:build darwin`, `//go:build linux`, etc.)
- No CGo required on any platform
- `init()` functions in tagged files set `platformGet`, `platformList`, `isNotFound` vars
- `ErrNotFound` sentinel error for missing secrets
- Module path: `github.com/dipockdas/keysync/clients/go`

## Migration: replacing os.Getenv with keysync

```go
import "github.com/dipockdas/keysync/clients/go"

// Global secret (shared across projects)
val, err := keysync.GetGlobal("API_KEY")

// Project-scoped secret (falls back to global if no project match)
val, err := keysync.GetSecret("myapp", "DATABASE_URL")

// With context
val, err := keysync.GetGlobalContext(ctx, "API_KEY")

// List all secrets for a scope
secrets, err := keysync.ListSecrets("myapp") // project-scoped including globals
secrets, err := keysync.ListGlobals()        // global only
```
