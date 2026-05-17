# keysync C++ Client Library

Retrieve secrets managed by keysync directly from the OS keychain.
No dependency on the `keysync` binary -- apps link directly against this
library to access secrets at runtime.

## Platform support

| Platform | Mechanism | Notes |
|----------|-----------|-------|
| macOS    | `security` CLI (built-in) | Uses `popen()` to call `security find-generic-password` |
| Linux    | `secret-tool` CLI (libsecret) | Requires `libsecret-tools` package |
| Windows  | Win32 Credential Manager API | Calls `CredReadW` / `CredEnumerateW` directly |

## Build

```bash
cd clients/cpp
mkdir build && cd build
cmake ..           # Configure (C++17 required)
cmake --build .    # Build the library
ctest              # Run the test suite
```

## Usage

```cpp
#include <keysync/keysync.hpp>
#include <iostream>

int main() {
    try {
        // Retrieve a global secret
        std::string apiKey = keysync::getSecret("API_KEY");
        std::cout << "API key retrieved successfully" << std::endl;

        // Retrieve a project-scoped secret (falls back to global)
        std::string dbUrl = keysync::getSecret("DATABASE_URL", "myapp");

        // List all global secrets
        auto globals = keysync::listSecrets();
        for (const auto& entry : globals) {
            std::cout << entry.key << std::endl;
        }

    } catch (const keysync::KeySyncError& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }
    return 0;
}
```

## CMake integration

```cmake
# Add keysync as a subdirectory
add_subdirectory(path/to/keysync-clients/cpp)

# Link against it
target_link_libraries(myapp PRIVATE keysync)
```

## How it works

**Resolution order** (for every `getSecret` call):

1. Check the environment variable identified by `key` (`std::getenv(key)`) first.
   This is the fast path for local dev (`eval $(keysync export)`) and cloud/CI
   environments where the platform injects environment variables directly.
2. If not found, fall back to the OS keychain.
3. If a project is provided, check the project scope (`keysync/project/<name>`)
   first, then fall back to the global scope (`keysync/global`).

**Service naming:**

| Scope   | Service Name               | Account (Key)       |
|---------|----------------------------|---------------------|
| Global  | `keysync/global`           | `DATABASE_URL`      |
| Project | `keysync/project/my-api`   | `DATABASE_URL`      |

**Platform-specific commands:**

- **macOS**: `security find-generic-password -s keysync/global -a DATABASE_URL -w`
- **Linux**: `secret-tool lookup service keysync/global account DATABASE_URL`
- **Windows**: `CredReadW(L"keysync_global", CRED_TYPE_GENERIC, ...)`

No subprocess chain, no dependency on the keysync CLI. Read operations work
standalone as long as secrets have been stored (via `keysync set` or any other
method that writes to the same keychain entries).

## Error handling

All keychain and lookup failures are reported via `keysync::KeySyncError`,
which provides an error code:

| Code | Meaning |
|------|---------|
| `ErrorCode::NotFound` | The secret was not found in any scope |
| `ErrorCode::KeychainError` | An OS-level keychain error occurred |
| `ErrorCode::UnsupportedPlatform` | The platform is not supported |

## Requirements

- C++17 or later
- CMake 3.15 or later
- macOS: no additional dependencies
- Linux: `libsecret-tools` installed (`apt install libsecret-tools` or equivalent)
- Windows: no additional dependencies (uses Win32 API directly)
