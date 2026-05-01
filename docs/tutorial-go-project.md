# Using keysync with an Existing Go Project

This tutorial walks through adding keysync to an existing Go application —
storing secrets in the OS keychain, retrieving them at runtime, and syncing
them to GitHub Actions and deployment platforms.

## What you'll end up with

- Secrets stored in your OS keychain (macOS Keychain, Linux libsecret, or
  Windows Credential Manager), never in `.env` files
- Runtime secret retrieval via Go client library — your app reads secrets
  directly from the OS keychain on startup
- Secrets synced to GitHub Actions on push, accessible as environment variables
  in CI
- Secrets pushed to Vercel, Railway, or Supabase for deployment

## Prerequisites

- Go 1.25+
- A Go project with a `go.mod` file
- `gh` CLI installed and authenticated (`gh auth login`)
- `make` (optional, for the Makefile helpers)

## Step 1: Initialize keysync in your project

```bash
# Install keysync
git clone https://github.com/dipockdas/keysync.git
cd keysync && make build
cp ./bin/keysync /usr/local/bin/keysync

# Go back to your project and initialize config
cd /path/to/your-project
keysync init --project my-api
```

This creates a `.keysync.json` file:

```json
{
  "projects": {
    "my-api": {
      "platforms": {}
    }
  }
}
```

Commit this file to your repository so other developers and CI can use it.

## Step 2: Store your first secrets

```bash
# Store a database URL as a global secret (shared across all projects)
keysync set DATABASE_URL=postgres://user:pass@host:5432/mydb

# Store an API key as a project-scoped secret (only for my-api)
keysync set -p my-api STRIPE_KEY=sk_live_abc123
```

These go into your OS keychain. Verify with:

```bash
keysync list
```

You should see:

```
  Scope    Project    Key
  global              DATABASE_URL
  project  my-api     STRIPE_KEY
```

## Step 3: Add the keysync Go client dependency

The client library uses build tags to call the OS keychain directly on each
platform — no `keysync` binary dependency at runtime.

```bash
go get github.com/dipockdas/keysync/client@latest
```

**Note:** The client library is currently being migrated from the `client/`
package to `clients/go/` with the hybrid architecture (direct OS keychain access
per platform instead of shelling out to `keysync get`). See
[clients/README.md](../clients/README.md) for the plan.

## Step 4: Use secrets in your application

Replace hardcoded config or `.env` file reads with keysync lookups:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/dipockdas/keysync/client"
)

func main() {
    // Retrieve a secret at startup
    dbURL, err := client.GetSecret("my-api", "DATABASE_URL")
    if err != nil {
        log.Fatalf("failed to get DATABASE_URL: %v", err)
    }

    // Use the secret (don't log it!)
    db, err := openDatabase(dbURL)
    // ...
}
```

**Key points:**
- `GetSecret` takes `(project, key)` — project can be `""` for global-only
- Project-scoped secrets override global secrets with the same key
- The function shells out to the OS keychain tool on each platform
- No `.env` files, no secret values in your codebase

For testing, use the in-memory store:

```go
func TestMyHandler(t *testing.T) {
    store := client.NewMemoryStore()
    store.SetSecret(context.Background(), "global", "", "DATABASE_URL", "postgres://test:test@localhost/testdb")

    // Inject the store into your handler
    handler := NewHandler(store)
    // ... test with handler
}
```

## Step 5: Set up GitHub Actions CI

Add a workflow step that makes secrets available in CI. GitHub doesn't expose
secret values via API, but `keysync sync` pushes them as GitHub Actions secrets
on push to main.

```yaml
# .github/workflows/ci.yml
name: CI
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Install libsecret (Linux keychain)
        run: sudo apt-get install -y libsecret-tools

      - name: Run tests
        run: go test ./...

      - name: Integration test with secrets
        run: |
          # Run a database test that needs DATABASE_URL
          # The secret was set by keysync sync on main branch
          go run ./cmd/migration
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

Add the sync workflow:

```yaml
# .github/workflows/sync-secrets.yml
name: Sync Secrets
on:
  push:
    branches: [main]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Build keysync
        run: |
          git clone https://github.com/dipockdas/keysync.git /tmp/keysync
          cd /tmp/keysync && go build -o /tmp/keysync/bin/keysync ./cmd/keysync
      - name: Sync secrets
        run: /tmp/keysync/bin/keysync sync -p my-api
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
          SUPABASE_TOKEN: ${{ secrets.SUPABASE_TOKEN }}
```

## Step 6: Configure deployment platforms

Add your platforms to `.keysync.json`:

```json
{
  "projects": {
    "my-api": {
      "platforms": {
        "vercel": {
          "projectId": "prj_xxxxx",
          "target": ["production"]
        }
      }
    }
  }
}
```

Set platform tokens as local environment variables:

```bash
export VERCEL_TOKEN=your_vercel_token
```

Then sync locally:

```bash
keysync sync -p my-api
```

Or let CI handle it on push to main.

## Migration from `.env`

If your project currently uses a `.env` file, migrate with:

```bash
keysync migrate --file .env
```

This interactively prompts for each secret's scope. After migration, remove the
`.env` file and update your code to use the keysync client library instead.

## Complete example

See the [example-go-app](../examples/go-app/) directory for a complete working
example with:
- Application code using `client.GetSecret`
- Unit tests with `client.MemoryStore`
- Makefile with `make build`, `make test`
- GitHub Actions workflow for CI and sync

## Troubleshooting

**"secret not found" error during `GetSecret`:**
Make sure you stored the secret with `keysync set`, and that you're passing the
correct project name. Try `keysync list` to verify.

**"exec: security: executable file not found" on Linux:**
Install `libsecret-tools`:
```bash
sudo apt-get install libsecret-tools
```

**Tests fail because keychain isn't available in CI:**
Use `client.NewMemoryStore()` in your tests to mock the keychain. The Go client
library's `MemoryStore` implements the same `Store` interface as the real
keychain access.
