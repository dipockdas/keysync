# keysync Ruby Client

## Overview

The Ruby client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via Open3.capture3 | Ready |
| Linux | `secret-tool` CLI via Open3.capture3 | Ready |
| Windows | PowerShell with inline C# (CredReadW / CredEnumerateW) | Available (not fully tested on Windows) |

## API

```ruby
# Module methods — no instantiation needed. Resolution order: ENV → project scope → global scope.
KeySync.get_secret(key, project: nil)       # raises SecretNotFoundError if not found
KeySync.list_secrets(project: nil)          # returns Array<CredentialEntry>
```

## Key design decisions

- **Runtime platform detection** via `RUBY_PLATFORM` — regex match on `/darwin/`, `/linux/`, `/mingw|mswin/`. Platform backend procs set at require time.
- **Windows mirrors the TypeScript client** — PowerShell compiles inline C# at runtime to P/Invoke `CredReadW` / `CredEnumerateW` from `advapi32.dll`. No native gem dependencies or compilation needed.
- **Zero native dependencies** — pure Ruby with system CLI calls. Works with any Ruby 3.0+ installation.
- **`KeySyncError < StandardError`** with `code` attr (not_found, keychain_error, unsupported_platform). `SecretNotFoundError < KeySyncError` adds `key` attr.
- **`Open3.capture3`** for subprocess execution — returns stdout, stderr, and Process::Status in one call.
- **macOS stderr handling** — `security` CLI on macos 13+ sends password to stderr. Implementation checks both streams.
- **`CredentialEntry` Struct** — immutable value object with service, account, scope, project fields, plus `from_raw` factory method.
- **Minitest** for testing with `assert_*` style assertions. 32 tests across 5 test files.
- **Service naming** matches keysync convention: `keysync/global` and `keysync/project/<name>`.
- **Module functions** via `module_function` for stateless platform backends.
