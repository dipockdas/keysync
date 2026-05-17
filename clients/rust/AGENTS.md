# keysync Rust Client — Agent Instructions

## Quick start

```bash
cd /Users/dipockdas/Projects/keysync/clients/rust
cargo build
cargo test
```

## Module structure

- `lib.rs` — Public API entry point, cfg-conditional platform module imports
- `error.rs` — Error type definitions and conversions
- `credential.rs` — Data types for credential entries
- `service.rs` — Service name construction and parsing
- `macos.rs` — macOS security CLI integration
- `linux.rs` — Linux secret-tool CLI integration
- `windows.rs` — Windows Credential Manager API integration
- `unsupported.rs` — Catch-all for unsupported platforms
