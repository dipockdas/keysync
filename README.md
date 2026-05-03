# keysync

**Unified secret management CLI** — store and sync secrets across local OS keychains, GitHub Actions, and deployment platforms (Vercel, Railway, Supabase).

Keysync replaces scattered `.env` files and manual secret management with a single workflow: secrets live in your OS keychain locally and are synced to GitHub Secrets and your deployment platforms on push.

## Features

- **OS-native secret storage** — macOS Keychain, Linux libsecret, Windows Credential Manager
- **Three scope levels** — global secrets (shared across projects), project-scoped secrets (override globals), and environment-scoped secrets (override project, e.g. `production`, `staging`)
- **Platform sync** — push secrets to Vercel, Railway, and Supabase via their APIs
- **Platform tokens in keychain** — API tokens are stored as global secrets in your OS keychain, not in plaintext files
- **GitHub Actions integration** — auto-sync secrets on push to `main`
- **Secret rotation** — generate cryptographically random secrets and update everywhere
- **Migration** — import from `.env` files or pull from Vercel/Railway/Supabase CLIs
- **Diagnostics** — `doctor` command to verify config and store are operational
- **Go client library** — importable `client` package for programmatic access
- **Cross-platform** — macOS, Linux, Windows

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

The binary is produced at `./bin/keysync`. Add it to your `PATH` or copy it to a directory in your `PATH`:

```bash
cp ./bin/keysync /usr/local/bin/keysync
```

## Quick Start

### 1. Initialize a config

```bash
cd your-project
keysync init --project my-app
```

This creates `.keysync.json` in the current directory.

### 2. Store a secret

```bash
# Store a global secret (available to all projects)
keysync set GLOBAL_API_KEY=abc123

# Store a project-scoped secret (overrides global for this project)
keysync set -p my-app DB_URL=postgres://localhost:5432/mydb

# Store an environment-scoped secret (overrides project for this environment)
keysync set -p my-app DB_URL=postgres://prod-host/proddb --env production
```

### 3. Sync to deployment platforms

Configure platforms in `.keysync.json`:

```json
{
  "projects": {
    "my-app": {
      "platforms": {
        "vercel": {
          "projectId": "prj_xxxxx",
          "target": ["production", "preview"]
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

Store API tokens as global secrets in your OS keychain:

```bash
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...
```

Tokens can also be set via environment variables (`VERCEL_TOKEN`, `RAILWAY_TOKEN`, `SUPABASE_TOKEN`) as a fallback — the keychain is checked first.

Then sync:

```bash
keysync sync -p my-app
```

### 4. Retrieve a secret

```bash
# Copies the value to your clipboard
keysync get DATABASE_URL
# Output: Key DATABASE_URL copied to clipboard

