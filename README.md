# keysync

**Unified secret management CLI** — store and sync secrets across local OS keychains, GitHub Actions, and deployment platforms (Vercel, Railway, Supabase).

Keysync replaces scattered `.env` files and manual secret management with a single workflow: secrets live in your OS keychain locally and are synced to GitHub Secrets and your deployment platforms on push.

## Features

- **OS-native secret storage** — macOS Keychain, Linux libsecret, Windows Credential Manager
- **Two scopes** — global secrets (shared across projects) and project-scoped secrets (override globals)
- **Platform sync** — push secrets to Vercel, Railway, and Supabase via their APIs
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

Set API tokens as environment variables:

```bash
export VERCEL_TOKEN=...
export RAILWAY_TOKEN=...
export SUPABASE_TOKEN=...
```

Then sync:

```bash
keysync sync -p my-app
```

## Commands

| Command | Description |
|---------|-------------|
| `keysync init` | Scaffold `.keysync.json` in the current directory |
| `keysync set KEY=value` | Store a secret in the OS keychain |
| `keysync get KEY` | Retrieve a secret value |
| `keysync list` | List all stored secrets |
| `keysync sync` | Push secrets to configured platforms + GitHub |
| `keysync pull` | Reconcile local secrets with GitHub secret names |
| `keysync rotate KEY` | Generate a new random secret and update everywhere |
| `keysync migrate` | Import secrets from `.env` file or cloud platform |
| `keysync doctor` | Run diagnostics |
| `keysync test-secrets` | Generate ephemeral test secrets |

### Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to `.keysync.json` (auto-searches parent directories) |
| `-p, --project` | Project name (from `.keysync.json`) |
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

### Platform tokens (environment variables)

| Variable | Platform |
|----------|----------|
| `VERCEL_TOKEN` | Vercel API token |
| `RAILWAY_TOKEN` | Railway API token |
| `SUPABASE_TOKEN` | Supabase Management API token |

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                     CLI (cobra)                      │
│  init │ set │ get │ list │ sync │ rotate    │
│  pull │ migrate │ doctor │ test-secrets   │
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

Secrets are stored with a service name like `keysync/global` (for global scope) or `keysync/project/my-app` (for project scope). The account/key name is the secret key itself. When syncing, project-scoped secrets take precedence over global secrets with the same key.

## OS-Specific Setup

### macOS

Keysync uses the **macOS Keychain** via the `security` CLI (no CGo required, no cross-compilation issues).

**Requirements:**
- macOS (tested on Ventura / Sonoma)
- `security` CLI (built-in)

**How it works:**
- Secrets are stored as generic passwords in the default Keychain
- Service names: `keysync/global` or `keysync/project/<name>`
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
- Service names: `keysync/global` or `keysync/project/<name>`
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
- Target names: `keysync_global` or `keysync_project_<name>`
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

## Go Client Library

Keysync exposes an importable Go client:

```go
import "github.com/dipockdas/keysync/client"

val, err := client.GetSecret("my-project", "DATABASE_URL")
```

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

## Project Status

Keysync is in active development. See [HANDOVER.md](HANDOVER.md) for detailed architecture and known gaps.
