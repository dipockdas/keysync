# Clients — Native Language Libraries

This directory contains per-language libraries that retrieve secrets directly from
the OS keychain, with **no dependency on the `keysync` binary**. Each library
calls the native OS secret storage tooling directly, using the same service-naming
convention that `keysync` uses internally.

## Design

### How secrets are stored

Keysync stores secrets in the OS keychain with a consistent naming scheme:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name (e.g. `DATABASE_URL`) |

When looking up a secret, project scope takes precedence over global scope.

### Per-platform access

Each library uses the platform's native tooling rather than wrapping the `keysync` binary:

| Platform | Tool / API | Get Command |
|----------|-----------|-------------|
| macOS | `security` CLI (built-in) | `security find-generic-password -s keysync/global -a KEY -w` |
| Linux | `secret-tool` CLI (libsecret) | `secret-tool lookup service keysync/global account KEY` |
| Windows | `wincred` Go library | Native Win32 API via `github.com/danieljoos/wincred` |
| Swift (macOS) | `Security.framework` | Native Swift API (no subprocess) |

### Why direct OS access instead of shelling out to `keysync get`

- **No binary dependency** — the keysync CLI is only needed for write operations
  (`set`, `sync`, `rotate`, etc.). Read operations work standalone.
- **No subprocess overhead** — secrets are retrieved in-process, critical for apps
  that load many secrets at startup.
- **Tighter security boundary** — secret values never cross a pipe or process
  boundary at read time.
- **Swift on macOS can use native Security framework** — the cleanest integration
  of all, using `SecItemCopyMatching` directly without shelling out.

### Directory layout

```
clients/
├── README.md               # This file
├── go/                     # Go client library
├── node/                   # Node.js / TypeScript package
├── python/                 # Python package
└── swift/                  # Swift Package Manager package
```

---

## Go library — `clients/go/`

Currently at `client/` in the project root. Will be migrated to `clients/go/`.

**Conventions to follow:**

```
clients/go/
├── client.go               # Public API: GetSecret, GetSecretContext
├── client_darwin.go        # macOS: exec security CLI
├── client_linux.go         # Linux: exec secret-tool CLI
├── client_windows.go       # Windows: wincred Go library
├── store.go                # Store interface
├── memory_store.go         # In-memory test store
├── client_test.go          # Cross-platform tests
└── go.mod / go.sum
```

**API:**

```go
package keysync

// GetSecret retrieves a secret from the OS keychain.
// Falls back from project scope to global scope.
func GetSecret(project, key string) (string, error)

// ListSecrets returns all key names for the given scope/project.
func ListSecrets(scope, project string) ([]string, error)
```

**Platform implementations:**

- macOS: `exec.Command("security", "find-generic-password", "-s", service, "-a", key, "-w")`
- Linux: `exec.Command("secret-tool", "lookup", "service", service, "account", key)`
- Windows: `wincred.GetGenericCredential(target)` with `UserName` matching key

**Key differences from current `client/client.go`:**
- Current: shells out to `keysync get KEY`
- New: shells out to OS keychain tools directly
- New: adds `ListSecrets` support
- New: build-tagged platform files for `darwin`, `linux`, `windows`

---

## Node.js / TypeScript library — `clients/node/`

Single TypeScript package that compiles to JavaScript. Ships both `.ts` source
and compiled `.js` for consumers who want either.

```
clients/node/
├── src/
│   ├── index.ts             # Public API
│   ├── darwin.ts            # macOS: exec security CLI
│   ├── linux.ts             # Linux: exec secret-tool CLI
│   ├── windows.ts           # Windows: exec powershell/cmdkey or wincred helper
│   └── types.ts             # Type definitions
├── src/__tests__/           # Tests
├── package.json
├── tsconfig.json
├── .gitignore
└── README.md
```

**API:**

```typescript
// Get a secret. Falls back from project scope to global scope.
function getSecret(key: string, project?: string): Promise<string>

// List all key names for a scope/project.
function listSecrets(scope?: string, project?: string): Promise<string[]>
```

**Platform implementations:**

```typescript
// darwin.ts
import { execFile } from 'child_process';
// execFile('security', ['find-generic-password', '-s', service, '-a', key, '-w'])

// linux.ts
import { execFile } from 'child_process';
// execFile('secret-tool', ['lookup', 'service', service, 'account', key])

// windows.ts
// Option A: exec cmdkey (limited, no value extraction)
// Option B: ship a tiny Go helper binary that wraps wincred
// Option C: use node-wincred native addon (npm package)
```

