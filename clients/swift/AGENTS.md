# KeySync Swift Client

## Overview

The Swift client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | Security.framework (`SecItemCopyMatching`) | Ready |
| Linux | `secret-tool` CLI via `Process` | Ready |
| Windows | Stub | Throws `unsupportedPlatform` |

## API

```swift
KeySync.getSecret(_ key: String, project: String? = nil) throws -> String
KeySync.listSecrets(scope: String? = nil, project: String? = nil) throws -> [(scope: String, project: String?, key: String)]
```

## Key design decisions

- macOS uses **native Security.framework** — no subprocess, secret never leaves
  process memory. This is the best integration of any client library.
- Linux falls back to `secret-tool` since there's no native Swift libsecret binding.
- Platform selection is compile-time via `#if os()` conditionals.
- Service naming matches keysync convention: `keysync/global` and `keysync/project/<name>`.
