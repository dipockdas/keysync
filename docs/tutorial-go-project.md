# Using keysync with an Existing Go Project

This tutorial walks through adding keysync to an existing Go application —
storing secrets in the OS keychain, retrieving them at runtime, and syncing
them to GitHub Secrets and deployment platforms from your local machine.

## What you'll end up with

- Secrets stored in your OS keychain (macOS Keychain, Linux libsecret, or
  Windows Credential Manager), never in `.env` files
- Runtime secret retrieval via Go client library — your app reads secrets
  directly from the OS keychain on startup
- Secrets synced to GitHub Secrets from your local machine, accessible as `${{ secrets.NAME }}` in GitHub Actions workflows
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
  "repos": {
    "yourorg/my-api": {
      "project": "my-api",
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
go get github.com/dipockdas/keysync/clients/go@latest
```

## Step 4: Use secrets in your application

Replace hardcoded config or `.env` file reads with keysync lookups:

```go
package main

import (
    "fmt"
    "log"

    "github.com/dipockdas/keysync/clients/go"
)

func main() {
    // Retrieve a project-scoped secret
    stripeKey, err := keysync.GetSecret("my-api", "STRIPE_KEY")
    if err != nil {
        log.Fatalf("failed to get STRIPE_KEY: %v", err)
    }

    // Retrieve a global secret
    dbURL, err := keysync.GetGlobal("DATABASE_URL")
    if err != nil {
        log.Fatalf("failed to get DATABASE_URL: %v", err)
    }

    // Use the secrets (don't log them!)
    db, err := openDatabase(dbURL)
    // ...
}
```

**Key points:**
- `GetSecret(project, key)` retrieves project-scoped secrets with global fallback
- `GetGlobal(key)` retrieves global-only secrets
- The library calls the OS keychain directly (macOS Keychain, libsecret, Windows Credential Manager)
- No `.env` files, no secret values in your codebase
- Supports environment scoping: `GetEnvSecret(project, env, key)`

For testing, use the in-memory store:

```go
import (
    "context"
    "testing"

    "github.com/dipockdas/keysync/clients/go"
)

func TestMyHandler(t *testing.T) {
    ctx := context.Background()
    store := keysync.NewMemoryStore()
    store.Set(ctx, keysync.ScopeGlobal, "", "", "DATABASE_URL", "postgres://test:test@localhost/testdb")

    // Inject the store into your handler for testing
    handler := NewHandler(store)
    // ... test with handler
}
```

## Step 5: Push secrets to GitHub

Push your local secrets to GitHub Secrets so they're available in CI:

```bash
keysync push -p my-api
```

This reads secrets from your OS keychain and pushes them to:
1. GitHub Secrets (via `gh` CLI)
2. Configured deployment platforms (Vercel, Railway, etc.)

Now you can use the secrets in your GitHub Actions workflows:

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

      - name: Run tests
        run: go test ./...

      - name: Integration test with secrets
        run: go run ./cmd/migration
        env:
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

The secrets are available as `${{ secrets.SECRET_NAME }}` in all your workflows.

## Step 6: Configure deployment platforms

Add your platforms to `.keysync.json`:

```json
{
  "repos": {
    "yourorg/my-api": {
      "project": "my-api",
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

Store platform tokens in your keychain:

```bash
keysync set VERCEL_TOKEN=your_vercel_token
```

Then sync to GitHub and all platforms:

```bash
keysync push -p my-api
```

## Migration from `.env`

If your project currently uses a `.env` file, migrate with:

```bash
keysync migrate --file .env
```

This interactively prompts for each secret's scope. After migration, remove the
`.env` file and update your code to use the keysync client library instead.

## Next Steps

- Explore other [client libraries](../clients/README.md) for TypeScript, Python, Swift, Java, C#, Rust, C++, and Ruby
- Set up secret rotation with `keysync rotate`
- Configure multiple environments with `--env` flag
- See [migration guide](migration-guide.md) for more advanced migration scenarios

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