**Windows consideration:** Node.js can't easily call Win32 API without native
addons. Options:
1. Shell out to `cmdkey /list` for enumeration, but `cmdkey` can't retrieve values.
2. Use `node-wincred` npm package (native addon, requires build toolchain).
3. Bundle a small Go binary (`wincred-helper`) compiled from shared code.

Recommendation: start with option 3 (tiny Go helper), migrate to option 2
when the package matures.

---

## Python library — `clients/python/`

Standard Python package with platform-specific modules.

```
clients/python/
├── src/
│   └── keysync/
│       ├── __init__.py       # Public API
│       ├── _darwin.py        # macOS: subprocess security CLI
│       ├── _linux.py         # Linux: subprocess secret-tool CLI
│       └── _windows.py       # Windows: ctypes Win32 API or subprocess helper
├── tests/
├── pyproject.toml
├── .gitignore
└── README.md
```

**API:**

```python
def get_secret(key: str, project: str | None = None) -> str:
    """Retrieve a secret from the OS keychain.
    Falls back from project scope to global scope.
    Raises SecretNotFoundError if not found.
    """

def list_secrets(scope: str | None = None, project: str | None = None) -> list[str]:
    """List all key names matching the given scope/project."""
```

**Platform implementations:**

```python
# _darwin.py
import subprocess
# subprocess.run(['security', 'find-generic-password', '-s', service, '-a', key, '-w'],
#                capture_output=True, text=True, check=True)

# _linux.py
import subprocess
# subprocess.run(['secret-tool', 'lookup', 'service', service, 'account', key],
#                capture_output=True, text=True, check=True)

# _windows.py
import ctypes
from ctypes import wintypes
# Call CredEnumerateW / CredReadW directly via ctypes
```

**Windows consideration:** Python can call Win32 API directly via `ctypes`
(or the `win32cred` PyPI package), making it the cleanest non-Go Windows
experience. No helper binary needed.

---

## Swift library — `clients/swift/`

Swift Package Manager package. macOS uses native `Security.framework` via
`SecItemCopyMatching` — no subprocess needed. Linux falls back to `secret-tool`.

```
clients/swift/
├── Sources/
│   └── KeySync/
│       ├── KeySync.swift          # Public API
│       ├── DarwinKeychain.swift   # macOS: Security.framework
│       ├── LinuxKeychain.swift    # Linux: Process + secret-tool
│       └── WindowsKeychain.swift  # Windows: Process + helper or WinSDK
├── Tests/
│   └── KeySyncTests/
├── Package.swift
└── README.md
```

**API:**

```swift
public struct KeySync {
    /// Retrieve a secret from the OS keychain.
    /// Falls back from project scope to global scope.
    public static func getSecret(_ key: String, project: String? = nil) throws -> String

    /// List all key names matching the given scope/project.
    public static func listSecrets(scope: String? = nil, project: String? = nil) throws -> [String]
}
```

**Platform implementations:**

```swift
// DarwinKeychain.swift — Native Security.framework (no subprocess)
import Security

func getSecret(service: String, account: String) throws -> String {
    let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: service,
        kSecAttrAccount as String: account,
        kSecReturnData as String: true,
        kSecMatchLimit as String: kSecMatchLimitOne,
    ]
    var result: CFTypeRef?
    let status = SecItemCopyMatching(query as CFDictionary, &result)
    guard status == errSecSuccess, let data = result as? Data else {
        throw KeySyncError.notFound
    }
    return String(data: data, encoding: .utf8) ?? ""
}

// LinuxKeychain.swift — Process + secret-tool
import Foundation
// Process().exec("secret-tool lookup service keysync/global account KEY")

// WindowsKeychain.swift — Process + helper or WinSDK
// (Windows Swift support is experimental; defer until the ecosystem matures)
```

**macOS advantage:** Swift on macOS can use `Security.framework` directly —
the cleanest, most idiomatic integration of any language library. No subprocess,
no shelling out, just native Keychain API calls.

---

## Service name helpers

Every library should implement these two functions:

```
serviceName(scope, project) → "keysync/global" | "keysync/project/<name>"
accountName(key)            → key (currently identity, may evolve)
```

Listing support requires knowing which keys exist. On macOS and Linux this
means parsing `security dump-keychain` or `secret-tool search service keysync`
output. On Windows, `wincred.List()` or `CredEnumerateW`. Each library should
cache the listing in memory (refreshed on each call).
