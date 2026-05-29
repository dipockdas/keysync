---
name: keysync-setup
description: Guide for setting up keysync from source, verifying required CLIs, authenticating supported platforms, and learning the initial CLI workflow for storing and retrieving secrets.
---

# keysync Setup

Guide for setting up [keysync](https://github.com/dipockdas/keysync) — a unified secret management CLI that stores secrets in OS-native keychains and syncs them to GitHub Secrets and deployment platforms (Vercel, Railway, Supabase).

## Prerequisites

Check each prerequisite before proceeding.

### 1. Go

```bash
go version
# Requires Go 1.25+
```

### 2. GitHub CLI (`gh`)

```bash
gh --version
gh auth status
# Required for GitHub Secrets integration
```

### 3. Platform CLIs (optional)

Only needed if the user plans to use `migrate --cloud`:

```bash
vercel --version    # Vercel
railway --version   # Railway
supabase --version  # Supabase
```

## Build from source

```bash
git clone https://github.com/dipockdas/keysync.git
cd keysync
make build
```

The binary is produced at `./bin/keysync`. The user should add it to their PATH:

```bash
cp ./bin/keysync /usr/local/bin/keysync
```

Verify:

```bash
keysync --help
```

## OS keychain setup

Keysync uses the OS-native keychain automatically — no manual setup is usually needed. However, check these per-platform notes if the user encounters issues:

### macOS
- Uses the built-in `security` CLI — no additional software needed
- First access may prompt for Keychain permission; the user can add their terminal to *Keychain Access > Always Allow* to suppress future prompts

### Linux
- Requires `secret-tool` (part of `libsecret`):
  ```bash
  # Debian / Ubuntu
  sudo apt-get install libsecret-tools

  # Fedora
  sudo dnf install libsecret

  # Arch Linux
  sudo pacman -S libsecret
  ```
- Needs a running D-Bus session and an unlocked keyring (GNOME Keyring, KDE Wallet, or KeePassXC)
- On headless servers, start a D-Bus session manually:
  ```bash
  export $(dbus-launch)
  echo -n "" | gnome-keyring-daemon --unlock --daemonize --components=secrets
  ```
- **Fallback**: If `secret-tool` is unavailable, keysync falls back to an encrypted file store at `~/.config/keysync/store.json`

### Windows
- Uses Windows Credential Manager (Win32 API via `wincred` library) — no additional software needed
- Works on Windows 10+, desktop and server editions

## Initialize a project

```bash
cd /path/to/project
keysync init --project my-app
```

This creates `.keysync.json` in the current directory. The user can edit this file to configure platform integrations (see [Configure platforms](#configure-platforms)).

## Store secrets

```bash
# Global secret (shared across all projects)
keysync set GLOBAL_API_KEY=abc123

# Project-scoped secret (overrides global for this project)
keysync set -p my-app DB_URL=postgres://localhost:5432/mydb

# Environment-scoped secret (overrides project for this environment, default: production)
keysync set -p my-app DB_URL=postgres://prod-host/proddb --env production

# Different value for staging
keysync set -p my-app DB_URL=postgres://staging-host/stagingdb --env staging
```

When a project argument is provided (`-p` or `--project`), the secret is scoped to that project. Add `--env` (or `-e`) to further scope it to a specific environment (default: `production`). Otherwise it is stored globally.

The `set` command also pushes the secret to GitHub Secrets (if `gh` is authenticated and a repo is configured or auto-detected). If GitHub push fails, a warning is printed but the local store is not rolled back.

## Retrieve secrets

```bash
# Default: copy to clipboard
keysync get DATABASE_URL
# Output: Key DATABASE_URL copied to clipboard

# Print to stdout (for scripting)
keysync get DATABASE_URL --unmask
keysync get DATABASE_URL -u
```

Resolution order:
1. Environment-scoped secret (`--project` + `--env` match)
2. Project-scoped secret (`--project` match, no environment)
3. Global secret (fallback)

## List all secrets

```bash
# List key names only
keysync list

# List with values revealed
keysync list --unmask

# Filter by project
keysync list -p my-app

# Filter by project and environment
keysync list -p my-app -e staging
```

## Export secrets as environment variables

Same scope rules as `keysync get`. Prefer exporting a single key in scripts (one keychain read):

```bash
eval $(keysync export API_KEY)
eval $(keysync export DATABASE_URL -p my-app)
```

Export all matching secrets for full local dev:

```bash
eval $(keysync export -p my-app)
source <(keysync export -p my-app)
```

On macOS after building from source:

```bash
make build-signed && keysync trust
```

Project-scoped secrets override globals when both have the same key. Values are single-quoted for POSIX shell safety.

## Configure platforms

Edit `.keysync.json` to add platform integrations:

```json
{
  "repos": {
    "yourorg/my-app": {
      "project": "my-app",
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

Store the corresponding API tokens as global secrets in the keychain:

```bash
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...
```

For CI/CD environments where the keychain is unavailable, the tokens fall back to environment variables (`VERCEL_TOKEN`, `RAILWAY_TOKEN`, `SUPABASE_TOKEN`).

## Push secrets

```bash
# Sync all configured platforms + GitHub
keysync push -p my-app

# Sync specific platforms only
keysync push -p my-app --platforms vercel,railway
```

When syncing, secrets are resolved with three-level precedence:
1. Environment-scoped secrets (matching `--env`, default: `production`)
2. Project-scoped secrets (no environment)
3. Global secrets

Higher-precedence secrets with the same key override lower ones.

## Rotate a secret

```bash
keysync rotate DATABASE_URL
```

Generates a cryptographically random 44-character value and updates the local store, GitHub Secrets, and all configured platforms.

## Migrate from existing sources

```bash
# Import from .env file
keysync migrate --file .env

# Import from cloud platform (requires CLI to be installed + authenticated)
keysync migrate --cloud vercel
keysync migrate --cloud railway
keysync migrate --cloud supabase

# Preview without storing
keysync migrate --file .env --dry-run
```

## Diagnostics

```bash
keysync doctor
```

Checks that the config file is valid and the OS keychain store is operational.

## Run all tests

```bash
make test
```

This runs all unit tests (currently 97+ tests) covering the store backends, crypto layer, platform clients, config loading, and CLI commands.

## Client libraries

Keysync has 9 native client libraries that read secrets directly from the OS keychain with no dependency on the keysync binary:

| Language | Location | macOS | Linux | Windows |
|----------|----------|-------|-------|---------|
| **Go** | `clients/go/` | security CLI | secret-tool CLI | wincred library |
| **Python** | `clients/python/` | security CLI | secret-tool CLI | ctypes Win32 API |
| **TypeScript/Node** | `clients/node/` | security CLI | secret-tool CLI | PowerShell + Win32 |
| **Swift** | `clients/swift/` | Security.framework | secret-tool CLI | Not planned |
| **Java** | `clients/java/` | security CLI | secret-tool CLI | JNA → Win32 API |
| **C# (.NET)** | `clients/csharp/` | security CLI | secret-tool CLI | P/Invoke → Win32 API |
| **Rust** | `clients/rust/` | security CLI | secret-tool CLI | windows-sys → Win32 API |
| **C++** | `clients/cpp/` | security CLI | secret-tool CLI | Win32 API (wincred.h) |
| **Ruby** | `clients/ruby/` | security CLI | secret-tool CLI | PowerShell + inline C# |

Example usage:

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

## Troubleshooting

### "secret not found"
If `keysync get KEY` returns "secret not found", the key may be stored under a different scope or environment. Check with `keysync list` to see all stored secrets. Try adding `--project` or `--env` flags to narrow the search scope.

### GitHub sync warnings
If `keysync set` shows `warning: failed to write to GitHub`, the `gh` CLI may not be authenticated or the repo could not be auto-detected. Run `gh auth status` and ensure the repo flag is set: `keysync set KEY=val --repo owner/repo`.

### macOS Keychain prompts
If macOS repeatedly prompts for Keychain access, open *Keychain Access > (right-click your keychain) > Lock/Unlock*, or add your terminal app to *Always Allow*.

### Linux "secret-tool not found"
Install `libsecret-tools` (see [Linux setup](#linux) above). If running headless, ensure a D-Bus session is active and the keyring is unlocked.

### Fallback store used on Linux
If keysync falls back to the encrypted file store, `secret-tool` is unavailable. Install `libsecret-tools` and ensure D-Bus is running. The fallback store is encrypted with NaCl (Curve25519 + XSalsa20-Poly1305) and the key is stored at `~/.config/keysync/key`.

### Tests fail on macOS
If tests fail on macOS, the keychain may be locked. Run `make test` manually — the first invocation will prompt for Keychain access. Subsequent runs should pass silently.

## References

- [CLAUDE.md](../../CLAUDE.md) — Project instructions for Claude Code
- [AGENTS.md](../../AGENTS.md) — General AI agent instructions
- [docs/architecture.md](../../docs/architecture.md) — Architecture overview
- [docs/tests.md](../../docs/tests.md) — Test documentation