# Print to stdout instead (for scripting)
keysync get DATABASE_URL --unmask
keysync get DATABASE_URL -u
```

### 5. Export all secrets as environment variables

```bash
eval $(keysync export --project my-app)
source <(keysync export --project my-app)
```

## Commands

| Command | Description |
|---------|-------------|
| `keysync init` | Scaffold `.keysync.json` in the current directory |
| `keysync set KEY=value` | Store a secret in the OS keychain (+ pushes to GitHub). Use `-e`/`--env` to set environment scope |
| `keysync get KEY` | Copy a secret to the clipboard (use `-u`/`--unmask` to print to stdout). Resolves project+env → project → global |
| `keysync list` | List all stored secrets (use `--unmask` to show values) |
| `keysync export` | Print secrets as `export KEY=VALUE` lines for shell eval. Merges all three scope levels |
| `keysync sync` | Push secrets to configured platforms + GitHub (use `--platforms` to select specific ones) |
| `keysync pull` | Reconcile local secrets with GitHub secret names |
| `keysync rotate KEY` | Generate a new random secret and update everywhere |
| `keysync migrate` | Import secrets from `.env` or cloud platform (use `--dry-run` to preview) |
| `keysync doctor` | Run diagnostics (checks config, store, and OS keychain) |
| `keysync test-secrets` | Generate ephemeral test secrets (use `--count`/`-c` to set number) |

### Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to `.keysync.json` (auto-searches parent directories) |
| `-p, --project` | Project name (from `.keysync.json`) |
| `-e, --env` | Environment name (default `production`). Used for environment-scoped secrets |
| `--repo` | GitHub repository (`owner/repo`), auto-detected if not set |

## Configuration

Keysync uses a `.keysync.json` file, searched for in the current directory and all parent directories. If none is found, a default (empty) config is used.

### Full config structure

```json
{
  "projects": {
    "my-app": {
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

### Platform tokens

API tokens are stored as **global secrets in your OS keychain** (not in `.keysync.json`). Environment variables are used as a fallback if no keychain entry exists.

Store them with:

```bash
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...
```

Fallback environment variables:

| Variable | Platform |
|----------|----------|
| `VERCEL_TOKEN` | Vercel API token |
| `RAILWAY_TOKEN` | Railway API token |
| `SUPABASE_TOKEN` | Supabase Management API token |

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                     CLI (cobra)                      │
│  init │ set │ get │ list │ export │ sync  │
│  rotate │ pull │ migrate │ doctor          │
│  test-secrets                             │
└──────────┬──────────────────────────────────────────┘
           │
     ┌─────┴─────┐
     │   Store   │────────── OS Keychain / Fallback file
     └─────┬─────┘
           │
     ┌─────┴──────────┐
     │   GitHub Sync   │── gh secret set
     └─────┬──────────┘
           │
     ┌─────┴──────────────────┐
     │   Platform Sync         │── Vercel / Railway / Supabase
     └────────────────────────┘
```

Secrets are stored with a service name like `keysync/global` (for global scope), `keysync/project/my-app` (for project scope), or `keysync/project/my-app/env/production` (for environment scope). The account/key name is the secret key itself. When syncing, secrets are resolved with three-level precedence: environment-scoped (highest) → project-scoped → global (lowest).

## OS-Specific Setup

### macOS

Keysync uses the **macOS Keychain** via the `security` CLI (no CGo required, no cross-compilation issues).

**Requirements:**
- macOS (tested on Ventura / Sonoma)
- `security` CLI (built-in)

**How it works:**
- Secrets are stored as generic passwords in the default Keychain
- Service names: `keysync/global` (global), `keysync/project/<name>` (project), or `keysync/project/<name>/env/<env>` (environment)
- Account names: the secret key (e.g., `DATABASE_URL`)
- An index file at `~/.config/keysync/index.json` tracks stored secret names for fast listing

**Keychain locations:**
- Config directory: `~/.config/keysync/`
- Index file: `~/.config/keysync/index.json`

**Notes:**
- The first time a secret is accessed, macOS may prompt for Keychain access
- Add the terminal app to Keychain Access > "Always Allow" to suppress prompts
- To view stored keysync secrets: open `Keychain Access.app` > search for "keysync"
- No additional setup or installation required

### Linux

Keysync uses **libsecret** via the `secret-tool` CLI (no D-Bus C bindings required).

**Requirements:**
- `libsecret` package (`secret-tool` CLI)
- A running D-Bus session and a secret service daemon (GNOME Keyring, KDE Wallet, or KeePassXC)

**Installation:**

```bash
# Debian / Ubuntu
sudo apt-get install libsecret-tools

# Fedora
sudo dnf install libsecret

# Arch Linux
sudo pacman -S libsecret
```

**How it works:**
- Secrets are stored in libsecret with attributes: `service=<serviceName>` and `account=<key>`
- Service names: `keysync/global` (global), `keysync/project/<name>` (project), or `keysync/project/<name>/env/<env>` (environment)
- An in-memory name cache is rebuilt on startup by searching for `service=keysync` entries

**Notes:**
- Requires an unlocked keyring (GNOME Keyring auto-unlocks on desktop login)
- On headless servers, set up a D-Bus session with `dbus-run-session` and start `gnome-keyring-daemon`
- If the keyring is locked, `secret-tool` operations will fail — use `make test` after login to verify
- **Fallback**: If `secret-tool` is unavailable or D-Bus is not running, keysync falls back to the encrypted file store at `~/.config/keysync/store.json`

**Headless server setup:**

```bash
# Install dependencies
sudo apt-get install libsecret-tools gnome-keyring dbus-x11

# Start a D-Bus session with an unlocked keyring
export $(dbus-launch)
echo -n "" | gnome-keyring-daemon --unlock --daemonize --components=secrets
```

### Windows

Keysync uses the **Windows Credential Manager** via the `wincred` Go library (Win32 API, no CGo).

**Requirements:**
- Windows 10 or later
- No additional software required — uses built-in Credential Manager

**How it works:**
- Secrets are stored as Generic Credentials in the Windows Credential Manager
- Target names: `keysync_global` (global), `keysync_project_<name>` (project), or `keysync_project_<name>_<env>` (environment)
- Credential attributes: `UserName` = secret key, `CredentialBlob` = secret value, persisted to `LocalMachine`
- An in-memory name cache is rebuilt on startup by scanning for `keysync_` prefixed credentials

**Notes:**
- Credentials are stored in the "Windows Credentials" section (not "Web Credentials")
- To view stored credentials: Control Panel > Credential Manager > Windows Credentials
- To manage via command line: `cmdkey /list` (enumerate), `cmdkey /delete` (remove individual)
- No additional setup or installation required
- Works on both desktop and server editions of Windows

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
        run: ./bin/keysync sync
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
          SUPABASE_TOKEN: ${{ secrets.SUPABASE_TOKEN }}
```

Add the platform API tokens as repository secrets in your GitHub repo settings for this to work.

## Migration

### From a `.env` file

```bash
keysync migrate --file .env
```

Interactively imports each secret, prompting for scope (global/project) and confirmation.

### From a cloud platform

```bash
# Requires the platform CLI to be installed and authenticated
keysync migrate --cloud vercel
keysync migrate --cloud railway
keysync migrate --cloud supabase
```

Use `--dry-run` to preview without storing.

## Client Libraries

Retrieve secrets at runtime in your application using native OS keychain
access — no dependency on the `keysync` binary.

| Language | Location | Status |
|----------|----------|--------|
| **Go** | `clients/go/` | Ready |
| **Python** | `clients/python/` | Ready |
| **TypeScript** | `clients/node/` | Ready (macOS/Linux) |
| **Swift** | `clients/swift/` | Ready (macOS/Linux) |

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

## Development

```bash
# Build
make build

# Run tests
make test

# Run platform tests (requires no external dependencies)
make test-platform

# Clean build artifacts
make clean
```

## Tutorials

- [Go Project Setup](docs/tutorial-go-project.md) — Using keysync with an existing Go project (CLI + Go client library)
- [Python Flask App](docs/tutorial-python-flask-app.md) — Retrieving secrets at runtime in a Python Flask application
- [Windows Setup](docs/tutorial-windows-setup.md) — Building, configuring, and using keysync on Windows
- [New Client Library Recipe](docs/new-client-library-recipe.md) — How to write a keysync client library in any language using an AI coding assistant

## Agent instructions

- [AGENTS.md](AGENTS.md) — general AI agent instructions for the whole project
- [CLAUDE.md](CLAUDE.md) — Claude Code instructions for this repository
- Each client library also has its own `AGENTS.md` and `CLAUDE.md`

## Project Status

Keysync is in active development. See [HANDOVER.md](HANDOVER.md) for detailed architecture and known gaps.
