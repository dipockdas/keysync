# keysync Rust Client — Claude Instructions

## Build & Test

```bash
cd clients/rust
cargo build           # Build the library
cargo test            # Run all tests
cargo test -- --nocapture  # Run tests with output
cargo doc --open      # Generate and open documentation
```

## Key files

```
src/
  lib.rs              # Public API (get_secret, list_secrets), re-exports
  error.rs            # KeySyncError enum, Result type, std::error::Error impl
  credential.rs       # CredentialEntry struct
  service.rs          # Service name builders and parsers
  macos.rs            # macOS: std::process::Command → security CLI
  linux.rs            # Linux: std::process::Command → secret-tool CLI
  windows.rs          # Windows: windows-sys → Win32 Credential Manager API
  unsupported.rs      # Other platforms: returns UnsupportedPlatform error
tests/
  (unit tests are inline in src/ modules via #[cfg(test)] modules)
Cargo.toml            # Package config, windows-sys dependency for Windows
```

## Conventions

- Platform selection via `#[cfg(target_os = "macos")]` / `#[cfg(target_os = "linux")]` / `#[cfg(target_os = "windows")]`
- Each platform module aliased as `platform` so the public API dispatches at compile time
- `unsupported.rs` catches all other platforms with `#[cfg(not(any(...)))]`
- `KeySyncError` enum with NotFound, KeychainError(String), UnsupportedPlatform variants
- `KeySyncError` implements std::fmt::Display and std::error::Error
- `From<std::io::Error>` conversion for seamless `?` operator usage
- `CredentialEntry` has scope, project, account fields
- All tests using `#[cfg(test)]` modules in each source file
- Use `std::env::var` to check environment variables first in get_secret

## Platform-specific details

### macOS (macos.rs)
- Uses `std::process::Command::new("security")` with args `find-generic-password -s <service> -a <account> -w`
- On macOS 13+, the password may appear in stderr instead of stdout — check both
- Exit code 44 indicates not found
- List: parses `security dump-keychain` output for genp records with keysync/ prefix

### Linux (linux.rs)
- Uses `std::process::Command::new("secret-tool")` with args `lookup service <svc> account <acct>`
- Exit code 1 indicates not found
- List: parses `secret-tool search service keysync` output, grouping by blank-line separators

### Windows (windows.rs)
- Uses `windows-sys` crate (version 0.52) with features `Win32_Security_Credentials` and `Win32_Foundation`
- Calls `CredReadW` and `CredFree` from advapi32.dll
- Target name format: `keysync_global`, `keysync_project_<name>` (slashes replaced with underscores)
- Uses `CredEnumerateW` with filter `keysync_*` for listing
- Wide string conversion via `encode_utf16()` and `String::from_utf16_lossy()`
- `#![allow(non_snake_case)]` for Windows API naming conventions

## Migration: replacing std::env::var with keysync

```rust
use keysync::get_secret;

// Before:
// let api_key = std::env::var("API_KEY").expect("API_KEY not set");

// After — global secret (shared across projects)
let api_key = get_secret("API_KEY", None)?;

// Project-scoped secret (falls back to global if no project match)
let db_url = get_secret("DATABASE_URL", Some("myapp"))?;

// List all secrets for a project (includes globals)
let entries = keysync::list_secrets(Some("myapp"))?;
```
