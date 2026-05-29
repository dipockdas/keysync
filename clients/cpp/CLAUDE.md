# keysync C++ Client — Claude Instructions

## Build & Test

```bash
cd clients/cpp
mkdir -p build && cd build
cmake ..           # Configure (C++17)
cmake --build .    # Build library and tests
ctest              # Run all tests
```

Clean build:

```bash
rm -rf build && mkdir build && cd build
cmake ..
cmake --build .
ctest
```

## Key files

```
include/keysync/
  keysync.hpp       # Public API (getSecret, listSecrets)
  errors.hpp        # KeySyncError exception, ErrorCode enum
  credential.hpp    # CredentialEntry struct (scope, project, environment, key)
src/
  keysync.cpp       # Main implementation + platform dispatch (#ifdef guards)
  macos.cpp         # macOS: popen() → security CLI
  linux.cpp         # Linux: popen() → secret-tool CLI
  windows.cpp       # Windows: Win32 API (CredReadW, CredEnumerateW)
  service.cpp       # Service name helpers (build, parse, target conversion)
  errors.cpp        # Error implementation
  internal_helpers.hpp  # Shared internal helpers header
tests/
  CMakeLists.txt
  test_main.cpp     # Error types, env var fallback, CredentialEntry tests
  test_service.cpp  # Service name construction, parsing, target conversion
CMakeLists.txt      # Root CMake project (C++17, conditional platform sources)
README.md
CLAUDE.md
```

## Conventions

- Platform detection via `#ifdef __APPLE__`, `#ifdef __linux__`, `#ifdef _WIN32`
- Platform-specific functions in separate `.cpp` files compiled conditionally via CMake
- `keysync::KeySyncError` thrown on failure (NotFound, KeychainError, UnsupportedPlatform)
- `keysync::CredentialEntry` struct for listing secrets (scope, project, key, environment)
- `std::string_view` for read-only string parameters, `std::string` for return values
- C++17 standard, no exceptions for control flow (exceptions only for error cases)
- Internal helpers in `namespace keysync::internal`
- `popen()` / `pclose()` for CLI access on macOS and Linux
- Win32 API directly on Windows (no external libraries)
- Simple assert-based tests (no external test framework required)

## Service Naming

| Scope       | Service Name                 |
|-------------|------------------------------|
| Global      | `keysync/global`             |
| Project     | `keysync/project/<name>`     |
| Environment | `keysync/project/<name>/env/<env>` |

On Windows, slashes in service names are replaced with underscores and `/env/` is stripped:
`keysync/global` → `keysync_global`, `keysync/project/my-app` → `keysync_project_my-app`, `keysync/project/my-app/env/dev` → `keysync_project_my-app_dev`

## Migration: replacing getenv/SecretManager with keysync

```cpp
#include <keysync/keysync.hpp>

// Global secret (shared across projects)
std::string apiKey = keysync::getSecret("API_KEY");

// Project-scoped secret (falls back to global if no project match)
std::string dbUrl = keysync::getSecret("DATABASE_URL", "myapp");

// Environment-scoped secret (falls back: env → project → global)
std::string stagingDb = keysync::getSecret("DATABASE_URL", "myapp", "staging");

// List all global secrets
auto globals = keysync::listSecrets();

// List project secrets
auto project = keysync::listSecrets("myapp");

// List environment secrets
auto staging = keysync::listSecrets("myapp", "staging");
```

## Resolution order (getSecret)

1. Check `std::getenv(key)` first — returns env var value if set
2. If `environment` is provided, try environment scope `keysync/project/<project>/env/<env>` first
3. If `project` is provided, try project scope `keysync/project/<project>` next
4. Fall back to global scope `keysync/global`
5. Throw `KeySyncError(ErrorCode::NotFound, ...)` if not found in any scope

## Platform-specific implementation details

**macOS** (`src/macos.cpp`):
- `popen("security find-generic-password -s <service> -a <account> -w 2>&1", "r")`
- Parse stdout, trim whitespace
- `security` returns exit code 44 when item not found
- `listSecrets` parses `security dump-keychain` output, filtering for generic password entries
- Extract attribute values using `findAttrValue()` helper

**Linux** (`src/linux.cpp`):
- `popen("secret-tool lookup service <service> account <account> 2>&1", "r")`
- Parse stdout, trim whitespace
- `secret-tool` returns exit code 1 and empty output when not found
- `listSecrets` parses `secret-tool search service keysync` output

**Windows** (`src/windows.cpp`):
- `#include <windows.h>`, `#include <wincred.h>`
- `CredReadW(L"keysync_global", CRED_TYPE_GENERIC, 0, &pCred)`
- `CredFree(pCred)` for cleanup (wrapped in RAII `unique_ptr` with custom deleter)
- Blob to string: `std::wstring(reinterpret_cast<const wchar_t*>(cred->CredentialBlob), cred->CredentialBlobSize / sizeof(wchar_t))`, then wide to UTF-8
- `listSecrets` uses `CredEnumerateW(L"keysync_*", 0, &count, &pCredentials)`
- `MultiByteToWideChar` / `WideCharToMultiByte` for string conversions
