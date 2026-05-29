# Dart/Flutter Client — Instructions for Claude Code

## Quick Reference

**Purpose**: Dart/Flutter library for reading secrets from environment variables or OS keychain

**Platforms**: macOS, Linux, Windows desktop (not mobile iOS/Android)

**Key files**:
- `lib/keysync.dart` — Main library implementation
- `example/main.dart` — Usage examples
- `pubspec.yaml` — Package metadata and dependencies

## Directory Structure

```
clients/dart/
├── lib/
│   └── keysync.dart          # Main library (single file)
├── example/
│   └── main.dart             # Usage examples
├── pubspec.yaml              # Package metadata
├── README.md                 # User documentation
├── CLAUDE.md                 # This file
└── AGENTS.md                 # AI agent integration guide
```

## How It Works

### Resolution Strategy

1. **Environment variables first** (primary path)
   - Check `Platform.environment[key]`
   - If found and non-empty, return immediately
   - No keychain access in production

2. **OS keychain fallback** (development)
   - Try environment scope (if project + environment provided)
   - Try project scope (if project provided)
   - Try global scope (default)
   - Throw `KeysyncException` if not found

### Platform Implementations

**macOS** (`_getMacOS`):
- Uses `security find-generic-password -s <service> -a <key> -w`
- Service name: `keysync/global`, `keysync/project/<name>`, or `keysync/project/<name>/env/<env>`
- Returns password text via `-w` flag

**Linux** (`_getLinux`):
- Uses `secret-tool lookup service <service> account <key>`
- Same service naming convention as macOS
- Requires libsecret-tools installed

**Windows** (`_getWindows`):
- Uses PowerShell to call Windows Credential Manager
- Tagged format: `keysync|s=<scope>|p=<project>|e=<environment>|k=<key>`
- Percent-encodes project/environment/key names
- Falls back to legacy underscore-delimited format for backward compatibility
- Uses `Get-StoredCredential` cmdlet to retrieve credentials
- Extracts password from `SecureString` using Marshal

## Development Commands

```bash
# Get dependencies
dart pub get

# Run example
dart run example/main.dart

# Analyze code
dart analyze

# Format code
dart format .

# Run tests (when tests are added)
dart test
```

## Code Conventions

- **Async/await**: All secret retrieval is async (uses `Process.run`)
- **Error handling**: Throw `KeysyncException` with descriptive messages
- **Documentation**: Use `///` doc comments for public APIs
- **Formatting**: Follow Dart style guide (use `dart format`)
- **Linting**: Use `lints` package (recommended Dart lints)

## Testing Locally

Store secrets using the keysync CLI:

```bash
# Global secret
keysync set MY_SECRET=test123

# Project-scoped secret
keysync set -p myapp DATABASE_URL=postgres://localhost/mydb

# Environment-scoped secret
keysync set -p myapp API_KEY=prod-key-123 --env production
```

Run the example:

```bash
cd clients/dart
dart run example/main.dart
```

## Common Tasks

### Adding a New Feature

1. Update `lib/keysync.dart` with the new function
2. Add doc comments explaining usage
3. Update `example/main.dart` with a usage example
4. Update `README.md` API Reference section
5. Run `dart analyze` and `dart format .`

### Debugging Platform-Specific Issues

**macOS**:
```bash
# Test security CLI manually
security find-generic-password -s keysync/global -a MY_SECRET -w
```

**Linux**:
```bash
# Test secret-tool manually
secret-tool lookup service keysync/global account MY_SECRET
```

**Windows**:
```powershell
# Test credential retrieval manually
$cred = Get-StoredCredential -Target "keysync|s=global|k=MY_SECRET" -Type Generic
$ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($cred.Password)
try {
  [System.Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr)
} finally {
  [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)
}
```

### Windows Credential Target Name Format

**v2 tagged format** (current):
```
keysync|s=global|k=MY_SECRET
keysync|s=project|p=myapp|k=DATABASE_URL
keysync|s=env|p=myapp|e=production|k=API_KEY
```

**Legacy underscore-delimited** (backward compatibility):
```
keysync_global_MY_SECRET
keysync_project_myapp_DATABASE_URL
keysync_project_myapp_env_production_API_KEY
```

The client tries v2 format first, falls back to legacy if not found.

## Known Limitations

1. **Mobile platforms not supported**: iOS/Android have different secret storage (Keychain on iOS, Keystore on Android). This library is for desktop apps only.

2. **Synchronous keychain access**: Dart doesn't support synchronous file I/O or process execution well, so all operations are async.

3. **Windows PowerShell dependency**: Requires PowerShell on Windows (included in Windows 10+). Alternative would be FFI to Win32 API, which adds complexity.

4. **No test coverage yet**: The client library doesn't have automated tests. Manual testing required on all three platforms.

## Architecture Notes

- **Single file library**: All code in `lib/keysync.dart` — Dart convention for simple packages
- **No external dependencies**: Uses only `dart:io` from the standard library
- **Process.run() for shell commands**: Consistent across all platforms
- **Percent encoding**: Custom implementation for Windows target names (no built-in URL encoding needed)

## Troubleshooting

### "Unsupported platform" error

**Cause**: Running on iOS, Android, or other non-desktop platform.

**Solution**: This library only supports macOS, Linux, and Windows desktop. For mobile, use platform-specific secret storage APIs.

### PowerShell errors on Windows

**Cause**: `Get-StoredCredential` cmdlet not found.

**Solution**: Install CredentialManagement module:
```powershell
Install-Module -Name CredentialManagement -Force
```

### Keychain prompts on macOS

**Cause**: App is not signed or "Always Allow" not clicked.

**Solution**:
- Sign your Flutter app with a Developer ID certificate
- Or use `keysync export` to load secrets into env vars before app starts

## Related Files

- [Main README](../../README.md) — keysync overview
- [AGENTS.md](../../AGENTS.md) — AI integration guide
- [Client Recipe](../../docs/new-client-library-recipe.md) — How to build clients
