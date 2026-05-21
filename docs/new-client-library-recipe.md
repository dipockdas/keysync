# Recipe: Writing a keysync Client Library

This document is a **recipe for AI coding assistants** (and human developers) to implement a new keysync client library in any programming language.

A keysync client library reads secrets directly from the OS keychain at runtime — no dependency on the `keysync` binary. The library exposes a simple `getSecret` / `listSecrets` API and handles platform-specific keychain access internally.

## Table of Contents

1. [Service Naming](#1-service-naming)
2. [Platform-Specific Keychain APIs](#2-platform-specific-keychain-apis)
3. [Required API Surface](#3-required-api-surface)
4. [Secret Resolution Order](#4-secret-resolution-order)
5. [Error Handling](#5-error-handling)
6. [File Structure](#6-file-structure)
7. [Testing Patterns](#7-testing-patterns)
8. [Meta-files](#8-meta-files)
9. [Step-by-Step Implementation Checklist](#9-step-by-step-implementation-checklist)

---

## 1. Service Naming

Secrets in the keychain are identified by a **service name** (or equivalent) and an **account name** (or equivalent). The service name encodes the scope and the account name is the secret key.

### Service Name Format

| Scope | Service Name | Example |
|-------|-------------|---------|
| Global | `keysync/global` | `keysync/global` |
| Project | `keysync/project/<name>` | `keysync/project/my-app` |
| Project + Environment | `keysync/project/<name>/env/<env>` | `keysync/project/my-app/env/production` |

### Account Name Format

The account/key name is always just the secret key itself:

| Secret Key | Account Name |
|------------|-------------|
| `DATABASE_URL` | `DATABASE_URL` |
| `API_KEY` | `API_KEY` |

### Windows Variant

Windows Credential Manager uses a tagged-field format with percent encoding (v2):

**Wire format**:
```
keysync|s=<scope>|p=<project>|e=<environment>|k=<key>
```

Examples:
- Global: `keysync|s=global|p=|e=|k=API_KEY`
- Project: `keysync|s=project|p=my-app|e=|k=DATABASE_URL`
- Project + Environment: `keysync|s=project|p=my-app|e=production|k=DATABASE_URL`

Values are percent-encoded using RFC 3986 rules (via `url.QueryEscape` or equivalent):
- Unreserved characters (A-Z, a-z, 0-9, -, ., _, ~) are NOT encoded
- Separators (|, =) are percent-encoded: | → %7C, = → %3D
- Spaces become `+` (not `%20`)

**Backward compatibility**: Client libraries should support reading legacy v1 credentials (e.g., `keysync_global_KEY`, `keysync_project_<name>_KEY`) as a fallback, but new writes use v2 format.

### Helper Functions Your Library Should Implement

**`serviceName(scope, project, environment?)`** → builds the service name string:

```
serviceName("global", "", "")                    → "keysync/global"
serviceName("project", "my-app", "")             → "keysync/project/my-app"
serviceName("project", "my-app", "production")   → "keysync/project/my-app/env/production"
```

**`parseServiceName(serviceName)`** → splits a service name back into (scope, project, environment):

```
parseServiceName("keysync/global")                                  → ("global", "", "")
parseServiceName("keysync/project/my-app")                          → ("project", "my-app", "")
parseServiceName("keysync/project/my-app/env/production")           → ("project", "my-app", "production")
parseServiceName("keysync/project/my/deep/path/env/staging")        → ("project", "my/deep/path", "staging")
```

**Key parsing rule**: Search for `/env/` as a distinct separator to handle project names that contain slashes. Do NOT use a simple `SplitN` by `/` with a fixed count — project names can contain `/` characters.

---

## 2. Platform-Specific Keychain APIs

### macOS — `security` CLI (subprocess)

The `security` command-line tool is built into macOS. Use `os/exec` / `subprocess` / equivalent.

#### Get a secret

```bash
security find-generic-password \
  -s "<serviceName>" \
  -a "<accountName>" \
  -w
```

- **Stdout**: The secret value (trailing newline)
- **Exit code 44**: Secret not found
- **Other non-zero exit**: Unexpected error

#### List all keysync secrets

```bash
security dump-keychain | grep "keysync"
```

Parse each line for `"svce"<blob>="<serviceName>"` and `"acct"<blob>="<accountName>"`.

**Alternative**: Use `security find-generic-password -s "keysync/global" -l` or iterate. The `dump-keychain` approach is simpler but requires parsing the bespoke output format.

#### Platform detection

```go
// Go — compile-time via build tags
//go:build darwin
```

```python
# Python — runtime via sys.platform
sys.platform == "darwin"
```

```typescript
// TypeScript — runtime via process.platform
process.platform === "darwin"
```

```swift
// Swift — compile-time via os() check
#if os(macOS)
```

### macOS — Security.framework (native API, Swift only)

The Swift client uses the native Security framework instead of a subprocess:

```swift
import Security

let query: [String: Any] = [
    kSecClass as String: kSecClassGenericPassword,
    kSecAttrService as String: serviceName,
    kSecAttrAccount as String: accountName,
    kSecReturnData as String: true,
    kSecMatchLimit as String: kSecMatchLimitOne,
]

var result: AnyObject?
let status = SecItemCopyMatching(query as CFDictionary, &result)

if status == errSecSuccess, let data = result as? Data {
    value = String(data: data, encoding: .utf8)
} else if status == errSecItemNotFound {
    // not found
} else {
    // unexpected error
}
```

### Linux — `secret-tool` CLI (subprocess)

The `secret-tool` command is part of `libsecret-tools`. Use `os/exec` / `subprocess`.

#### Get a secret

```bash
secret-tool lookup service "<serviceName>" account "<accountName>"
```

- **Stdout**: The secret value (trailing newline)
- **Exit code 1**: Secret not found or error (check stderr)
- **Note**: `secret-tool` exits with 1 for both "not found" and "unexpected error". Check stderr to distinguish: if stderr contains "not found" or is empty, it's a not-found condition; otherwise, it's an error.

#### List all keysync secrets

```bash
secret-tool search service keysync
```

Output format:

```
attribute: service = keysync/global
attribute: account = DATABASE_URL
attribute: <key> = <value>
```

Look for entries where the `service` attribute starts with `keysync/`.

#### Platform detection

```python
sys.platform == "linux"
```

```typescript
process.platform === "linux"
```

### Windows — Win32 API (CredReadW / CredEnumerateW)

Windows uses the Credential Manager Win32 API. There are two approaches:

#### Approach A: `wincred` Go library

```go
import "github.com/danieljoos/wincred"

// Get
cred, err := wincred.GetGenericCredential(targetName)
if err != nil {
    return "", err
}
value := string(cred.CredentialBlob)

// List
creds, err := wincred.List()
for _, cred := range creds {
    // Check for both v2 (keysync|) and legacy v1 (keysync_) formats
    if strings.HasPrefix(cred.TargetName, "keysync|") || strings.HasPrefix(cred.TargetName, "keysync_") {
        // Parse target name to extract scope, project, environment, key
        // ...
    }
}
```

#### Approach B: `ctypes` Win32 API (Python)

```python
import ctypes
from ctypes import wintypes

# CREDENTIALW struct
class CREDENTIALW(ctypes.Structure):
    _fields_ = [
        ("Flags", wintypes.DWORD),
        ("Type", wintypes.DWORD),
        ("TargetName", wintypes.LPCWSTR),
        ("Comment", wintypes.LPCWSTR),
        ("LastWritten", wintypes.FILETIME),
        ("CredentialBlobSize", wintypes.DWORD),
        ("CredentialBlob", wintypes.LPBYTE),
        ("Persist", wintypes.DWORD),
        ("AttributeCount", wintypes.DWORD),
        ("Attributes", ctypes.c_void_p),
        ("TargetAlias", wintypes.LPCWSTR),
        ("UserName", wintypes.LPCWSTR),
    ]

# CredReadW
advapi32 = ctypes.WinDLL("advapi32")
cred_read = advapi32.CredReadW
cred_read.restype = wintypes.BOOL
cred_read.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD, ctypes.POINTER(ctypes.POINTER(CREDENTIALW))]

# CredEnumerateW
cred_enumerate = advapi32.CredEnumerateW
cred_enumerate.restype = wintypes.BOOL
cred_enumerate.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, ctypes.POINTER(wintypes.DWORD), ctypes.POINTER(ctypes.POINTER(ctypes.POINTER(CREDENTIALW)))]
```

#### Platform detection

```python
sys.platform == "win32"
```

```go
//go:build windows
```

```typescript
process.platform === "win32"
```

### Unsupported Platform Handling

For platforms that are not macOS, Linux, or Windows, return a clear "unsupported platform" error immediately rather than failing mysteriously.

---

## 3. Required API Surface

Every client library must implement these two functions:

### `getSecret(key, project?)` / `GetSecret(key, project?)` / `get_secret(key, project=None)`

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | Yes | The secret key (e.g., `"DATABASE_URL"`) |
| `project` | string or null | No | Project name. If provided, checks project scope first, then falls back to global. If omitted, only checks global scope. |

**Returns**: The secret value as a string.

**Resolution behavior** (when `project` is provided):
1. Check `keysync/project/<name>/env/<environment>` — only if the caller also provides an environment
2. Check `keysync/project/<name>` — project scope
3. Check `keysync/global` — global scope (fallback)
4. If none found, return a not-found error

**Resolution behavior** (when `project` is NOT provided):
1. Check `keysync/global`
2. If not found, return a not-found error

> **Design note**: The client library is **read-only**. Unlike the internal `Store` interface used by the CLI, client libraries do NOT implement `Set` or `Delete`. This is intentional — client libraries are designed for runtime secret retrieval in production applications, not secret management.

### `listSecrets(scope?, project?)` / `ListSecrets(scope?, project?)` / `list_secrets(scope=None, project=None)`

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `scope` | string or null | No | Filter by scope: `"global"` or `"project"`. Empty/null means no filter. |
| `project` | string or null | No | Filter by project name. Empty/null means no filter. |

**Returns**: An array of objects with `{scope, project?, key}`. Values are NOT included — the list operation only returns metadata.

### Additional Exports

| Export | Type | Description |
|--------|------|-------------|
| `SecretNotFoundError` | Error class | Raised/returned when a secret key is not found. Should include the key name in the error message. |

---

## 4. Secret Resolution Order

When a client library resolves a secret, it follows this precedence:

```
Highest:  Environment-scoped (project + env match)
           ↓
          Project-scoped (project match, no env)
           ↓
Lowest:   Global (no project, no env)
```

This means:
- A global `DATABASE_URL` is overridden by a project-scoped `DATABASE_URL`
- A project-scoped `DATABASE_URL` is overridden by an environment-scoped `DATABASE_URL` (with matching env)
- The `getSecret` function should implement this fallback chain internally

---

## 5. Error Handling

### Error Types

| Condition | Error | Message Pattern |
|-----------|-------|-----------------|
| Secret key not found in any scope | Not Found | `"secret not found: DATABASE_URL"` |
| Keychain access failure (locked, unavailable) | Keychain Error | `"keychain error: <details>"` |
| Unsupported platform | Unsupported | `"unsupported platform: <platform>"` |
| Unexpected data (parsing failure) | Parse Error | `"unexpected keychain output: <details>"` |

### Language-Specific Patterns

```go
// Go — sentinel error
var ErrNotFound = errors.New("secret not found")

func GetSecret(key, project string) (string, error) {
    // ...
    return "", fmt.Errorf("%w: %s", ErrNotFound, key)
}
```

```python
# Python — exception classes
class KeySyncError(Exception):
    def __init__(self, code: str, message: str):
        self.code = code
        super().__init__(message)

class SecretNotFoundError(KeySyncError):
    def __init__(self, key: str):
        super().__init__("notFound", f"secret not found: {key}")
```

```typescript
// TypeScript — error class with code
class KeySyncError extends Error {
    constructor(
        public code: "notFound" | "keychainError" | "unsupportedPlatform",
        message: string
    ) {
        super(message);
        this.name = "KeySyncError";
    }
}
```

```swift
// Swift — enum error
enum KeySyncError: Error {
    case notFound
    case keychainError(String)
    case unexpectedData
    case unsupportedPlatform
}
```

---

## 6. File Structure

A client library should follow this file structure:

```
clients/<language>/
├── README.md          # User-facing documentation (see §8)
├── CLAUDE.md          # Claude Code instructions (see §8)
├── AGENTS.md          # AI agent instructions (see §8)
├── <source files>     # Platform-specific implementations
├── <test files>       # Tests
├── <config files>     # go.mod, pyproject.toml, package.json, Package.swift, etc.
└── <build artifacts>  # dist/, bin/, .build/, etc. (gitignored)
```

### Recommended Source File Organization

| File | Purpose |
|------|---------|
| `main.<ext>` | Public API, service name helpers, platform dispatch |
| `darwin.<ext>` | macOS implementation (security CLI) |
| `linux.<ext>` | Linux implementation (secret-tool CLI) |
| `windows.<ext>` | Windows implementation (Credential Manager API) |
| `unsupported.<ext>` | Stub that returns unsupported-platform error |

For compiled languages with conditional compilation (Go, Swift), use the language's native mechanism:

- **Go**: `//go:build darwin`, `//go:build linux`, `//go:build windows`
- **Swift**: `#if os(macOS)`, `#elseif os(Linux)`, `#else`

For interpreted languages (Python, TypeScript), use runtime detection:

- **Python**: `sys.platform == "darwin"`, `sys.platform == "linux"`, `sys.platform == "win32"`
- **TypeScript**: `process.platform === "darwin"`, `process.platform === "linux"`, `process.platform === "win32"`

---

## 7. Testing Patterns

### What to Test

| Test | Description |
|------|-------------|
| Service name construction | `serviceName()` produces correct strings for global, project, and project+env |
| Service name parsing | `parseServiceName()` correctly handles normal names, deep project paths, and edge cases |
| Global-only get | `getSecret("KEY")` with no project returns global value |
| Project-scoped get | `getSecret("KEY", "my-app")` with a project returns project value |
| Fallback from project to global | `getSecret("KEY", "my-app")` falls back to global when project key doesn't exist |
| Not-found error | `getSecret("NONEXISTENT")` returns appropriate not-found error |
| List all | `listSecrets()` returns all entries |
| List filtered by scope | `listSecrets(scope="global")` returns only global entries |
| List filtered by project | `listSecrets(project="my-app")` returns only matching entries |

### How to Mock the Keychain

For languages that use `subprocess`/`exec`:

1. **Replace the exec function at test time**: Store the keychain function as a variable that can be replaced with a mock:

```go
// In production
var platformGet = darwinGet

// In test
platformGet = func(service, account string) (string, error) {
    return "mock-value", nil
}
```

```python
# In production
_platform_get = _darwin.darwin_get

# In test (monkeypatch)
import your_module
your_module._platform_get = lambda service, account: "mock-value"
```

2. **Test the service name logic separately** — it doesn't require keychain access and makes up the bulk of your test coverage.

3. **For integration tests**: If the OS keychain is available, run one real get/list test (typically guarded by a build tag or skipped on CI).

### Go Test Pattern (from the existing Go client)

```go
func TestGetSecret_GlobalOnly(t *testing.T) {
    // Override platform implementation with mock
    origGet := platformGet
    platformGet = func(service, account string) (string, error) {
        if service == "keysync/global" && account == "TEST_KEY" {
            return "test-value", nil
        }
        return "", ErrNotFound
    }
    defer func() { platformGet = origGet }()

    val, err := GetSecret("TEST_KEY", "")
    if err != nil {
        t.Fatalf("GetSecret failed: %v", err)
    }
    if val != "test-value" {
        t.Errorf("got %q, want %q", val, "test-value")
    }
}
```

---

## 8. Meta-files

Every client library needs three meta-files. Copy the structure from the existing Go client.

### `README.md`

Contents:
- Brief description of the library
- Installation instructions
- Quick start example (get + list)
- API reference
- Platform support table
- Link to the main keysync project

### `CLAUDE.md`

Contents:
- Build and test commands
- File listing
- Key conventions for the specific language

Example (from Go client):

```markdown
# keysync Go Client — Project Instructions

## Files
keysync.go   # Public API
darwin.go    # macOS (security CLI)
linux.go     # Linux (secret-tool CLI)
windows.go   # Windows (wincred)
store.go     # Store interface for testing
```

### `AGENTS.md`

Contents:
- Overview of what the client does
- Platform support table
- API summary
- Key design decisions (read-only, platform-specific files, error types)

---

## 9. Step-by-Step Implementation Checklist

Use this checklist when implementing a new client library in any language.

### Phase 1: Service Name Helpers

- [ ] Implement `serviceName(scope, project, environment?)` function
- [ ] Implement `parseServiceName(serviceName)` function
- [ ] Handle the `/env/` separator correctly for deep project paths
- [ ] Write tests for all service name cases (global, project, project+env, deep paths, edge cases)
- [ ] For Windows: implement v2 tagged-field format with percent encoding, plus v1 fallback support

### Phase 2: Platform Implementations

- [ ] macOS implementation:
  - [ ] Implement `get(service, account)` via `security find-generic-password`
  - [ ] Implement `list()` via `security dump-keychain` parsing
  - [ ] Handle exit code 44 (not found)
- [ ] Linux implementation:
  - [ ] Implement `get(service, account)` via `secret-tool lookup`
  - [ ] Implement `list()` via `secret-tool search service keysync`
  - [ ] Handle exit code 1 (check stderr to distinguish not-found vs error)
- [ ] Windows implementation:
  - [ ] Implement `get(target)` via `CredReadW` Win32 API (try v2 format first, fallback to v1)
  - [ ] Implement `list()` via `CredEnumerateW` Win32 API (parse both v2 and v1 formats)
  - [ ] Parse v2 tagged format: `keysync|s=<scope>|p=<project>|e=<env>|k=<key>`
  - [ ] Parse v1 legacy format: `keysync_global_<key>`, `keysync_project_<name>_<key>`
  - [ ] Implement percent decoding for v2 values (url.QueryUnescape or equivalent)
  - [ ] Handle ERROR_NOT_FOUND
- [ ] Unsupported platform stub (return clear error)

### Phase 3: Public API

- [ ] Implement `getSecret(key, project?)` with fallback chain (env → project → global → error)
- [ ] Implement `listSecrets(scope?, project?)` with filtering
- [ ] Define error types (`SecretNotFoundError`, `KeySyncError`)
- [ ] Platform dispatch mechanism (compile-time or runtime)

### Phase 4: Testing

- [ ] Tests for service name helpers (8-10 test cases)
- [ ] Tests for platform-agnostic fallback logic
- [ ] Tests for error behavior (not-found, malformed output)
- [ ] Optionally: one real keychain integration test (guarded)

### Phase 5: Documentation & Polish

- [ ] Add platform-specific build notes to README
- [ ] Handle edge cases: empty strings, trailing newlines in CLI output, unicode values
- [ ] Test on all three platforms (or note platform limitations)

---

## Existing Client Library Reference

The existing client libraries are the best reference for implementation details:

| Library | Location | Key Files |
|---------|----------|-----------|
| **Go** | `clients/go/` | `keysync.go`, `darwin.go`, `linux.go`, `windows.go` |
| **Python** | `clients/python/` | `src/keysync/__init__.py`, `_darwin.py`, `_linux.py`, `_windows.py` |
| **TypeScript** | `clients/node/` | `src/index.ts`, `src/darwin.ts`, `src/linux.ts`, `src/windows.ts` |
| **Swift** | `clients/swift/` | `Sources/KeySync/KeySyncClient.swift`, `DarwinKeychain.swift`, `LinuxKeychain.swift` |

Study the target-language client most closely, then follow the checklist above for a complete, consistent implementation.
