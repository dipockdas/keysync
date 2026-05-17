# keysync Java Client

## Overview

The Java client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via ProcessBuilder | Ready |
| Linux | `secret-tool` CLI via ProcessBuilder | Ready |
| Windows | JNA → advapi32.dll (CredReadW / CredEnumerateW) | Ready |

## API

```java
KeySyncClient client = KeySyncClient.getInstance();

// Thread-safe singleton. Resolution order: env var → project scope → global scope.
String getSecret(String key) throws KeySyncException
String getSecret(String key, String project) throws KeySyncException

List<Credential> listSecrets() throws KeySyncException
List<Credential> listSecrets(String project) throws KeySyncException
```

## Key design decisions

- **Thread-safe singleton** via double-checked locking — no need to create multiple instances.
- **Runtime platform detection** via `System.getProperty("os.name").toLowerCase()`.
- **Windows uses JNA** to call Win32 APIs from advapi32.dll (CredReadW, CredEnumerateW, CredFree). No native compilation required — JNA handles FFI at runtime. CREDENTIALW structure defined with proper `@FieldOrder` annotation.
- **Unchecked exceptions** — `KeySyncException` extends RuntimeException for idiomatic Java usage.
- **Error codes** via `KeySyncError` enum with NOT_FOUND, KEYCHAIN_ERROR, UNSUPPORTED_PLATFORM.
- **macOS stderr handling** — `security` CLI on macos 13+ writes password to stderr. Captures both streams.
- **Service naming** matches keysync convention: `keysync/global` and `keysync/project/<name>`.
- **Windows target names** use underscores instead of slashes: `keysync_global`, `keysync_project_myapp`.
- **Maven project** (groupId: io.keysync, artifactId: keysync) requiring Java 17+ and JNA 5.14.0.
- **JUnit 5** tests with `@Nested` inner classes for organized test structure.
