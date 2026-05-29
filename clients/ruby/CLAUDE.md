# keysync Ruby Client -- Claude Instructions

## Build & Test

```bash
cd clients/ruby
bundle install          # Install dependencies (minitest, rake)
bundle exec rake test   # Run all tests
```

Or without bundler:

```bash
ruby -Ilib -Itest test/test_*.rb
```

## Key files

```
lib/
  keysync.rb            # Main entry point + platform dispatch at load time
  keysync/
    version.rb          # VERSION constant
    errors.rb           # KeySyncError, SecretNotFoundError
    client.rb           # KeySync::Client -- get_secret, list_secrets with env var fallback + env scope
    credential.rb       # CredentialEntry Struct (service, account, scope, project, environment)
    service.rb          # Service name helpers (build + parse, env-aware)
    macos.rb            # macOS: Open3.capture3 → security CLI
    linux.rb            # Linux: Open3.capture3 → secret-tool CLI
    windows.rb          # Windows: PowerShell with inline C# (CredReadW from advapi32.dll)
test/
    test_helper.rb      # Minitest setup, $LOAD_PATH
    test_service.rb     # Service name construction + parsing tests
    test_errors.rb      # Error code + type tests
    test_client.rb      # Env var fallback + scope resolution tests
    test_keysync.rb     # Platform detection, CredentialEntry, Windows target conversion
```

## Conventions

- `RUBY_PLATFORM` for runtime platform detection (`/darwin/`, `/linux/`, `/mingw|mswin/`)
- Platform dispatch happens at require time in `lib/keysync.rb` -- sets `KeySync.platform_get` and `KeySync.platform_list`
- `KeySyncError < StandardError` with `code` attr (not_found, keychain_error, unsupported_platform)
- `SecretNotFoundError < KeySyncError` with `key` attr for the missing key
- `Client` module implements resolution: ENV first, then project scope, then global scope
- `Open3.capture3` for subprocess execution (returns stdout, stderr, status)
- `CredentialEntry` is a Struct with service, account, scope, project, environment
- No native gem dependencies -- pure Ruby with system CLI calls
- Minitest for testing (assert_* style)
- Module methods via `module_function` for stateless platform backends

### macOS notes

- The `security find-generic-password -s <service> -a <account> -w` command outputs the password to stderr on newer macOS versions. The implementation checks both stdout and stderr.
- Exit code 44 from `security` means "item not found".
- `list` parses `security dump-keychain` output, finding `class: "genp"` records with `keysync/` or `keysync_` service names.

### Linux notes

- Uses `secret-tool lookup keysync-service <service> keysync-key <account>` for get.
- Uses `secret-tool search --all keysync-service` for list.
- Handles `Errno::ENOENT` when secret-tool is not installed.
- Parses attribute lines in the form `keysync-service = value`.

### Windows notes

- PowerShell approach mirrors the TypeScript client's `windows.ts`.
- `read_cred_ps` generates a PowerShell script defining C# struct/class that P/Invokes `CredReadW`.
- `list_creds_ps` similarly uses `CredEnumerateW` with filter `keysync_*`.
- PowerShell output format: `userName\tsecret` (tab-separated).
- Target name conversion: slashes become underscores, then reversed on list.

## Migration: replacing ENV with keysync

```ruby
require "keysync"

# Global secret (shared across projects)
api_key = KeySync.get_secret("API_KEY")

# Project-scoped secret (falls back to global if no project match)
db_url = KeySync.get_secret("DATABASE_URL", project: "myapp")

# Environment-scoped secret (falls back: env → project → global)
staging_db = KeySync.get_secret("DATABASE_URL", project: "myapp", environment: "staging")

# List all secrets
globals = KeySync.list_secrets                                                    # everything
project = KeySync.list_secrets(project: "myapp")                                  # globals + myapp
staging = KeySync.list_secrets(project: "myapp", environment: "staging")          # globals + myapp + staging
```

## Adding new platforms

1. Create a new module in `lib/keysync/<platform>.rb` with `module_function` methods `get(service, account)` and `list()`.
2. In `lib/keysync.rb`, add a `when` branch to the platform dispatch `case RUBY_PLATFORM` block to set `platform_get` and `platform_list`.
3. Add tests in `test/test_keysync.rb` for the new platform's methods.
