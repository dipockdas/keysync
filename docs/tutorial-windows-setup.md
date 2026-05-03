# Tutorial: Using keysync on Windows

This tutorial covers building, configuring, and using keysync on Windows. The Windows Credential Manager is the OS-native secret store, and keysync integrates with it through the `wincred` Go library (Win32 API, no CGo required).

## Prerequisites

- **Windows 10 or later** (desktop or server edition)
- **Go 1.25+** — download from [go.dev](https://go.dev/dl/)
- **Git for Windows** — `git` on your PATH
- **GitHub CLI (`gh`)** — optional, for GitHub Secrets integration
- **Python 3.11+** — optional, for the Python client library

## Step 1: Install Go

1. Download and run the Go installer from [go.dev](https://go.dev/dl/).
2. Verify the installation:

```powershell
go version
# Output: go version go1.25.X windows/amd64
```

## Step 2: Build keysync

```powershell
git clone https://github.com/dipockdas/keysync.git
cd keysync
go build -o ./bin/keysync.exe ./cmd/keysync
```

The binary is produced at `.\bin\keysync.exe`. Add it to your PATH:

```powershell
# Add to PATH for the current session
$env:Path += ";$pwd\bin"

# Make it permanent (adds to your PowerShell profile)
$newPath = Join-Path (Get-Location) "bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $newPath, [EnvironmentVariableTarget]::User)
```

Open a new PowerShell window and verify:

```powershell
keysync --help
```

## Step 3: Understand Windows Credential Manager

Windows stores secrets in the **Credential Manager**, which has two sections:

- **Windows Credentials** — where keysync stores its secrets
- **Web Credentials** — browser passwords (not used by keysync)

### How keysync stores secrets on Windows

Keysync uses the `wincred` Go library to interact with the Win32 Credential Manager API directly. Secrets are stored as **Generic Credentials** with these conventions:

| Scope | Target Name |
|-------|-------------|
| Global | `keysync_global` |
| Project | `keysync_project_<name>` |
| Project + Environment | `keysync_project_<name>_<env>` |

The Credential's `UserName` field holds the secret key, and `CredentialBlob` holds the secret value. All credentials are persisted with `Type = Generic` and persist to the local machine.

### Viewing stored credentials

Via Control Panel:
1. Open **Control Panel > Credential Manager > Windows Credentials**
2. Look for entries prefixed with `keysync_`

Via PowerShell:

```powershell
# List all keysync credentials using the built-in cmdkey tool
cmdkey /list | Select-String "keysync"

# View a specific credential
cmdkey /list | Select-String "keysync_global"
```

> **Note**: `cmdkey` can list and delete credentials but cannot create them. Keysync uses the Win32 API directly (via `wincred`) for full read/write access.

## Step 4: Initialize and store secrets

```powershell
# Change to your project directory
cd C:\Projects\my-app

# Initialize keysync
keysync init --project my-app

# Store platform tokens as global secrets
keysync set VERCEL_TOKEN=...
keysync set RAILWAY_TOKEN=...
keysync set SUPABASE_TOKEN=...

# Store a project-scoped secret
keysync set -p my-app DATABASE_URL=postgresql://dbhost:5432/myapp

# Store an environment-scoped secret
keysync set -p my-app DATABASE_URL=postgresql://prod-host:5432/proddb --env production
```

### PowerShell integration

Add convenient aliases to your PowerShell profile (`$PROFILE`):

```powershell
# Add to $PROFILE
function ks-get { keysync get $args }
function ks-set { keysync set $args }
function ks-list { keysync list $args }

# Environment export helper
function ks-export {
    $output = keysync export @args
    if ($LASTEXITCODE -eq 0) {
        $output -split "`n" | ForEach-Object {
            if ($_ -match '^export\s+(\w+)=''(.*)''$') {
                [Environment]::SetEnvironmentVariable($matches[1], $matches[2], "Process")
                Write-Output "Set $($matches[1])"
            }
        }
    }
}
```

Reload your profile:

```powershell
. $PROFILE
```

Now use the aliases:

```powershell
ks-list -p my-app
ks-export --project my-app
```

## Step 5: Using the Python client on Windows

The Python client has the best Windows support among all client libraries — it uses pure `ctypes` Win32 API calls with no external dependencies.

```powershell
# Install the Python client
pip install C:\path\to\keysync\clients\python

# Or install from the keysync repo
cd keysync
pip install ./clients/python
```

### Sample usage

Create `config.py`:

```python
import os
from keysync import get_secret, SecretNotFoundError

PROJECT = "my-app"

def load_database_url():
    """Load DATABASE_URL from keychain, falling back to env var."""
    try:
        return get_secret("DATABASE_URL", project=PROJECT)
    except SecretNotFoundError:
        return os.environ.get("DATABASE_URL", "")

if __name__ == "__main__":
    url = load_database_url()
    print(f"Database URL: {url.split('@')[-1] if '@' in url else 'set'}")
```

The Python client uses `ctypes` to call `CredReadW` and `CredEnumerateW` directly from `advapi32.dll` — no compiled extensions, no external packages.

## Step 6: Using the Go client on Windows

```powershell
# Add the Go client to your module
go get github.com/dipockdas/keysync/clients/go
```

```go
package main

import (
    "fmt"
    "log"
    keysync "github.com/dipockdas/keysync/clients/go"
)

func main() {
    dbURL, err := keysync.GetSecret("DATABASE_URL", "my-app")
    if err != nil {
        log.Fatalf("failed to get secret: %v", err)
    }
    fmt.Println("Database URL resolved")
    _ = dbURL
}
```

The Go client uses `github.com/danieljoos/wincred` which calls the Win32 API via `golang.org/x/sys/windows` — no CGo, cross-compilation friendly.

## Step 7: CI/CD with GitHub Actions on Windows

```yaml
name: Sync Secrets
on:
  push:
    branches: [main]

jobs:
  sync:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Build keysync
        run: go build -o ./bin/keysync.exe ./cmd/keysync
      - name: Sync all platforms
        run: ./bin/keysync sync
        shell: pwsh
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
          SUPABASE_TOKEN: ${{ secrets.SUPABASE_TOKEN }}
```

> **Note**: In GitHub Actions, the platform tokens are provided as environment variables since there is no OS keychain available in CI runners.

## Troubleshooting

### "wincred: not found" or credential access errors

Ensure you're running PowerShell or Command Prompt as the same user who stored the credentials. Credential Manager is per-user.

### Cannot find secrets with `cmdkey /list`

`cmdkey` only shows a subset of credential types. Use the Python or Go client's `list_secrets()` function instead, or open **Credential Manager > Windows Credentials** and look for `keysync_*` entries.

### Build fails with CGo errors

Keysync does not require CGo on Windows. If you see CGo-related errors, ensure you're not setting `CGO_ENABLED=1`:

```powershell
go env CGO_ENABLED
# Should output: 0
go build -o ./bin/keysync.exe ./cmd/keysync
```

### PowerShell execution policy blocks scripts

If you get an execution policy error when loading your profile:

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

## Summary

In this tutorial you:

1. Built keysync from source on Windows
2. Stored and retrieved secrets via the Windows Credential Manager
3. Set up PowerShell aliases for daily use
4. Used both the Python and Go client libraries on Windows
5. Configured a GitHub Actions workflow for Windows runners

Keysync on Windows provides full Credential Manager integration through the `wincred` library (Go client) and `ctypes` (Python client) — no CGo, no external DLLs, no manual configuration required.
