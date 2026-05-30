# Keysync test suite

Quick reference for running unit tests and CI security checks. For contributor patterns (table-driven tests, `httptest`, command helpers), see [testing.md](testing.md).

## Running tests locally

```bash
# All internal packages (with race detector)
make test

# Same, less verbose
make test-short

# Platform client tests (generic engine + first-party configs)
make test-platform

# Go client library only
make test-clients

# Internal + Go client
make test-all
```

### Per-package

```bash
go test ./internal/commands/... -v -count=1 -run 'TestSetCmd_ProjectScope'
go test ./internal/store/... -count=1
go test ./internal/platforms/... -count=1
```

### Other client libraries

```bash
cd clients/go && go test ./... -race -count=1
cd clients/python && uv run pytest    # if uv/pytest configured
cd clients/dart && dart test          # when tests exist
```

## What is covered

| Area | Package / path | Focus |
|------|----------------|-------|
| CLI commands | `internal/commands` | `set`, `get`, `list`, `export`, `push` planning, `migrate`, `doctor`, `rotate`, scope/env resolution |
| Keychain store | `internal/store` | Service naming, MemoryStore, FallbackStore (NaCl), concurrency |
| Config | `internal/config` | `.keysync.json` load/save, repo lookup |
| Crypto | `internal/crypto` | NaCl box encrypt/decrypt, key handling |
| GitHub client | `internal/github` | `gh` integration (mocked) |
| Platforms | `internal/platforms` | Generic CLI/HTTP engine, Vercel/Railway/Supabase equivalence, token lookup, response sanitization |
| Go client | `clients/go` | `GetSecret`, service names, Windows cred targets |

Test counts change as the project grows; run `go test ./internal/... -list . | rg '^Test' | wc -l` for a current count.

## Conventions (summary)

- **Go:** table-driven tests with `t.Run`; see [testing.md](testing.md).
- **Commands:** `setupTest()` + in-memory `MemoryStore`; output via `os.Pipe()`.
- **Platforms:** `httptest.NewServer` for HTTP APIs; no live network in unit tests.
- **Stores:** `t.TempDir()` for FallbackStore persistence tests.

## CI security checks (required on `main`)

These run on GitHub Actions for pushes and pull requests to `main` (and on a schedule for some workflows). They are not `go test`, but they gate merges alongside unit tests.

| Check | Workflow | Tool | Purpose |
|-------|----------|------|---------|
| Unit tests + race | [ci.yml](../.github/workflows/ci.yml) | `go test -race ./internal/...` | Regressions and data races |
| Cross-platform build | [cross-platform.yml](../.github/workflows/cross-platform.yml) | `go test` on macOS, Linux, Windows | OS-specific store code |
| Vulnerability scan | [security.yml](../.github/workflows/security.yml) | [govulncheck](https://go.dev/blog/vuln) | Known Go dependency CVEs |
| Secret detection | [security.yml](../.github/workflows/security.yml) | [Gitleaks](https://github.com/gitleaks/gitleaks) | Leaked secrets in the repo |
| Static analysis | [codeql.yml](../.github/workflows/codeql.yml) | [CodeQL](https://codeql.github.com/) | Go security/quality queries |
| Supply chain score | [scorecard.yml](../.github/workflows/scorecard.yml) | [OpenSSF Scorecard](https://scorecard.dev/) | Repository security posture |

Branch protection on `main` requires **`test`**, **`govulncheck`**, and **`gitleaks`** to pass before merge.

### Run security checks locally

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

gitleaks detect --source . --verbose
```

Details and reporting: [SECURITY-SCANNING.md](SECURITY-SCANNING.md).
