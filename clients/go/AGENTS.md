# keysync Go Client

## Overview

The Go client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary. Apps link directly against this
library to access secrets at runtime.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via os/exec | Ready |
| Linux | `secret-tool` CLI via os/exec | Ready |
| Windows | `github.com/danieljoos/wincred` | Ready |

## API

```go
GetSecret(key, project, environment string) (string, error)
ListSecrets(scope, project, environment string) ([]SecretEntry, error)
```

## Testing

Use `MemoryStore` to test without a real keychain:

```go
store := keysync.NewMemoryStore()
store.SetSecret(ctx, "global", "", "MY_KEY", "my-val")
```

## Key design decisions

- Build tags select the platform implementation at compile time — no runtime
  platform detection needed.
- The Go client is the only library that can **write** to the keychain. All
  other clients are read-only.
- Windows uses `wincred` (a pure Go Win32 library) rather than shelling out
  to cmdkey, which can't retrieve secret values.
