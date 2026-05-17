# KeySync C# Client -- Claude Instructions

## Build & Test

```bash
cd clients/csharp
dotnet build          # Build the library
dotnet test           # Run all tests
```

## Key files

```
Keysync.csproj                   # .NET 8.0 class library
KeySyncClient.cs                 # Public API (GetSecret, ListSecrets), ServiceName helpers
KeySyncError.cs                  # Error type enum (NotFound, KeychainError, UnsupportedPlatform)
KeySyncException.cs              # Exception class wrapping KeySyncError
IKeychainProvider.cs             # Interface for platform backends
MacKeychainProvider.cs           # macOS: Process.Start → security CLI
LinuxKeychainProvider.cs         # Linux: Process.Start → secret-tool CLI
WindowsKeychainProvider.cs       # Windows: P/Invoke to advapi32.dll
CredentialEntry.cs               # Record type for list results
KeySync.Tests/                   # xUnit test project
  KeySync.Tests.csproj
  KeySyncClientTests.cs          # Service name, error types, env var, Windows target, struct layout tests
```

## Conventions

- Target framework: `net8.0`
- Nullable reference types enabled throughout
- Platform selection via `RuntimeInformation.IsOSPlatform()`
- `KeySyncClient` is a static class (no need for instantiation)
- `IKeychainProvider` interface for platform abstraction
- `KeySyncException` with `ErrorCode` property for pattern matching
- Service naming: `keysync/global`, `keysync/project/<name>`

### Windows P/Invoke

- `DllImport` with `CharSet = CharSet.Unicode` and `SetLastError = true`
- `CREDENTIAL` struct with `StructLayout(LayoutKind.Sequential)` and `IntPtr` fields
- `CredReadW` for single credential retrieval
- `CredEnumerateW` for listing
- `CredFree` to release allocated memory
- Target names: `keysync_global`, `keysync_project_myapp`

### Platform-specific notes

- **macOS**: `security` CLI writes password to stderr on newer macOS. Capture both
  stdout and stderr. Exit code 44 means "not found".
- **Linux**: `secret-tool` uses `service` and `account` attribute names. Exit code 1
  means "not found". `search` subcommand lists all matching entries.
- **Windows**: The `CREDENTIAL` struct is 80 bytes on x64. Use `Marshal.PtrToStringUni`
  to read `CredentialBlob` (the password) based on `CredentialBlobSize`.

## Migration: replacing Environment.GetEnvironmentVariable with KeySync

```csharp
using KeySync;

// Global secret (shared across projects)
string apiKey = KeySyncClient.GetSecret("API_KEY");

// Project-scoped secret (falls back to global if no project match)
string dbUrl = KeySyncClient.GetSecret("DATABASE_URL", "myapp");

// List all secrets
var globals = KeySyncClient.ListSecrets();
var project = KeySyncClient.ListSecrets("myapp");
```
