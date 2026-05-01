# @dipockdas/keysync — Node.js / TypeScript Client

Retrieve secrets managed by [keysync](https://github.com/dipockdas/keysync)
directly from the OS keychain, with no dependency on the `keysync` binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS    | `security` CLI (built-in) | Ready |
| Linux    | `secret-tool` CLI (libsecret) | Ready |
| Windows  | Companion helper binary (Go) | Planned |

## Installation

```bash
npm install @dipockdas/keysync
```

## Usage

```typescript
import { getSecret, listSecrets } from "@dipockdas/keysync";

// Project-scoped secret with global fallback
const dbUrl = await getSecret("DATABASE_URL", "my-api");

// Global-only secret
const apiKey = await getSecret("GLOBAL_API_KEY");

// List all secrets
const secrets = await listSecrets();
for (const s of secrets) {
  console.log(`${s.scope}/${s.project ?? ""} => ${s.key}`);
}

// Filter by scope and project
const projectSecrets = await listSecrets("project", "my-api");
```

## Error handling

```typescript
import { getSecret, KeySyncError } from "@dipockdas/keysync";

try {
  const val = await getSecret("DATABASE_URL", "my-api");
} catch (err) {
  if (err instanceof KeySyncError) {
    switch (err.code) {
      case "notFound":
        // Secret doesn't exist in any scope
        break;
      case "keychainError":
        // OS-level keychain error
        break;
      case "unsupportedPlatform":
        // Platform not yet supported
        break;
    }
  }
}
```

## How it works

Secrets are stored in the OS keychain with this naming convention:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name |

The library shells out to the OS keychain tooling directly:

- **macOS**: `security find-generic-password -s keysync/global -a DATABASE_URL -w`
- **Linux**: `secret-tool lookup service keysync/global account DATABASE_URL`
- **Windows**: Not yet available (use Go or Python client)

No dependency on the `keysync` binary. Read operations work standalone as long
as secrets have been stored (via `keysync set` or any other method that writes
to the same keychain entries).
