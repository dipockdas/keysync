# KeySync Swift Client — Claude Instructions

## Build & Test

```bash
cd clients/swift
swift build           # Build the package
swift test            # Run all tests
```

## Key files

```
Sources/KeySync/
  KeySync.swift          # Error types, service name helpers
  KeySyncClient.swift    # Public API (getSecret, listSecrets)
  DarwinKeychain.swift   # macOS: Security.framework
  LinuxKeychain.swift    # Linux: Process + secret-tool
  WindowsKeychain.swift  # Windows: unsupported stub
Tests/KeySyncTests/
  KeySyncTests.swift     # Tests using Swift Testing framework
```

## Conventions

- Use `import Foundation` alongside `import Security` for CFString bridging
- Platform selection via `#if os(macOS) / #elseif os(Linux) / #else`
- `KeySyncError` for all error types (notFound, keychainError, unsupportedPlatform)
- Swift Testing framework (`@Test`, `#expect`) for tests
- Service naming: `keysync/global`, `keysync/project/<name>`
