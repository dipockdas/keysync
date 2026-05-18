# keysync Java Client -- Claude Instructions

## Build & Test

```bash
cd clients/java
mvn clean compile      # Compile
mvn test               # Run all tests
mvn package            # Build JAR
```

## Key files

```
pom.xml                          # Maven project (groupId: io.keysync, artifactId: keysync)
src/main/java/io/keysync/
  KeySyncClient.java             # Public API (getSecret, listSecrets), service name helpers
  KeySyncError.java              # Error type enum (NOT_FOUND, KEYCHAIN_ERROR, UNSUPPORTED_PLATFORM)
  KeySyncException.java          # Unchecked exception with error code and message
  KeychainProvider.java          # Interface for platform backends
  Credential.java                # POJO for list results (service + account + environment)
  MacKeychain.java               # macOS: ProcessBuilder -> security CLI
  LinuxKeychain.java             # Linux: ProcessBuilder -> secret-tool CLI
  WindowsKeychain.java           # Windows: JNA -> advapi32.dll CredReadW/CredEnumerateW
src/test/java/io/keysync/
  KeySyncClientTest.java         # JUnit 5 tests
```

## Conventions

- `System.getProperty("os.name").toLowerCase()` for runtime platform detection
- Platform detection checks for "mac", "linux", or "win" substrings
- `KeySyncError` enum with `getCode()` for error type strings
- `KeySyncException` (unchecked) for all error conditions
- Service naming: `keysync/global`, `keysync/project/<name>`, `keysync/project/<name>/env/<env>`
- Windows: target names use underscores instead of slashes and strip `/env/` keyword (`keysync_global`, `keysync_project_myapp`, `keysync_project_myapp_dev`)
- JNA for Windows Win32 API access (CredReadW, CredEnumerateW, CredFree from advapi32.dll)
- JUnit 5 (`@Test`, `@Nested`, `@DisplayName`) for tests
- Tests cover service name construction, parsing, error types, env var fallback, platform detection, Windows target conversion, environment scope, and singleton behavior

## Important implementation details

- **macOS**: The `security` CLI writes the password to stderr on newer macOS versions (13+). Capture both stdout and stderr. Exit code 44 indicates "item not found".
- **Linux**: Uses `secret-tool lookup keysync-service <service> keysync-key <account>`. Exit code 1 indicates "not found". For listing, uses `secret-tool search keysync-service keysync` and parses attribute=value lines.
- **Windows**: Uses JNA to call CredReadW from advapi32.dll. Defines a CREDENTIALW Structure with the correct field order. CredentialBlob is stored as UTF-16LE; read as char array from the native pointer. CredEnumerateW with filter "keysync_*" lists all matching credentials. Target names have slashes replaced with underscores.

## Migration: replacing System.getenv with keysync

```java
import io.keysync.KeySyncClient;
import io.keysync.KeySyncException;

KeySyncClient client = KeySyncClient.getInstance();

// Global secret (shared across projects)
String apiKey = client.getSecret("API_KEY");

// Project-scoped secret (falls back to global if no project match)
String dbUrl = client.getSecret("DATABASE_URL", "myapp");

// Environment-scoped secret (falls back: env → project → global)
String stagingDb = client.getSecret("DATABASE_URL", "myapp", "staging");

// List all secrets
var globals = client.listSecrets();
var project = client.listSecrets("myapp");
var staging = client.listSecrets("myapp", "staging");
```
