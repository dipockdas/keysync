# Keysync Test Suite

## Running Tests

```bash
# Core packages: store, config, crypto, commands
make test

# Core packages (no verbose output)
make test-short

# Platform API tests (Vercel, Railway, Supabase)
make test-platform

# Go client library
cd clients/go && go test ./...

# Python client library
cd clients/python && uv run pytest
```

## Test Coverage

| Package | File | Tests | Description |
|---------|------|-------|-------------|
| `internal/store` | `store_test.go` | 11 | Service naming, MemoryStore CRUD, concurrency |
| `internal/store` | `fallback_test.go` | 7 | FallbackStore CRUD, persistence, concurrency |
| `internal/config` | `config_test.go` | 9 | Config load/save, search, parsing |
| `internal/crypto` | `crypto_test.go` | 12 | Key generation, encrypt/decrypt, wrong key rejection |
| `internal/platforms` | `platform_test.go` | 3 | Platform registry |
| `internal/platforms` | `vercel_test.go` | 5 | Vercel API upsert, errors, token validation |
| `internal/platforms` | `railway_test.go` | 5 | Railway API upsert, errors, token validation |
| `internal/platforms` | `supabase_test.go` | 5 | Supabase API upsert, errors, token validation |
| `internal/commands` | `commands_test.go` | 16 | set/get/list/rotate/doctor/migrate commands |
| `clients/go/` | `keysync_test.go` | 13 | Go client library get/list/service names |
| `clients/python/` | `test_client.py` | 11 | Python client service names, error types |
| **Total** | | **97** | |

## Test Conventions

### Table-driven tests (Go)

All Go packages use table-driven tests with `t.Run` subtests. Each test case is a struct in a slice with a `name` field for readable output:

```go
tests := []struct {
    name    string
    scope   Scope
    project string
    want    string
}{
    {"global no project", ScopeGlobal, "", "keysync/global"},
    // ...
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

### Platform API tests

Each platform (Vercel, Railway, Supabase) uses `net/http/httptest` to mock API servers:

```go
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Validate request method, headers, body
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{}`))
}))
defer ts.Close()

client := &VercelClient{baseURL: ts.URL, client: ts.Client(), ...}
```

Tests cover success, API errors, and missing tokens.

### Command tests

Commands use a `setupTest()` helper that saves and restores package-level global variables, since the CLI package uses globals rather than dependency injection:

```go
func setupTest(t *testing.T) func() {
    t.Helper()
    origSecretSt := secretSt
    secretSt = store.NewMemoryStore()
    // ...
    return func() {
        secretSt = origSecretSt
        // ...
    }
}
// Usage:
defer setupTest(t)()
```

Command output is captured via OS-level pipe redirection (`os.Pipe()` with stdout/stderr substitution).

### Encryption tests

The crypto package tests verify:
- Key generation produces non-nil keys
- Encrypt/decrypt round-trips for empty, small, and 1MB payloads
- Decryption with the wrong key fails
- Decryption of corrupted ciphertext fails
- Two random keys are unique

### Store tests

Both `MemoryStore` and `FallbackStore` cover:
- Set and Get round-trip
- Get returns `ErrNotFound` for missing keys
- Delete removes keys
- Delete returns `ErrNotFound` for missing keys
- List returns all entries, with scope/project filtering
- Concurrent reads and writes are safe
- Persistence (FallbackStore): data survives store close/reopen

### FallbackStore encryption

The FallbackStore encrypts data using NaCl box (Curve25519 + XSalsa20-Poly1305) before writing to disk. On load, it transparently migrates plaintext JSON from previous versions to encrypted format.

## Key test patterns

| Pattern | Used in |
|---------|---------|
| `t.TempDir()` | Config, Fallback, Go client tests |
| `httptest.NewServer` | Vercel, Railway, Supabase tests |
| `os.Pipe()` output capture | Command tests |
| `t.Setenv()` | Platform token validation tests |
| `sync.WaitGroup` | Store concurrency tests |
| `bytes.Equal` | Crypto round-trip verification |
