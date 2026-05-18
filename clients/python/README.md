# keysync — Python Client Library

Retrieve secrets managed by [keysync](https://github.com/dipockdas/keysync)
directly from the OS keychain, with no dependency on the `keysync` binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS    | `security` CLI (built-in) | Ready |
| Linux    | `secret-tool` CLI (libsecret) | Ready |
| Windows  | ctypes Win32 API (Credential Manager) | Ready |

## Requirements

- Python 3.11+
- **macOS**: No additional dependencies
- **Linux**: `libsecret-tools` (`sudo apt-get install libsecret-tools`)
- **Windows**: No additional dependencies (uses built-in `ctypes`)

## Installation

```bash
pip install keysync
```

## Usage

```python
from keysync import get_secret, list_secrets, SecretNotFoundError

# Project-scoped secret with global fallback
db_url = get_secret("DATABASE_URL", project="my-api")

# Environment-scoped secret (per-environment overrides)
db_url_dev = get_secret("DATABASE_URL", project="my-api", environment="dev")

# Global-only secret
api_key = get_secret("GLOBAL_API_KEY")

# List all secrets
secrets = list_secrets()
for s in secrets:
    print(f"{s['scope']}/{s['project'] or ''} => {s['key']}")

# Filter by scope, project, and environment
project_secrets = list_secrets(scope="project", project="my-api")
dev_secrets = list_secrets(scope="project", project="my-api", environment="dev")
```

## Error handling

```python
from keysync import get_secret, SecretNotFoundError, KeySyncError

try:
    val = get_secret("DATABASE_URL", project="my-api")
except SecretNotFoundError:
    print("Secret doesn't exist")
except KeySyncError as e:
    if e.code == "keychainError":
        print(f"Keychain error: {e}")
    elif e.code == "unsupportedPlatform":
        print(f"Unsupported platform: {e}")
```

## How it works

Secrets are stored in the OS keychain with this naming convention:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name |
| Environment | `keysync/project/<name>/env/<env>` | key name |

On Windows, `/env/` is stripped from the target name stored in Credential
Manager (e.g. `keysync/project/myapp/env/dev` becomes `keysync_project_myapp_dev`).

## Resolution order

`get_secret` searches for a secret in the following order:

1. Environment variable (`os.environ`) -- always checked first
2. `keysync/project/<project>/env/<environment>` -- if both *project* and *environment* are provided
3. `keysync/project/<project>` -- if *project* is provided
4. `keysync/global` -- always falls back to global scope

This allows per-environment overrides while still having a project-level default
and a global fallback.

The library accesses the OS keychain directly:

- **macOS**: `security find-generic-password` via subprocess
- **Linux**: `secret-tool lookup` via subprocess
- **Windows**: `CredReadW` / `CredEnumerateW` via ctypes (no DLLs required)

No dependency on the `keysync` binary. Read operations work standalone.
