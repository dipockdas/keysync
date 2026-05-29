# KeySync -- C# / .NET Client Library

Retrieve secrets managed by [keysync](https://github.com/dipockdas/keysync)
directly from the OS keychain, with no dependency on the `keysync` binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS    | `security` CLI (built-in) | Ready |
| Linux    | `secret-tool` CLI (libsecret) | Ready |
| Windows  | P/Invoke to `advapi32.dll` (Credential Manager) | Available (not fully tested on Windows) |

## Requirements

- .NET 8.0+
- macOS, Linux, or Windows

On Linux, `libsecret-tools` must be installed for the `secret-tool` CLI.

## Installation

### NuGet (TBD)

```bash
dotnet add package KeySync
```

### Local project reference

Add the project to your solution:

```bash
dotnet sln add clients/csharp/Keysync.csproj
```

Then reference it from your application:

```bash
dotnet add reference ../path/to/clients/csharp/Keysync.csproj
```

## Usage

```csharp
using KeySync;

// Get a global secret
string apiKey = KeySyncClient.GetSecret("API_KEY");

// Get a project-scoped secret (falls back to global)
string dbUrl = KeySyncClient.GetSecret("DATABASE_URL", "myapp");

// Get an environment-scoped secret (falls back: env → project → global)
string stagingDb = KeySyncClient.GetSecret("DATABASE_URL", "myapp", "staging");

// List all global secrets
List<CredentialEntry> globals = KeySyncClient.ListSecrets();

// List project secrets (includes global fallback)
List<CredentialEntry> project = KeySyncClient.ListSecrets("myapp");

// List environment secrets (includes globals + project)
List<CredentialEntry> staging = KeySyncClient.ListSecrets("myapp", "staging");
```

## Error handling

```csharp
using KeySync;

try
{
    string value = KeySyncClient.GetSecret("DATABASE_URL", "myapp");
    // use value
}
catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.NotFound)
{
    // Secret doesn't exist in any scope
}
catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.KeychainError)
{
    // OS-level keychain error
}
catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.UnsupportedPlatform)
{
    // Platform not supported
}
```

## Testing

```bash
cd clients/csharp/KeySync.Tests
dotnet test
```

## How it works

Secrets are stored in the OS keychain with this naming convention:

| Scope | Service Name | Account Name |
|-------|-------------|--------------|
| Global | `keysync/global` | key name (e.g. `DATABASE_URL`) |
| Project | `keysync/project/<name>` | key name |
| Environment | `keysync/project/<name>/env/<env>` | key name |

The library accesses the OS keychain directly:

- **macOS**: `security find-generic-password -s keysync/global -a DATABASE_URL -w`
  Captures both stdout and stderr (newer macOS writes the password to stderr).

- **Linux**: `secret-tool lookup service keysync/global account DATABASE_URL`
  Requires `libsecret-tools`.

- **Windows**: P/Invoke to `CredReadW` / `CredEnumerateW` in `advapi32.dll`.
  Target names use underscores: `keysync_global`, `keysync_project_myapp`.

No dependency on the `keysync` binary. Read operations work standalone as
long as secrets have been stored (via `keysync set` or any other method that
writes to the same keychain entries).

## Resolution order

Every `GetSecret` call follows this order:

1. Check environment variable first (`Environment.GetEnvironmentVariable(key)`)
   -- for cloud/CI where the platform injects env vars directly.
2. If `environment` is provided, check the environment-scoped service
   (`keysync/project/<name>/env/<env>`). Use for environment-specific overrides.
3. If `project` is provided, check the project-scoped service
   (`keysync/project/<name>`).
4. Fall back to the global-scoped service (`keysync/global`).

## Testing

Tested on Windows ARM64 and Windows AMD64 (x86-64). 34 xUnit tests across
6 test classes, all passing. Run with:

```bash
cd clients/csharp
dotnet test
```
