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

// Retrieve an environment-scoped secret (falls back to project → global)
let envDBURL = try KeySync.getSecret("DATABASE_URL", project: "my-api", environment: "staging")

// Retrieve a project-scoped secret (falls back to global scope)
let dbURL = try KeySync.getSecret("DATABASE_URL", project: "my-api")

// Retrieve a global-only secret
let apiKey = try KeySync.getSecret("GLOBAL_API_KEY")

// List all secrets
let secrets = try KeySync.listSecrets()
for secret in secrets {
    print("\(secret.scope)/\(secret.project ?? "")/\(secret.environment ?? "") => \(secret.key)")
}

// Filter list by scope, project, and environment
let stagingSecrets = try KeySync.listSecrets(scope: "project", project: "my-api", environment: "staging")
```

## Error handling

```swift
do {
    let value = try KeySync.getSecret("DATABASE_URL", project: "my-api", environment: "staging")
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
| Environment | `keysync/project/<name>/env/<env>` | key name |

### Resolution order for `getSecret`

When `project` and `environment` are both provided, secrets are resolved in
this order:

1. **Environment variable** -- checks `ProcessInfo.processInfo.environment`
   first. This is the primary path for local development (`eval $(keysync
   export)`) and cloud/CI deployments where platforms inject environment
   variables directly.

2. **Environment scope** -- checks `keysync/project/<project>/env/<env>` in
   the OS keychain. Use this for environment-specific overrides (e.g. a
   different `DATABASE_URL` for staging vs. production).

3. **Project scope** -- checks `keysync/project/<project>` in the OS
   keychain. This is the project-level default.

4. **Global scope** -- checks `keysync/global` in the OS keychain. This is
   the last resort fallback.

On **macOS**, this library calls `Security.framework` directly via
`SecItemCopyMatching` — no subprocess, no shelling out. The secret value
never leaves your process's address space.

On **Linux**, it shells out to `secret-tool lookup` since there's no native
Swift binding for libsecret. This matches what the keysync CLI itself does.

## Testing

Tested on macOS (ARM64). 19 tests, all passing. Swift is macOS/Linux only;
not available on Windows. Run with:

```bash
cd clients/swift
swift test
```
