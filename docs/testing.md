# Testing Guide

This document describes how tests are structured in keysync, how to run them, and how to add new ones. It's aimed at contributors who want to understand or extend the test suite.

## Quick Start

```bash
# Run all internal package tests (with race detector)
make test

# Run platform-specific API client tests
make test-platform

# Run Go client library tests
make test-clients

# Everything
make test-all
```

## Test Layout

Tests live alongside the code they test, following Go conventions:

```
internal/
├── commands/          # CLI command tests
├── config/            # .keysync.json config tests
├── crypto/            # Encryption/decryption tests
├── github/            # GitHub Secrets client tests
├── platforms/         # Vercel, Railway, Supabase API client tests
└── store/             # OS keychain + fallback store tests
```

## Running Tests

### Full Test Suite

```bash
make test
# Runs: go test ./internal/... -v -race -count=1
```

### Single Package

```bash
go test ./internal/github/... -v -count=1
```

### Specific Test

```bash
go test ./internal/commands/... -v -count=1 -run 'TestExportCmd_GlobalOnly'
```

### Race Detector

All tests run with `-race` by default. This is important because secret stores are accessed concurrently during sync operations.

```bash
go test ./internal/... -race -count=1
```

## Testing Patterns

### 1. Table-Driven Tests

Used for pure functions where you want to test many inputs. Follow this pattern:

```go
func TestScopeLabel(t *testing.T) {
    tests := []struct {
        name    string
        scope   store.Scope
        project string
        want    string
    }{
        {"global", store.ScopeGlobal, "", "global"},
        {"project", store.ScopeProject, "my-app", "project/my-app"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := scopeLabel(tt.scope, tt.project)
            if got != tt.want {
                t.Errorf("scopeLabel(%q, %q) = %q, want %q", tt.scope, tt.project, got, tt.want)
            }
        })
    }
}
```

### 2. Global State Management (commands package)

CLI commands store state in package-level variables (`secretSt`, `cfg`, `project`, `envFlag`, etc.). The `setupTest()` helper saves and restores these:

```go
func TestGetCmd_GlobalScope(t *testing.T) {
    defer setupTest(t)()  // saves globals, injects MemoryStore + mock config

    secretSt.Set(ctx, store.ScopeGlobal, "", "", "MY_KEY", "global-val")
    cmd := newGetCmd()
    getUnmask = true
    defer func() { getUnmask = false }()

    stdout, stderr, err := captureCommand(cmd, []string{"MY_KEY"})
    // assert...
}
// setupTest restore runs automatically via defer
```

### 3. Capturing Command Output

Commands write directly to `os.Stdout`/`os.Stderr` (not cobra's output streams). The `captureCommand` helper redirects these at the OS level:

```go
func captureCommand(cmd *cobra.Command, args []string) (stdout, stderr string, err error) {
    // Saves os.Stdout/os.Stderr, creates pipes, runs cmd.RunE,
    // restores originals, reads pipe contents.
}
```

### 4. HTTP API Mocking (platforms package)

Platform API clients accept an `HTTPClient` interface, making them testable with `httptest`:

```go
func TestVercelUpsert_Success(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request method, headers, body
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{}`))
    }))
    defer ts.Close()

    client := &VercelClient{
        token:     "test-token",
        projectID: "proj_abc",
        client:    ts.Client(),  // inject test HTTP client
    }
    // test...
}
```

### 5. CLI Subprocess Mocking (github package)

The GitHub client wraps the `gh` CLI. Tests use Go's standard `TestHelperProcess` pattern to replace `exec.Command` with a subprocess that emulates the CLI:

```go
func TestHelperProcess(t *testing.T) {
    if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
        return
    }
    defer os.Exit(0)
    // Inspect os.Args to determine which command to emulate
    // Read GH_TEST_FAIL_MODE env var to simulate errors
}
```

Test setup injects the helper:

```go
func TestNewClient_AutoDetect(t *testing.T) {
    defer setHelperProcess(t)()  // replaces exec.Command
    client, err := NewClient("") // will use fake git/gh
    // assert...
}

func TestSet_Error(t *testing.T) {
    defer setFailMode(t, "gh-secret-set")()  // gh CLI fails
    err := client.Set("KEY", "val")
    // assert error...
}
```

### 6. Environment Variable Override (platforms package)

Token lookup functions use an overridable `osGetenv` variable:

```go
func TestLookupToken_FromEnvVarFallback(t *testing.T) {
    orig := osGetenv
    osGetenv = func(key string) string {
        if key == "VERCEL_TOKEN" {
            return "env-token-xyz"
        }
        return ""
    }
    defer func() { osGetenv = orig }()

    token := lookupToken(ctx, nil, "vercel")
    // assert...
}
```

### 7. In-Memory Store

All store-dependent tests use `store.MemoryStore` instead of real OS keychains:

```go
secretSt := store.NewMemoryStore()
secretSt.Set(ctx, store.ScopeGlobal, "", "", "API_KEY", "value")

val, err := secretSt.Get(ctx, store.ScopeGlobal, "", "", "API_KEY")
```

## Writing Tests for a New Command

1. Create `internal/commands/your_command_test.go`
2. Use `setupTest(t)` at the start of each test
3. Pre-seed `secretSt` with test data
4. Create the command with `newYourCmd()`
5. Call `captureCommand(cmd, args)` to execute
6. Assert on stdout, stderr, and the store state

```go
func TestYourCmd_Basic(t *testing.T) {
    defer setupTest(t)()
    secretSt.Set(ctx, store.ScopeGlobal, "", "", "MY_KEY", "val")

    cmd := newYourCmd()
    stdout, stderr, err := captureCommand(cmd, []string{"arg1"})

    assert.NoError(t, err)
    assert.Contains(t, stdout, "expected output")
    assert.Empty(t, stderr)
}
```

## CI Integration

Tests run on GitHub Actions across three OSes (ubuntu, macos, windows):

```yaml
- name: Run tests
  run: go test ./internal/... -race -count=1
```

The CI configuration is in `.github/workflows/ci.yml`.

## Coverage Summary by Package

| Package | What's Tested |
|---------|---------------|
| `crypto` | Key generation, encrypt/decrypt round-trip, wrong key, corrupted data, large/empty payloads |
| `config` | Save/load, parent-directory search, invalid JSON, empty file, project-to-repo lookup |
| `store` | Service name helpers, MemoryStore CRUD, list/sort, concurrency, FallbackStore persistence |
| `github` | Auto-detect, explicit repo, Set/List/Delete success and error, JSON parsing errors |
| `platforms` | Registry, Vercel/Railway/Supabase Upsert (success, API error, GraphQL error, bulk), token resolution, response sanitization |
| `commands` | get (scope fallback), set (global/project/validation), list (filtering), rm (global/project/env/not-found/validation), export (scope precedence, shell quoting), doctor, rotate, sync helpers (resolveRepoConfig, collectSecrets, configuredPlatforms) |

## Adding Tests for a New Platform Client

1. Create `internal/platforms/yourplatform_test.go`
2. Use `httptest.NewServer` to simulate the platform API
3. Construct your client with `client: ts.Client()`
4. Test success and error responses
5. Test missing-token scenarios
6. See `internal/platforms/example_test.go` for a template
