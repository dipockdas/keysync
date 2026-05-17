# keysync Rust Client — Agent Instructions

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via std::process::Command | Ready |
| Linux | `secret-tool` CLI via std::process::Command | Ready |
| Windows | `windows-sys` → Win32 API (CredReadW / CredEnumerateW) | Available (not fully tested on Windows) |

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
