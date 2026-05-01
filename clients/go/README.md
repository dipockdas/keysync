# keysync Go Client Library

Retrieve secrets managed by keysync directly from the OS keychain.
No dependency on the `keysync` binary — apps link directly against this
library to access secrets at runtime.

## Platform support

| Platform | Mechanism | |
|----------|-----------|-|
| macOS    | `security` CLI (built-in) | |
| Linux    | `secret-tool` CLI (libsecret) | Requires `libsecret-tools` |
| Windows  | `wincred` Go library (Win32 API) | No CGo required |

## Installation

```bash
go get github.com/dipockdas/keysync/clients/go
```

## Usage

```go
import "github.com/dipockdas/keysync/clients/go"

// Retrieve a project-scoped secret (falls back to global scope)
dbURL, err := keysync.GetSecret("DATABASE_URL", "my-api")
if err == keysync.ErrNotFound {
    // secret doesn't exist
}

// Retrieve a global-only secret
apiKey, err := keysync.GetSecret("GLOBAL_API_KEY", "")

// List all secrets
entries, err := keysync.ListSecrets("", "")
for _, e := range entries {
    fmt.Printf("%s/%s => %s\n", e.Scope, e.Project, e.Key)
}

// Filter by scope and project
projectEntries, err := keysync.ListSecrets("project", "my-api")
```

## Testing

Use `MemoryStore` to avoid OS keychain dependencies in unit tests:

```go
import (
    "context"
    "github.com/dipockdas/keysync/clients/go"
)

func TestMyHandler(t *testing.T) {
    store := keysync.NewMemoryStore()
    ctx := context.Background()
    store.SetSecret(ctx, "global", "", "DATABASE_URL", "postgres://test:test@localhost/testdb")

    // Inject store into your handler
    handler := NewHandler(store)
    // ...
}
```

## How it works

Secrets are stored in the OS keychain with this naming convention:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name |

The library calls the OS keychain tooling directly:

- **macOS**: `security find-generic-password -s keysync/global -a DATABASE_URL -w`
- **Linux**: `secret-tool lookup service keysync/global account DATABASE_URL`
- **Windows**: `wincred.GetGenericCredential("keysync_global")`

No subprocess chain, no dependency on the keysync CLI. Read operations work
standalone as long as secrets have been stored (via `keysync set` or any other
method that writes to the same keychain entries).
