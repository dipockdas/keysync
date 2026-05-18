# KeySync Ruby Client

Read secrets from the OS-native keychain -- zero dependency on the `keysync` binary.

Each platform uses its native keychain tooling:
- **macOS**: `security` CLI (built-in)
- **Linux**: `secret-tool` CLI (libsecret)
- **Windows**: PowerShell with inline C# P/Invoke to `advapi32.dll` (CredReadW / CredEnumerateW) -- available, not fully tested on Windows

## Installation

Add this line to your application's Gemfile:

```ruby
gem "keysync", path: "path/to/keysync/clients/ruby"
```

Or install it yourself:

```bash
gem build keysync.gemspec
gem install keysync-*.gem
```

## Usage

```ruby
require "keysync"

# Get a global secret (shared across projects)
api_key = KeySync.get_secret("API_KEY")

# Get a project-scoped secret (falls back to global if not found)
db_url = KeySync.get_secret("DATABASE_URL", project: "myapp")

# Get an environment-scoped secret (falls back: env → project → global)
staging_db = KeySync.get_secret("DATABASE_URL", project: "myapp", environment: "staging")

# List all global secrets
globals = KeySync.list_secrets

# List project secrets (includes global fallback)
project = KeySync.list_secrets(project: "myapp")

# List environment secrets (includes globals + project)
staging = KeySync.list_secrets(project: "myapp", environment: "staging")
```

## Secret Resolution Order

Every `get_secret` call follows this order:

1. **Environment variable** (`ENV[key]`) -- for cloud/CI where the platform injects env vars
2. **Environment scope** (if environment is provided) -- `keysync/project/<name>/env/<env>`. Use for environment-specific overrides (e.g., staging vs. production).
3. **Project scope** (if project is provided) -- `keysync/project/<name>`
4. **Global scope** -- `keysync/global`

## Error Types

All errors inherit from `KeySync::KeySyncError < StandardError` and carry a machine-readable `code`:

| Error class | Code | When |
|---|---|---|
| `KeySync::SecretNotFoundError` | `not_found` | Secret not found in any scope |
| `KeySync::KeySyncError` | `keychain_error` | Keychain tool failed or not found |
| `KeySync::KeySyncError` | `unsupported_platform` | Running on an unsupported OS |

```ruby
begin
  KeySync.get_secret("MISSING_KEY")
rescue KeySync::SecretNotFoundError => e
  puts "Missing: #{e.key}"       # "MISSING_KEY"
  puts "Code: #{e.code}"         # "not_found"
rescue KeySync::KeySyncError => e
  puts "Error code: #{e.code}"   # "keychain_error" or "unsupported_platform"
end
```

## Platform Requirements

### macOS

The `security` CLI is built into macOS. No additional installation required.

### Linux

Requires `libsecret-tools` (provides the `secret-tool` CLI):

```bash
# Debian / Ubuntu
sudo apt install libsecret-tools

# Fedora
sudo dnf install libsecret

# Arch
sudo pacman -S libsecret
```

### Windows

Requires PowerShell (built into Windows). No additional installation required.

The client uses PowerShell's ability to compile inline C# code that calls `CredReadW` and `CredEnumerateW` from `advapi32.dll`. No native gem dependencies are needed.

## Development

```bash
cd clients/ruby

# Install dependencies
bundle install

# Run tests
bundle exec rake test
# or
ruby -Ilib -Itest test/test_*.rb
```

## License

MIT
