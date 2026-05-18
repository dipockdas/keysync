# @dipockdas/keysync — Node.js / TypeScript Client

Retrieve secrets managed by [keysync](https://github.com/dipockdas/keysync)
directly from the OS keychain, with no dependency on the `keysync` binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS    | `security` CLI (built-in) | Ready |
| Linux    | `secret-tool` CLI (libsecret) | Ready |
| Windows  | PowerShell with inline C# (CredReadW/CredEnumerateW) | Available (not fully tested on Windows) |

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

// Environment-specific secret (checks env scope first, then project, then global)
const apiUrl = await getSecret("API_URL", "my-api", "production");

// List all secrets
const secrets = await listSecrets();
for (const s of secrets) {
  console.log(`${s.scope}/${s.project ?? ""}/${s.environment ?? ""} => ${s.key}`);
}

// Filter by scope and project
const projectSecrets = await listSecrets("project", "my-api");

// Filter by environment
const prodSecrets = await listSecrets("project", "my-api", "production");
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
| Project + Environment | `keysync/project/<name>/env/<env>` | key name |

The library shells out to the OS keychain tooling directly:

- **macOS**: `security find-generic-password -s keysync/global -a DATABASE_URL -w`
- **Linux**: `secret-tool lookup service keysync/global account DATABASE_URL`
- **Windows**: PowerShell with inline C# calling CredReadW / CredEnumerateW from advapi32.dll

No dependency on the `keysync` binary. Read operations work standalone as long
as secrets have been stored (via `keysync set` or any other method that writes
to the same keychain entries).
