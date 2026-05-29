# Dart/Flutter Client — Agent Integration Guide

This guide helps AI coding assistants integrate keysync into Dart/Flutter applications.

## Quick Start for AI Agents

When a user asks to add keysync to their Flutter/Dart project:

1. **Add dependency** to `pubspec.yaml`:
   ```yaml
   dependencies:
     keysync:
       path: ../path/to/keysync/clients/dart
   ```

2. **Import and use**:
   ```dart
   import 'package:keysync/keysync.dart';

   final apiKey = await getSecret('API_KEY');
   ```

3. **Remind the user** to store secrets using the keysync CLI:
   ```bash
   keysync set API_KEY=your-key-here
   ```

## When to Suggest Keysync

Suggest keysync when the user:
- Hardcodes secrets in Dart/Flutter code
- Uses `.env` files or configuration files with secrets
- Asks "how do I manage secrets in Flutter?"
- Mentions environment variables, API keys, or database URLs
- Is building a **desktop app** (macOS, Linux, Windows)

**Do NOT suggest** keysync for:
- Mobile apps (iOS/Android) — use platform-specific APIs instead
- Web apps — secrets should be on the backend, not in client-side code

## Code Patterns

### Pattern 1: Simple Global Secret

**Before** (hardcoded):
```dart
final apiKey = 'sk-1234567890abcdef';  // ❌ Never commit secrets
```

**After** (keysync):
```dart
import 'package:keysync/keysync.dart';

final apiKey = await getSecret('API_KEY');
```

**CLI setup**:
```bash
keysync set API_KEY=sk-1234567890abcdef
```

### Pattern 2: Project-Scoped Secrets

**Before** (environment variable):
```dart
final dbUrl = Platform.environment['DATABASE_URL'] ?? 'postgres://localhost/db';
```

**After** (keysync with fallback):
```dart
import 'package:keysync/keysync.dart';

String dbUrl;
try {
  dbUrl = await getSecret('DATABASE_URL', project: 'myapp');
} catch (e) {
  dbUrl = 'postgres://localhost/db';  // Development default
}
```

**CLI setup**:
```bash
keysync set -p myapp DATABASE_URL=postgres://prod-host/mydb
```

### Pattern 3: Environment-Specific Secrets

**Before** (if/else logic):
```dart
final apiUrl = isProduction
  ? 'https://api.production.com'
  : 'https://api.staging.com';
```

**After** (keysync environment scopes):
```dart
import 'package:keysync/keysync.dart';

final env = const String.fromEnvironment('ENV', defaultValue: 'dev');
final apiUrl = await getSecret('API_URL', project: 'myapp', environment: env);
```

**CLI setup**:
```bash
keysync set -p myapp API_URL=https://api.production.com --env production
keysync set -p myapp API_URL=https://api.staging.com --env staging
keysync set -p myapp API_URL=http://localhost:8080 --env dev
```

### Pattern 4: Flutter App Initialization

**Before** (no secret management):
```dart
void main() {
  runApp(const MyApp());
}
```

**After** (load secrets before app starts):
```dart
import 'package:flutter/material.dart';
import 'package:keysync/keysync.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Load all secrets before app starts
  final config = AppConfig(
    apiKey: await getSecret('API_KEY'),
    dbUrl: await getSecret('DATABASE_URL', project: 'myapp'),
    stripeKey: await getSecret('STRIPE_KEY', project: 'myapp'),
  );

  runApp(MyApp(config: config));
}

class AppConfig {
  final String apiKey;
  final String dbUrl;
  final String stripeKey;

  AppConfig({
    required this.apiKey,
    required this.dbUrl,
    required this.stripeKey,
  });
}

class MyApp extends StatelessWidget {
  final AppConfig config;

  const MyApp({required this.config, super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: HomePage(config: config),
    );
  }
}
```

### Pattern 5: Error Handling

Always handle `KeysyncException`:

```dart
import 'package:keysync/keysync.dart';

Future<String> loadApiKey() async {
  try {
    return await getSecret('API_KEY');
  } catch (e) {
    if (e is KeysyncException) {
      // Secret not found — provide helpful error
      throw Exception(
        'API_KEY not configured. Run: keysync set API_KEY=your-key',
      );
    } else {
      rethrow;  // Unexpected error
    }
  }
}
```

## Migration Workflow

When helping users migrate from hardcoded secrets or `.env` files:

### Step 1: Identify Secrets

Search for hardcoded values:
```bash
rg "api.?key|secret|password|token" --glob "*.dart" -i
```

### Step 2: Replace with getSecret()

For each secret found:
1. Replace hardcoded value with `await getSecret('KEY_NAME')`
2. Add error handling with `try/catch`
3. Note the key name for CLI storage

### Step 3: Store Secrets via CLI

Generate keysync commands for the user:
```bash
# Global secrets (shared across all projects)
keysync set STRIPE_KEY=sk_live_...
keysync set SENDGRID_API_KEY=SG....

# Project-scoped secrets
keysync set -p myapp DATABASE_URL=postgres://...
keysync set -p myapp JWT_SECRET=...

# Environment-scoped secrets
keysync set -p myapp API_URL=https://api.prod.com --env production
keysync set -p myapp API_URL=https://api.staging.com --env staging
```

