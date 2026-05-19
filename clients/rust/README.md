# keysync Rust Client Library

Retrieve secrets managed by keysync directly from the OS keychain.
No dependency on the `keysync` binary — apps link directly against this
library to access secrets at runtime.

## Platform support

| Platform | Mechanism | Requirements | Status |
|----------|-----------|--------------|--------|
| macOS    | `security` CLI (built-in) | None | Ready |
| Linux    | `secret-tool` CLI (libsecret) | Requires `libsecret-tools` | Ready |
| Windows  | Win32 Credential Manager API | Requires `windows-sys` crate | Available (not fully tested on Windows) |

## Installation

Add this to your `Cargo.toml`:

```toml
[dependencies]
keysync = { path = "../clients/rust" }
```

Or from the workspace root:

```toml
[dependencies]
keysync = { path = "clients/rust" }
```

## Usage

```rust
use keysync::{get_secret, get_secret_with_env, list_secrets, list_secrets_with_env, KeySyncError};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Retrieve a project-scoped secret (falls back to global scope)
    match get_secret("DATABASE_URL", Some("my-api")) {
        Ok(val) => println!("DB URL: {}", val),
        Err(KeySyncError::NotFound) => eprintln!("Secret not found"),
        Err(e) => return Err(e.into()),
    }

    // Retrieve a global-only secret
    let api_key = get_secret("GLOBAL_API_KEY", None)?;

    // Retrieve an environment-scoped secret (falls back: env → project → global)
    let staging_db = get_secret_with_env("DATABASE_URL", "my-api", "staging")?;

    // List all secrets
    let entries = list_secrets(None)?;
    for entry in &entries {
        println!("{}/{} -> {}", entry.scope, entry.project, entry.account);
    }

    // List only project secrets (includes globals)
    let project_entries = list_secrets(Some("my-api"))?;

    // List environment secrets (includes globals + project)
    let staging_entries = list_secrets_with_env("my-api", Some("staging"))?;

    Ok(())
}
```

## How it works

Secrets are stored in the OS keychain with this naming convention:

| Scope   | Service/Target              | Account Name                  |
|---------|-----------------------------|-------------------------------|
| Global  | `keysync/global`            | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>`    | key name                       |
| Environment | `keysync/project/<name>/env/<env>` | key name                 |

The library calls the OS keychain tooling directly:

- **macOS**: `security find-generic-password -s keysync/global -a DATABASE_URL -w`
- **Linux**: `secret-tool lookup service keysync/global account DATABASE_URL`
- **Windows**: `CredReadW("keysync_global")` via the Win32 API

No subprocess chain to the keysync CLI. Read operations work standalone as
long as secrets have been stored (via `keysync set` or any other method that
writes to the same keychain entries).

## Environment variable fallback

`get_secret` checks the environment variable matching `key` first. If set, it
returns the value immediately without touching the OS keychain. This is the
primary path for:

- **Local development**: secrets injected via `eval $(keysync export)`
- **Cloud/CI deployments**: platforms inject environment variables directly

## Resolution order

For an environment-scoped call `get_secret_with_env("DATABASE_URL", "myapi", "staging")`:

1. Check `DATABASE_URL` environment variable
2. Check keychain service `keysync/project/myapi/env/staging` for account `DATABASE_URL`
3. Check keychain service `keysync/project/myapi` for account `DATABASE_URL`
4. Check keychain service `keysync/global` for account `DATABASE_URL`

For a project-scoped call `get_secret("DATABASE_URL", Some("myapi"))`:

1. Check `DATABASE_URL` environment variable
2. Check keychain service `keysync/project/myapi` for account `DATABASE_URL`
3. Check keychain service `keysync/global` for account `DATABASE_URL`

## Testing

Tests that interact with the OS keychain will run against the real keychain
on your development machine. Without keysync secrets stored, these tests
expect `NotFound` or `KeychainError` results.

To add secrets for testing:

```bash
keysync set DATABASE_URL "postgres://localhost/testdb"
keysync set --project myapi DATABASE_URL "postgres://localhost/myapi"
```

Then run the tests:

```bash
cd clients/rust
cargo test
```

## Testing

Tested on macOS (ARM64), Windows ARM64, and Windows AMD64 (x86-64). 45 unit
tests + 3 doc-tests, all passing.
