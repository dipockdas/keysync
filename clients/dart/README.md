# Keysync Dart/Flutter Client

Dart/Flutter client library for retrieving secrets from environment variables or OS keychain.

**Designed for Flutter desktop applications** (macOS, Linux, Windows). Not for mobile apps (iOS/Android), which use different secret storage mechanisms.

## Installation

Add to your `pubspec.yaml`:

```yaml
dependencies:
  keysync:
    path: ../path/to/keysync/clients/dart
```

Or if published to pub.dev:

```yaml
dependencies:
  keysync: ^1.0.0
```

## Usage

### Basic Example

```dart
import 'package:keysync/keysync.dart';

void main() async {
  // Global secret
  final apiKey = await getSecret('API_KEY');
  print('API Key: $apiKey');

  // Project-scoped secret
  final dbUrl = await getSecret('DATABASE_URL', project: 'myapp');
  print('Database URL: $dbUrl');

  // Environment-scoped secret
  final prodDbUrl = await getSecret(
    'DATABASE_URL',
    project: 'myapp',
    environment: 'production',
  );
  print('Production DB: $prodDbUrl');
}
```

### Error Handling

```dart
try {
  final secret = await getSecret('MY_SECRET');
  print('Secret: $secret');
} catch (e) {
  if (e is KeysyncException) {
    print('Failed to retrieve secret: ${e.message}');
  } else {
    rethrow;
  }
}
```

### Flutter App Example

```dart
import 'package:flutter/material.dart';
import 'package:keysync/keysync.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Load secrets before app starts
  final apiKey = await getSecret('API_KEY');
  final dbUrl = await getSecret('DATABASE_URL', project: 'myapp');

  runApp(MyApp(apiKey: apiKey, dbUrl: dbUrl));
}

class MyApp extends StatelessWidget {
  final String apiKey;
  final String dbUrl;

  const MyApp({
    required this.apiKey,
    required this.dbUrl,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text('Keysync Example')),
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text('API Key: ${apiKey.substring(0, 8)}...'),
              Text('Database: ${dbUrl.split('@').last}'),
            ],
          ),
        ),
      ),
    );
  }
}
```

## How It Works

### Resolution Order

The library checks for secrets in this order (highest to lowest precedence):

1. **Environment variable** matching the key name
2. **OS keychain** (if not in env vars):
   - Environment-scoped (if `project` + `environment` provided)
   - Project-scoped (if `project` provided)
   - Global scope (default)

### Platform Support

| Platform | Backend | Tool |
|----------|---------|------|
| **macOS** | Keychain Access | `security` CLI |
| **Linux** | GNOME Keyring / KDE Wallet | `secret-tool` CLI |
| **Windows** | Credential Manager | PowerShell + Win32 API |

**Mobile platforms** (iOS/Android) are not supported — this library is for Flutter desktop apps only.

### Environment Variables (Primary Path)

In production, secrets are typically injected as environment variables by the platform (Vercel, Railway, AWS, etc.). The client checks env vars **first**, so no keychain access happens in production.

```bash
# Local development (macOS/Linux)
eval $(keysync export --project myapp)
flutter run

# Windows
keysync export --project myapp | Out-String | Invoke-Expression
flutter run
```

In this workflow:
- `keysync export` loads secrets from the OS keychain into env vars
- Your app reads from env vars (no keychain prompts during runtime)
- The keychain is only accessed by the keysync CLI, not your app

## API Reference

### `Future<String> getSecret(String key, {String? project, String? environment})`

Retrieves a secret value.

**Arguments:**
- `key` (required): Secret key name (e.g., `'DATABASE_URL'`)
- `project` (optional): Project name for project-scoped secrets
- `environment` (optional): Environment name (e.g., `'dev'`, `'staging'`, `'production'`)

**Returns:** `Future<String>` — the secret value

**Throws:** `KeysyncException` if the secret is not found

**Examples:**

```dart
// Global scope
final key = await getSecret('API_KEY');

// Project scope
final url = await getSecret('DATABASE_URL', project: 'myapp');

// Environment scope
final prodUrl = await getSecret(
  'DATABASE_URL',
  project: 'myapp',
  environment: 'production',
);
```

### `class KeysyncException`

Exception thrown when a secret cannot be retrieved.

**Properties:**
- `message`: Error description

## Platform Requirements

### macOS

- Uses the built-in `security` CLI (no installation needed)
- Secrets stored in the default Keychain

### Linux

- Requires `libsecret-tools`:

  ```bash
  # Debian / Ubuntu
  sudo apt-get install libsecret-tools

  # Fedora
  sudo dnf install libsecret

  # Arch Linux
  sudo pacman -S libsecret
  ```

- Requires a running keyring (GNOME Keyring, KDE Wallet, or KeePassXC)

### Windows

- Uses Windows Credential Manager (built-in, no installation needed)
- Requires PowerShell (included in Windows 10+)
- Uses `CredentialManagement` PowerShell module (auto-loaded)

## Storing Secrets

Use the keysync CLI to store secrets:

```bash
# Global secret
keysync set API_KEY=abc123

# Project-scoped secret
keysync set -p myapp DATABASE_URL=postgres://localhost/mydb

# Environment-scoped secret
keysync set -p myapp DATABASE_URL=postgres://prod-host/mydb --env production
```

See the [main keysync documentation](https://github.com/dipockdas/keysync) for full CLI usage.

## Example Project

See `example/main.dart` for a complete working example demonstrating:
- Global secret retrieval
- Project-scoped secrets
- Environment-scoped secrets
- Error handling

Run the example:

```bash
cd clients/dart
dart run example/main.dart
```

## Development

### Running Tests

```bash
# Run tests (when added)
dart test

# Analyze code
dart analyze

# Format code
dart format .
```

### Linting

The package uses the `lints` package for Dart linting. All code follows the recommended Dart style guide.

## Troubleshooting

### "Failed to retrieve secret from macOS Keychain"

**Cause:** Secret not found or `security` CLI unavailable.

**Solution:**
- Verify the secret exists: `keysync list -p myapp`
- Store the secret: `keysync set -p myapp MY_SECRET=value`

### "Failed to retrieve secret from Linux keyring"

**Cause:** `secret-tool` not installed or no keyring running.

**Solution:**
- Install libsecret: `sudo apt-get install libsecret-tools`
- Ensure GNOME Keyring or KDE Wallet is running
- Test manually: `secret-tool lookup service keysync/global account MY_SECRET`

### "Failed to retrieve secret from Windows Credential Manager"

**Cause:** PowerShell unavailable or credential not found.

**Solution:**
- Verify PowerShell is available: `powershell -Command "Get-Host"`
- Check credential exists: `keysync list -p myapp`
- Store the secret: `keysync set -p myapp MY_SECRET=value`

### Windows: "CredentialManagement module not found"

**Cause:** Older Windows versions may not have the module.

**Solution:** Install the CredentialManagement module:

```powershell
Install-Module -Name CredentialManagement -Force
```

## Related Documentation

- [Main keysync README](../../README.md) — Overview and CLI usage
- [Keysync AGENTS.md](../../AGENTS.md) — Architecture and integration guide
- [Client Library Recipe](../../docs/new-client-library-recipe.md) — How this client was built

## License

Apache 2.0 — see [LICENSE](../../LICENSE)
