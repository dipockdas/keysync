# KeySync Java Client

Java library for reading secrets from the OS-native keychain with zero dependency on the `keysync` binary.

## Supported platforms

| Platform | Backend | Dependency | Status |
|----------|---------|------------|--------|
| macOS | `security` CLI | Built-in | Ready |
| Linux | `secret-tool` CLI | `libsecret-tools` package | Ready |
| Windows | Win32 Credential Manager (JNA) | `net.java.dev.jna:jna` | Available (not fully tested on Windows) |

## Requirements

- Java 17 or later
- Maven 3.6+ (for building)
- JNA 5.14.0 (included as a transitive dependency)

## Installation

Add the dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>io.keysync</groupId>
    <artifactId>keysync</artifactId>
    <version>0.1.0</version>
</dependency>
```

For Gradle:

```groovy
implementation 'io.keysync:keysync:0.1.0'
```

## Quick start

```java
import io.keysync.*;

// Get the singleton client
KeySyncClient client = KeySyncClient.getInstance();

// Read a global secret
String apiKey = client.getSecret("API_KEY");

// Read a project-scoped secret (falls back to global if no project match)
String dbUrl = client.getSecret("DATABASE_URL", "myapp");

// List all global secrets
List<Credential> globals = client.listSecrets();

// List secrets for a project (includes global fallback)
List<Credential> project = client.listSecrets("myapp");

for (Credential c : project) {
    System.out.println(c.getService() + " -> " + c.getAccount());
}
```

## Resolution order

Every `getSecret` call follows this order:

1. **Environment variable** -- `System.getenv(key)` is checked first. If the environment variable is set, the value is returned immediately without touching the keychain. This is the primary path for cloud/CI deployments where platforms inject environment variables directly, and for local development where secrets are exported via `eval $(keysync export)`.

2. **Project-scoped keychain** (if a project is provided) -- looks up the secret under `keysync/project/<name>`.

3. **Global keychain fallback** -- looks up the secret under `keysync/global`.

## Error handling

The library throws `KeySyncException` (an unchecked exception) with one of three error codes:

| Code | Meaning |
|------|---------|
| `notFound` | The secret does not exist in any scope |
| `keychainError` | The native keychain tool failed |
| `unsupportedPlatform` | The current OS is not supported |

```java
try {
    String secret = client.getSecret("MISSING_KEY");
} catch (KeySyncException e) {
    switch (e.getError()) {
        case NOT_FOUND:
            System.err.println("Secret not found: " + e.getMessage());
            break;
        case KEYCHAIN_ERROR:
            System.err.println("Keychain error: " + e.getMessage());
            break;
        case UNSUPPORTED_PLATFORM:
            System.err.println("Unsupported platform");
            break;
    }
}
```

## Building from source

```bash
cd clients/java
mvn clean compile   # Compile
mvn test            # Run tests
mvn package         # Build JAR
```

## Service naming convention

Secrets are stored in the keychain with service names that encode scope and project:

| Scope | Service name |
|-------|-------------|
| Global | `keysync/global` |
| Project | `keysync/project/<name>` |

## License

MIT
