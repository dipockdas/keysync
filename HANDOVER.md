# keysync — Handover Document

## Overview

CLI tool for unified secret management. Source of truth is **GitHub Secrets**; secrets are synced to the local **macOS Keychain** (or Linux libsecret / Windows Credential Manager) and pushed to deployment platforms (**Vercel, Railway, Supabase**) via a pluggable platform system.

## Current State (Phase 4 Complete)

### Implemented Commands

| Command | Status | Description |
|---------|--------|-------------|
| `init` | Done | Scaffold `.keysync.json` in current directory |
| `migrate` | Done | Interactive import from `.env` file → OS keychain |
| `set` | Done | Write to OS keychain + GitHub Secrets |
| `get` | Done | Read from OS keychain (project scope → global fallback) |
| `list` | Done | List all managed secrets |
| `inject` | Done | Generate `.env.local` or shell exports; `--ci` for GitHub Actions |
| `pull` | Done | Reconcile secret names with GitHub (values not exposed by GitHub API) |
| `sync` | Done | Push secrets to all configured platforms via pluggable system |
| `rotate` | Done | Generate crypto/rand 32-byte base64 value, update everywhere |
| `doctor` | Done | Verify config, OS store, platform token availability |
| `test-secrets` | Done | Generate N random test secrets for a project |

### Not Yet Implemented

- **`--cloud` flag on migrate** — pull live env vars from Vercel/Railway/Supabase CLIs instead of a local `.env` file
- **`--dry-run` flag on migrate** — preview what migrate would do without side effects
- **Client-side encryption** — `internal/crypto/crypto.go` exists (NaCl sealed boxes) but is not wired into the GitHub client yet
- **Linux/Windows store implementations** — `libsecret_linux.go` and `wincred_windows.go` are stubs that return "not yet implemented"
- **Full test suite** — only manual testing done so far; `MemoryStore` is ready for unit tests

## Project Structure

```
cmd/keysync/main.go          — Entry point
internal/
  commands/                  — cobra CLI commands (one file per command)
  config/config.go           — .keysync.json loader/parser
  store/                     — Store interface + OS-specific implementations
    store.go                 — Store interface, MemoryStore
    keychain_darwin.go       — macOS Keychain via `security` CLI
    fallback.go              — Encrypted file store (headless Linux fallback)
    libsecret_linux.go       — Linux stub
    wincred_windows.go       — Windows stub
  github/github.go           — GitHub Secrets client via `gh` CLI
  platforms/                 — Pluggable platform sync system
    platform.go              — Platform interface + registry
    vercel.go                — Vercel REST API
    railway.go               — Railway GraphQL API
    supabase.go              — Supabase Management API
  crypto/crypto.go           — NaCl sealed box encryption (unused, available)
client/                      — Importable Go client library
  client.go                  — GetSecret() via keysync CLI
  store.go                   — Store interface re-export
  memory_store.go            — In-memory Store for testing
```

## Key Architecture Decisions

1. **`security` CLI instead of CGo** — macOS Keychain accessed via `security` CLI to avoid CGo cross-compilation issues. Same pattern for other platforms: `secret-tool` (Linux), `cmdkey` (Windows).

2. **Pluggable platform registry** — New platforms register via `init()`:
   ```go
   func init() {
       Register("my-platform", newMyPlatformFromConfig)
   }
   ```

3. **Index file for fast listing** — `~/.config/keysync/index.json` tracks which keys exist in the keychain so `list` is fast instead of parsing `security dump-keychain` output.

4. **Scope resolution** — Project scope overrides global:
   ```
   keysync/global/OPENAI_API_KEY → "sk-org-xxx"    (shared)
   keysync/project/my-app/OPENAI_API_KEY → "sk-proj-yyy" (override)
   ```

## Build & Run

```bash
make build              # Builds to ./bin/keysync
make clean              # Removes ./bin/
keysync init            # Start a new project
keysync doctor          # Verify everything works
```

Single binary, ~11MB. Cross-compile with `GOOS=linux GOARCH=amd64 go build`.

## GitHub

- Repo: `github.com/dipockdas/keysync` (private)
- CI: `.github/workflows/sync-secrets.yml` — runs on push to main

## Next Steps

1. **`--cloud` flag on migrate** — shell out to `vercel env pull`, `railway variables`, `supabase secrets list` and pipe into the interactive migrate flow
2. **`--dry-run` flag on migrate** — preview without side effects
3. **Encrypt fallback store** — wire `internal/crypto` into `fallback.go` for at-rest encryption of the file store
4. **Linux/Windows store impls** — implement `libsecret_linux.go` (via `secret-tool`) and `wincred_windows.go` (via `cmdkey`)
5. **Test suite** — write unit tests using `MemoryStore` and `httptest` for platform clients
6. **Homebrew tap** — distribute via `brew install steath/keysync`
7. **Skill packaging** — package as a Claude Code skill for LLM-guided migration