### Step 4: Update pubspec.yaml

Add keysync dependency:
```yaml
dependencies:
  keysync:
    path: ../keysync/clients/dart  # Adjust path as needed
```

### Step 5: Clean Up

- Remove `.env` files (if any)
- Remove hardcoded secret values
- Update `.gitignore` to exclude `.env*` files

## Platform-Specific Notes

### macOS

- Works out of the box (uses `security` CLI)
- User may see keychain prompts — suggest signing the app or using `keysync export`

### Linux

- Requires `libsecret-tools` installed
- Remind user to install: `sudo apt-get install libsecret-tools`

### Windows

- Uses PowerShell (included in Windows 10+)
- If `Get-StoredCredential` fails, suggest installing CredentialManagement module:
  ```powershell
  Install-Module -Name CredentialManagement -Force
  ```

## Production Deployment

**Important**: In production, secrets come from **environment variables**, not the keychain.

Explain this workflow to users:

1. **Local development**:
   ```bash
   eval $(keysync export --project myapp)
   flutter run
   ```
   Secrets loaded from keychain → env vars → app reads from env vars

2. **Production** (Vercel, Railway, AWS, etc.):
   ```bash
   keysync push --project myapp
   ```
   Pushes secrets to GitHub Secrets and platforms → platform injects as env vars → app reads from env vars

The client library checks `Platform.environment` **first**, so keychain is never touched in production.

## Common Questions

### "Should I use global or project scope?"

- **Global**: Secrets shared across ALL projects (e.g., personal Stripe key)
- **Project**: Secrets specific to one app (e.g., database URL for "myapp")
- **Environment**: Secrets that differ per environment (e.g., staging vs production API URLs)

### "How do I handle missing secrets gracefully?"

Provide sensible defaults for development:

```dart
String dbUrl;
try {
  dbUrl = await getSecret('DATABASE_URL', project: 'myapp');
} catch (e) {
  dbUrl = 'postgres://localhost/mydb';  // Dev default
  print('Using default DATABASE_URL: $dbUrl');
}
```

### "Can I use this in a mobile app?"

No — this is for **desktop apps only** (macOS, Linux, Windows). Mobile platforms have different secret storage:
- **iOS**: Use Keychain via `flutter_secure_storage`
- **Android**: Use Keystore via `flutter_secure_storage`

### "What about web apps?"

No — secrets should never be in client-side web code. Keep secrets on the backend.

## Testing

Suggest this test workflow to users:

1. **Store a test secret**:
   ```bash
   keysync set -p myapp TEST_SECRET=hello123
   ```

2. **Create a test file**:
   ```dart
   import 'package:keysync/keysync.dart';

   void main() async {
     final secret = await getSecret('TEST_SECRET', project: 'myapp');
     print('Retrieved: $secret');
     assert(secret == 'hello123');
     print('✓ Test passed');
   }
   ```

3. **Run the test**:
   ```bash
   dart run test_keysync.dart
   ```

4. **Clean up**:
   ```bash
   keysync rm -p myapp TEST_SECRET
   ```

## Example: Full Migration

**Before** (config.dart with hardcoded secrets):
```dart
class Config {
  static const apiKey = 'sk-1234567890';  // ❌
  static const dbUrl = 'postgres://user:pass@localhost/db';  // ❌
}
```

**After** (using keysync):

`lib/config.dart`:
```dart
import 'package:keysync/keysync.dart';

class Config {
  final String apiKey;
  final String dbUrl;

  Config._({required this.apiKey, required this.dbUrl});

  static Future<Config> load({String? environment}) async {
    return Config._(
      apiKey: await getSecret('API_KEY'),
      dbUrl: await getSecret('DATABASE_URL', project: 'myapp', environment: environment),
    );
  }
}
```

`lib/main.dart`:
```dart
import 'package:flutter/material.dart';
import 'config.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final config = await Config.load();

  runApp(MyApp(config: config));
}
```

**CLI setup**:
```bash
keysync set API_KEY=sk-1234567890
keysync set -p myapp DATABASE_URL=postgres://user:pass@localhost/db --env dev
keysync set -p myapp DATABASE_URL=postgres://user:pass@prod-host/db --env production
```

## Related Documentation

- [Main keysync README](../../README.md) — CLI and architecture overview
- [Client README](README.md) — Dart client API reference
- [Client Recipe](../../docs/new-client-library-recipe.md) — How clients are built

## Tips for AI Agents

1. **Always remind users to store secrets via CLI** — they can't retrieve what they haven't stored
2. **Suggest environment scopes** when users have staging/production environments
3. **Check platform** — only suggest for desktop Flutter apps, not mobile
4. **Add error handling** — always wrap `getSecret()` in try/catch
5. **Export for local dev** — remind users about `eval $(keysync export)` for smooth local development
