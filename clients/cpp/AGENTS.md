# keysync C++ Client

## Overview

The C++ client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via popen() | Ready |
| Linux | `secret-tool` CLI via popen() | Ready |
| Windows | Win32 API (CredReadW / CredEnumerateW) | Ready |

## API

```cpp
#include <keysync/keysync.hpp>

// Resolution order: getenv → project scope keychain → global scope keychain.
std::string keysync::getSecret(std::string_view key, std::string_view project = "")

std::vector<keysync::CredentialEntry> keysync::listSecrets(std::string_view project = "")
```

## Key design decisions

- **Compile-time platform dispatch** via preprocessor `#ifdef __APPLE__` / `#ifdef __linux__` / `#ifdef _WIN32` in `keysync.cpp`. No runtime branching.
- **Platform-specific `.cpp` files compiled conditionally** by CMake, matching Go's build-tag pattern.
- **C++17 standard** — `std::string_view` for parameters, structured bindings, `std::variant` alternatives available.
- **Exception-based errors** — `KeySyncError` with `ErrorCode` enum (NotFound, KeychainError, UnsupportedPlatform).
- **RAII everywhere** — `unique_ptr` with custom deleter wraps `CredFree` on Windows, file handles cleaned up automatically.
- **No external library dependencies** — `popen()/pclose()` for CLI access on macOS/Linux, `<windows.h>` and `<wincred.h>` on Windows.
- **Assert-based tests** — no test framework required (Catch2/Google Test optional). Self-contained with `ASSERT_TRUE` macros.
- **Service naming** matches keysync convention: `keysync/global` and `keysync/project/<name>`.
- **Windows target names** use underscores: `keysync_global`, `keysync_project_my-app`. UTF-16LE blobs converted to UTF-8 via `WideCharToMultiByte`.
