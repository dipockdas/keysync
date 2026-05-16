# keysync

[![CI](https://github.com/dipockdas/keysync/actions/workflows/ci.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/ci.yml)
[![Sync Secrets](https://github.com/dipockdas/keysync/actions/workflows/sync-secrets.yml/badge.svg)](https://github.com/dipockdas/keysync/actions/workflows/sync-secrets.yml)
[![Release](https://img.shields.io/github/v/release/dipockdas/keysync)](https://github.com/dipockdas/keysync/releases)
[![Go Report](https://goreportcard.com/badge/github.com/dipockdas/keysync)](https://goreportcard.com/report/github.com/dipockdas/keysync)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**Unified secret management CLI** — store and sync secrets across local OS keychains, GitHub Secrets, and deployment platforms (Vercel, Railway, Supabase).

Keysync replaces scattered `.env` files and manual secret management with a single workflow: secrets live in your OS keychain locally and are synced to GitHub Secrets and your deployment platforms on push.

## Features

- **OS-native secret storage** — macOS Keychain, Linux libsecret, Windows Credential Manager
- **Three scope levels** — global, project-scoped, and environment-scoped secrets with automatic fallback
- **Platform sync** — push secrets to Vercel, Railway, and Supabase via their APIs
- **Platform tokens in keychain** — API tokens stored as global secrets, never in plaintext files
- **GitHub Actions integration** — auto-sync secrets on push to `main`
- **Secret rotation** — generate cryptographically random secrets and update everywhere
- **Migration** — import from `.env` files or pull from Vercel/Railway/Supabase CLIs
- **Client libraries** — Go, Python, TypeScript, Swift (read secrets at runtime without the keysync binary)
- **Cross-platform** — macOS, Linux, Windows

## For AI coding assistants

If you are an AI coding assistant helping a user set up or work with keysync, start here:

- [AGENTS.md](AGENTS.md) — project overview, architecture, and client library writing instructions for AI agents
- [CLAUDE.md](CLAUDE.md) — build commands, directory structure, and code conventions
- Client libraries each have their own `AGENTS.md` and `CLAUDE.md` with language-specific instructions (see [clients/](clients/))
- [Client library recipe](docs/new-client-library-recipe.md) — how to implement a keysync client in any language
- [Tutorials](#tutorials) — step-by-step guides for common setups

---

## How it works

Keysync has two layers: a **CLI** for managing secrets and a set of **client libraries** for reading them at runtime.

### What goes where

| Layer | Holds | Example |
|-------|-------|---------|
| **`.keysync.json`** | Non-secret project metadata | Platform project IDs, service names, deployment targets |
| **OS keychain** | All secret values | API tokens, database URLs, encryption keys |
| **Environment variables** | Primary runtime path for secrets | Injected by `eval $(keysync export)` (local) or platform (cloud) |

The `.keysync.json` file tells keysync *which platforms* each project deploys to. The actual secret values — including platform API tokens — live in your OS keychain. This means checking `.keysync.json` into version control is safe: it contains no secrets.

### Runtime architecture: env vars first

Client libraries check environment variables before touching the OS keychain:

| Environment | How secrets get into env vars | App reads via |
|---|---|---|
| **Local dev** | `eval $(keysync export)` in shell profile or service launch script | `os.environ` (no keychain call) |
| **Cloud (Vercel, Railway, Supabase, AWS)** | Platform injects env vars (set via `keysync sync`) | `os.environ` (no keychain call) |
| **CI/CD** | GitHub Actions injects env vars from GitHub Secrets | `os.environ` (no keychain call) |

The keychain is touched only by the CLI during setup and management — never at application runtime. This means:

- **Local dev**: One `eval $(keysync export)` at shell startup, zero prompts for the rest of the session
- **CI/CD**: Platform injects env vars, no keychain involved at all
- **Cloud deployment**: Platform injects env vars via `keysync sync`, app reads from `os.environ` — exactly as it works today
- **No .env files anywhere**: The entire chain is `keychain → export → env var`, no plaintext files

### Three scope levels

Secrets are organized by scope, with higher-precedence scopes overriding lower ones:

```
Highest:  Environment-scoped (project + environment match)
              ↓
          Project-scoped (project match only)
              ↓
Lowest:   Global (no project, no environment)
```

**Global** — a secret available to all projects on this machine.

| Example | Use case |
|---------|----------|
| `keysync set MY_API_KEY=abc123` | A single API key shared across every project. Centralised config where all projects use the same credential — no need to repeat it per project. |

**Project** — a secret scoped to a specific project, overriding any global secret with the same key.

| Example | Use case |
|---------|----------|
| `keysync set -p my-api MY_API_KEY=def456` | Each project has its own API key. Isolation between projects while keeping environments consistent within a project. |

**Environment** — a secret scoped to a specific project *and* environment (e.g. `production`, `staging`, `development`).

| Example | Use case |
|---------|----------|
| `keysync set -p my-api DB_URL=postgres://prod-host/db --env production` | Different database URLs for production vs staging vs development. Environment-specific credentials for security and separation of concerns. |
| `keysync set -p my-api DB_URL=postgres://staging-host/db --env staging` | |
| `keysync set -p my-api DB_URL=postgres://dev-host/db --env development` | |

### Sync pipeline

When you run `keysync sync`, secrets are collected at all three scope levels (with higher precedence overriding lower) and pushed to:

1. **GitHub Secrets** — via the `gh` CLI
2. **Deployment platforms** — via their REST/GraphQL APIs

> **Requires `gh` CLI**: The `sync`, `set`, `rotate`, and `pull` commands all interact with GitHub Secrets via the `gh` CLI. Install it from [cli.github.com](https://cli.github.com) and authenticate with `gh auth login` before using these commands.

**Built-in platforms**: Vercel, Railway, and Supabase are supported out of the box — add their project IDs to `.keysync.json` and store the API tokens as global secrets.

**Custom platforms**: The `Platform` interface (`internal/platforms/platform.go`) is extensible — just implement `Name()` and `Upsert(key, value)` and register via `platforms.Register()`. If you use an AI coding assistant, it can help you write the integration: open `internal/platforms/` and ask it to implement a new platform following the existing patterns.

---

## Installation

### Prerequisites

- Go 1.25+ (for building from source)
- `gh` CLI — authenticated and installed (for GitHub Secrets integration)
- Platform CLIs (optional) — `vercel`, `railway`, `supabase` for `migrate --cloud`

### Build from source

```bash
git clone https://github.com/dipockdas/keysync.git
cd keysync
make build
```

The binary is produced at `./bin/keysync`. Add it to your `PATH`:

```bash
cp ./bin/keysync /usr/local/bin/keysync
```

---

## Quick Start

### 1. Initialize a project

```bash
cd your-project
keysync init --project my-app
```

This creates `.keysync.json` in the current directory.

### 2. Store secrets

```bash
# Global secret (available to all projects on this machine)
keysync set GLOBAL_API_KEY=abc123

# Project-scoped secret (overrides global for this project)
keysync set -p my-app DATABASE_URL=postgres://localhost:5432/mydb

# Environment-scoped secret (overrides project for staging)
keysync set -p my-app DATABASE_URL=postgres://staging-host:5432/stagingdb --env staging
```

Each `set` command also pushes the secret to GitHub Secrets if `gh` is authenticated.

### 3. Store platform API tokens

API tokens for deployment platforms are stored as global secrets in your keychain — never in `.keysync.json`:

```bash
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...
```

Tokens can also be set via environment variables (`VERCEL_TOKEN`, `RAILWAY_TOKEN`, `SUPABASE_TOKEN`) as a fallback for CI/CD environments where the keychain is unavailable.

### 4. Configure platforms

Edit `.keysync.json` to tell keysync which platforms your project deploys to:

```json
{
  "repos": {
    "myorg/my-app": {
      "project": "my-app",
      "globals": ["STRIPE_KEY"],
      "platforms": {
        "vercel": { "projectId": "prj_xxxxx", "target": ["production", "preview"] },
        "railway": { "environment": "production", "service": "abc123" },
        "supabase": { "ref": "abcdefghijklmnopqrst" }
      }
    }
  }
}
```

See [Configuration](#configuration) for all available fields.

### 5. Sync to GitHub and deployment platforms

```bash
keysync sync -p my-app
```

This reads secrets from the keychain at all three scope levels and pushes them to GitHub Secrets, Vercel, Railway, and Supabase.

### 6. Retrieve a secret

```bash
# Copies the value to your clipboard
keysync get DATABASE_URL

# Print to stdout (for scripting)
keysync get DATABASE_URL --unmask
```

Resolution order when `--project` is provided: environment-scoped → project-scoped → global.

### 7. Export as environment variables

```bash
eval $(keysync export --project my-app)
source <(keysync export --project my-app)
```

---

## Commands

| Command | Description |
|---------|-------------|
| `keysync init` | Scaffold `.keysync.json` in the current directory |
| `keysync set KEY=value` | Store a secret in the OS keychain and push to GitHub |
| `keysync get KEY` | Copy a secret to the clipboard (`-u`/`--unmask` to print to stdout) |
| `keysync list` | List stored secrets (`--unmask` to show values) |
| `keysync export` | Print secrets as `export KEY=VALUE` lines |
| `keysync sync` | Push secrets to configured platforms + GitHub (`--platforms` to select specific ones) |
| `keysync pull` | Reconcile local secrets with GitHub secret names |
| `keysync rotate KEY` | Generate a new random secret and update everywhere |
| `keysync migrate` | Import secrets from `.env` or cloud platform CLI (`--dry-run` to preview) |
| `keysync doctor` | Run diagnostics (config, keychain, and store checks) |
| `keysync test-secrets` | Generate ephemeral test secrets (`-c`/`--count` to set number) |

### Global flags

| Flag | Description |
|------|-------------|
| `--config` | Path to `.keysync.json` (auto-searches parent directories) |
| `-p, --project` | Project name (from `.keysync.json`) |
| `-e, --env` | Environment name (default `production`). Used for environment-scoped secrets |
| `--repo` | GitHub repository (`owner/repo`), used with `sync` as an alternative to `--project` |
| `--store` | Secret store backend (`"fallback"` to use NaCl-encrypted file instead of OS keychain). Also settable via `KEYSYNC_STORE` env var |

---

## Configuration

The `.keysync.json` file maps GitHub repos to their project names, allowed global secrets, and deployment platforms. It is searched for in the current directory and all parent directories. If none is found, a default (empty) config is used.

### Full schema

```json
{
  "repos": {
    "myorg/my-app": {
      "project": "my-app",
      "globals": ["STRIPE_KEY"],
      "platforms": {
        "vercel": {
          "projectId": "prj_xxxxx",
          "target": ["production", "preview", "development"]
        },
        "railway": {
          "environment": "production",
          "service": "abc123"
        },
        "supabase": {
          "ref": "abcdefghijklmnopqrst"
        }
      }
    }
  }
}
```

### Repo fields

| Field | Required | Description |
|-------|----------|-------------|
| `project` | Yes | Project name used for scoping secrets in the keychain |
| `globals` | No | List of global keys to include when syncing this repo |
| `platforms` | No | Deployment platform configurations |

### Platform fields

| Platform | Field | Required | Description |
|----------|-------|----------|-------------|
| Vercel | `projectId` | Yes | Vercel project ID (`prj_xxxxx`) |
| Vercel | `target` | No | Environments: `production`, `preview`, `development` |
| Railway | `environment` | No | Railway deployment environment name |
| Railway | `service` | No | Railway project/service ID |
| Supabase | `ref` | Yes | Supabase project reference ID |

### Platform tokens

API tokens are stored as global secrets in your OS keychain, not in `.keysync.json`:

```bash
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...
```

Environment variable fallbacks (for CI/CD):

| Variable | Platform |
|----------|----------|
| `VERCEL_TOKEN` | Vercel API token |
| `RAILWAY_TOKEN` | Railway API token |
| `SUPABASE_TOKEN` | Supabase Management API token |

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                  CLI (cobra commands)                     │
│  init │ set │ get │ list │ export │ sync  │ pull        │
│  rotate │ migrate │ doctor │ test-secrets               │
└──────────┬───────────────────────────────────────────────┘
           │
     ┌─────┴─────┐
     │   Store   │──OS Keychain (macOS Keychain / Linux libsecret / Windows Credential Manager)
     └─────┬─────┘            └── Fallback: NaCl-encrypted file (~/.config/keysync/store.json)
           │
     ┌─────┴──────────┐
     │  GitHub Sync    │── gh secret set (source of truth)
     └─────┬──────────┘
           │
     ┌─────┴──────────────────┐
     │  Platform Sync          │── Vercel / Railway / Supabase APIs
     └────────────────────────┘
```

Secrets in the keychain are identified by a **service name** and an **account name**:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | Secret key (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | Secret key |
| Project + Environment | `keysync/project/<name>/env/<env>` | Secret key |

Windows uses underscores: `keysync_global`, `keysync_project_<name>`, `keysync_project_<name>_<env>`.

When syncing, secrets are merged with three-level precedence: environment-scoped (highest) → project-scoped → global (lowest).

---

## Client Libraries

Retrieve secrets at runtime in your application. Client libraries check environment variables first (the primary path), then fall back to the OS keychain. No dependency on the `keysync` binary.

| Language | Location | macOS | Linux | Windows | Status |
|----------|----------|-------|-------|---------|--------|
| **Go** | `clients/go/` | security CLI | secret-tool CLI | wincred library | Ready |
| **Python** | `clients/python/` | security CLI | secret-tool CLI | ctypes Win32 API | Ready |
| **TypeScript** | `clients/node/` | security CLI | secret-tool CLI | — | Ready (macOS/Linux) |
| **Swift** | `clients/swift/` | Security.framework | secret-tool CLI | — | Ready (macOS/Linux) |

```go
// Go
import "github.com/dipockdas/keysync/clients/go"
dbURL, err := keysync.GetSecret("DATABASE_URL", "my-api")
```

```python
# Python
from keysync import get_secret
db_url = get_secret("DATABASE_URL", project="my-api")
```

```typescript
// TypeScript
import { getSecret } from "@dipockdas/keysync";
const dbUrl = await getSecret("DATABASE_URL", "my-api");
```

```swift
// Swift
import KeySync
let dbURL = try KeySync.getSecret("DATABASE_URL", project: "my-api")
```

See [clients/](clients/) for full documentation and per-library README files.

---

## Platform setup

Each OS keychain is accessed automatically — no manual configuration is usually needed. The notes below cover edge cases and troubleshooting.

### macOS

- Uses the built-in `security` CLI — no additional software required
- Secrets stored as generic passwords in the default Keychain
- Index file at `~/.config/keysync/index.json` tracks stored secret names for fast listing

**Avoiding keychain prompts**

macOS may prompt for your keychain password when accessing secrets. There are two ways to avoid this:

1. **Sign the binary** — A signed binary lets macOS remember your "Always Allow" choice across rebuilds:

   ```bash
   make build && make sign
   # or in one step:
   make build-signed
   ```

   Requires a Developer ID Application certificate in your keychain. If you don't have one, see [Apple's docs](https://developer.apple.com/developer-id).

2. **Use the fallback store** — Skips the keychain entirely and uses the NaCl-encrypted file at `~/.config/keysync/store.json`. Best for services and scripts:

   ```bash
   keysync --store fallback get DATABASE_URL
   # or set the env var persistently:
   export KEYSYNC_STORE=fallback
   ```

**Service launch scripts** — For services that restart, combine the fallback store with export:

```bash
eval $(keysync --store fallback export --project my-app)
exec my-app
```

- View stored secrets: open `Keychain Access.app` > search for "keysync"

### Linux

- Uses `secret-tool` (part of `libsecret`):

  ```bash
  # Debian / Ubuntu
  sudo apt-get install libsecret-tools

  # Fedora
  sudo dnf install libsecret

  # Arch Linux
  sudo pacman -S libsecret
  ```

- Requires a running D-Bus session and an unlocked keyring (GNOME Keyring, KDE Wallet, or KeePassXC)
- On headless servers, start a D-Bus session manually:

  ```bash
  sudo apt-get install libsecret-tools gnome-keyring dbus-x11
  export $(dbus-launch)
  echo -n "" | gnome-keyring-daemon --unlock --daemonize --components=secrets
  ```

- **Fallback**: If `secret-tool` is unavailable or D-Bus is not running, keysync falls back to an encrypted file store at `~/.config/keysync/store.json` (encrypted with NaCl: Curve25519 + XSalsa20-Poly1305)

### Windows

- Uses Windows Credential Manager (Win32 API via `wincred` library) — no additional software required
- Works on Windows 10+, desktop and server editions
- Credentials are stored in "Windows Credentials" (not "Web Credentials")
- View via: Control Panel > Credential Manager > Windows Credentials
- Manage via command line: `cmdkey /list`, `cmdkey /delete`

---

## GitHub Actions Integration

The included workflow at `.github/workflows/sync-secrets.yml` syncs secrets on every push to `main`:

```yaml
name: Sync Secrets
on:
  workflow_dispatch:
  push:
    branches: [main]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build keysync
        run: go build -o ./bin/keysync ./cmd/keysync
      - name: Sync all platforms
        run: ./bin/keysync sync --repo ${{ github.repository }} --store fallback
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
          SUPABASE_TOKEN: ${{ secrets.SUPABASE_TOKEN }}
```

Add the platform API tokens as repository secrets in your GitHub repo settings for this to work. In CI, the tokens are passed as environment variables since the OS keychain is not available.

---

## Migration

### From a `.env` file

```bash
keysync migrate --file .env
```

Interactively imports each secret, prompting for scope (global/project) and confirmation.

### From a cloud platform

Requires the platform CLI to be installed and authenticated:

```bash
keysync migrate --cloud vercel
keysync migrate --cloud railway
keysync migrate --cloud supabase
```

Use `--dry-run` to preview without storing.

---

## Tutorials

- [Go Project Setup](docs/tutorial-go-project.md) — Using keysync with an existing Go project
- [Python Flask App](docs/tutorial-python-flask-app.md) — Retrieving secrets at runtime in a Flask application using the Python client
- [Windows Setup](docs/tutorial-windows-setup.md) — Building, configuring, and using keysync on Windows
- [Client Library Recipe](docs/new-client-library-recipe.md) — How to write a keysync client library in any language using an AI coding assistant

---

## Development

```bash
# Build
make build

# Build and codesign for macOS (avoids keychain "Always Allow" prompt reset)
make build-signed

# Run tests
make test

# Run platform tests (no external dependencies required)
make test-platform

# Clean build artifacts
make clean
```

---

## Project Status

Keysync is in active development. See [HANDOVER.md](HANDOVER.md) for detailed architecture and known gaps.
