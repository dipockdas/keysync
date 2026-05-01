# KeySync — Swift Client Library

Retrieve secrets managed by [keysync](https://github.com/dipockdas/keysync)
directly from the OS keychain, with no dependency on the `keysync` binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS    | Security.framework (`SecItemCopyMatching`) | Ready |
| Linux    | `secret-tool` CLI (libsecret) | Ready |
| Windows  | Not yet supported in Swift | Stub throws `unsupportedPlatform` |

## Requirements

- Swift 5.9+
- macOS 13+ or Linux with `libsecret-tools` installed

## Installation

### Swift Package Manager

Add to your `Package.swift`:

```swift
dependencies: [
    .package(url: "https://github.com/dipockdas/keysync.git", branch: "main"),
]
```

Then add `"KeySync"` to your target's dependencies.

## Usage

```swift
import KeySync

// Retrieve a project-scoped secret (falls back to global scope)
let dbURL = try KeySync.getSecret("DATABASE_URL", project: "my-api")

// Retrieve a global-only secret
let apiKey = try KeySync.getSecret("GLOBAL_API_KEY")

// List all secrets
let secrets = try KeySync.listSecrets()
for secret in secrets {
    print("\(secret.scope)/\(secret.project ?? "") => \(secret.key)")
}

// Filter list by scope and project
let projectSecrets = try KeySync.listSecrets(scope: "project", project: "my-api")
```

## Error handling

```swift
do {
    let value = try KeySync.getSecret("DATABASE_URL", project: "my-api")
    // use value
} catch KeySyncError.notFound {
    // Secret doesn't exist in any scope
} catch KeySyncError.keychainError(let message) {
    // OS-level keychain error
} catch {
    // Other errors
}
```

## How it works

Secrets are stored in the OS keychain with this naming convention:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name |

On **macOS**, this library calls `Security.framework` directly via
`SecItemCopyMatching` — no subprocess, no shelling out. The secret value
never leaves your process's address space.

On **Linux**, it shells out to `secret-tool lookup` since there's no native
Swift binding for libsecret. This matches what the keysync CLI itself does.
