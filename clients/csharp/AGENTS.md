# KeySync C# (.NET) Client

## Overview

The C# client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via Process.Start | Ready |
| Linux | `secret-tool` CLI via Process.Start | Ready |
| Windows | P/Invoke → advapi32.dll (CredReadW / CredEnumerateW) | Available (not fully tested on Windows) |

## API

```csharp
// Static class — no instantiation needed. Resolution order: env var → project scope → global scope.
public static string KeySyncClient.GetSecret(string key, string? project = null)

public static List<CredentialEntry> KeySyncClient.ListSecrets(string? project = null)
```

## Key design decisions

- **Static class** — `KeySyncClient` has no instance state; platform provider resolved once statically.
- **Runtime platform detection** via `RuntimeInformation.IsOSPlatform(OSPlatform.OSX)`, etc.
- **Windows uses P/Invoke** directly to advapi32.dll — no external NuGet packages needed. `CREDENTIAL` struct with `StructLayout(LayoutKind.Sequential)` is 80 bytes on x64.
- **`KeySyncException` with `ErrorCode` property** — supports C# pattern matching with `when` clauses (e.g., `catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.NotFound)`).
- **`KeySyncError` enum** — NotFound, KeychainError, UnsupportedPlatform.
- **Nullable reference types** enabled throughout the project.
- **Target framework** `net8.0` with xUnit test project.
- **Service naming** matches keysync convention: `keysync/global` and `keysync/project/<name>`.
- **Windows target names** converted bidirectionally: `keysync/global` ⇔ `keysync_global`, `keysync/project/my-app` ⇔ `keysync_project_my-app`.
- **33 xUnit tests** across 6 test classes covering service names, error types, env var fallback, Windows target conversion, credential entry equality, and struct layout verification.
