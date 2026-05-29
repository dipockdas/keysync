# keysync — Agent Instructions

## Using keysync safely (read first)

Full policy: **[docs/coding-assistants.md](docs/coding-assistants.md)**. Agent skill: **[`.agents/skills/keysync-agent/SKILL.md`](.agents/skills/keysync-agent/SKILL.md)**.

### User-initiated (do not run for the user)

- **`keysync set`** — requires the secret in chat or terminal; tell the user to run it locally. Never ask them to paste secret values into the conversation.
- **`keysync migrate`** — reads `.env` on their machine; user runs it. For code help they may paste only the migrate **result JSON** (key names + scopes), not `.env` contents.
- **`keysync push`** (real push) — user runs after reviewing `keysync push --dry-run`.

### Assistant-friendly

- Edit **`.keysync.json`** (repo slugs, platform IDs, generic platform configs — no tokens).
- **`keysync push --dry-run`**, **`keysync init`** (scaffold), **`keysync doctor`**, **`keysync trust`** instructions.
- **Client library migration** and **`keysync export`** documentation (user runs export in their shell).
- **Never** log, commit, or paste secret values — key names and scopes only.
- **Never `keysync init` or `keysync push` in the `dipockdas/keysync` repo** — example config only.

Other rules: **`set` is local only** (use `push` for cloud); with **`-p`**, `set`/`push`/`list` default to **`dev`** unless `--env` is set (`--env ""` = project-wide).

User flow: [docs/getting-started.md](docs/getting-started.md).

## Project overview

keysync is a unified secret management tool that stores secrets in OS-native
keychains (macOS Keychain, Linux libsecret, Windows Credential Manager) and
syncs them to GitHub Secrets and deployment platforms via declarative configs
(Vercel, Railway, Supabase, Cloudflare, GitLab, and more — see
[docs/platform-examples.md](docs/platform-examples.md)).

## Architecture

```
CLI (cobra) → Store (OS keychain) → GitHub Sync → Platform Sync
```

Secrets have three scopes (with precedence from lowest to highest):
- **Global**: shared across all projects (`keysync/global`)
- **Project**: scoped to a specific project (`keysync/project/<name>`)
- **Environment**: scoped to a project + environment (`keysync/project/<name>/env/<env>`)

Higher scopes override lower scopes when the same key exists at multiple levels.

## Adding platform support

Keysync uses a **generic platform engine** for deployment platform support. All platforms (including Vercel, Railway, Supabase) should use declarative JSON configs with `"type": "cli"` or `"type": "http"`.

**⚠️ Deprecation**: Built-in hardcoded implementations (Vercel, Railway, Supabase) are deprecated and will be removed in keysync 2.0. Use first-party configs from `docs/platform-configs/` instead.

### Adding a generic platform (AI workflow)

When a user asks to add a new platform (e.g., "add Cloudflare Workers support"):

1. **Search for the platform's API/CLI documentation**
2. **Determine if it's CLI-based or HTTP API-based**
3. **Generate the JSON config** following the pattern below
4. **Add to `.keysync.json` under `"platforms"`**
5. **No Go code needed** - test immediately with `keysync push`

**CLI-based platform template**:
```json
{
  "platform-name": {
    "type": "cli",
    "command": "cli-tool command-with {KEY} placeholder",
    "stdin": "{VALUE}",
    "token_env": "PLATFORM_TOKEN_ENV_VAR",
    "template_vars": {
      "CUSTOM_FIELD": "custom-value"
    },
    "validation": {
      "command_check": "cli-tool --version"
    }
  }
}
```

**HTTP-based platform template**:
```json
{
  "platform-name": {
    "type": "http",
    "endpoint": "https://api.platform.com/path/{CONFIG_VAR}",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer {TOKEN}"
    },
    "body": {
      "key": "{KEY}",
      "value": "{VALUE}"
    },
    "token_env": "PLATFORM_TOKEN_ENV_VAR",
    "template_vars": {
      "CONFIG_VAR": "value-from-config"
    }
  }
}
```

**Built-in placeholders**: `{KEY}` (secret name), `{VALUE}` (secret value), `{TOKEN}` (platform API token)

**User-defined placeholders**: Any field in `template_vars` can be used as `{FIELD_NAME}` in endpoint, headers, or body

See [docs/platform-examples.md](docs/platform-examples.md) for complete examples: GitHub, Cloudflare, GitLab, Netlify, Render, Heroku, Fly.io, DigitalOcean, and more.

## Client libraries

The `clients/` directory contains per-language libraries that read secrets
directly from the OS keychain with no dependency on the keysync binary:

| Library | Location | macOS | Linux | Windows | Status |
|---------|----------|-------|-------|---------|--------|
| Go | `clients/go/` | security CLI | secret-tool CLI | wincred library | Ready |
| Python | `clients/python/` | security CLI | secret-tool CLI | ctypes Win32 API | Ready |
| TypeScript | `clients/node/` | security CLI | secret-tool CLI | PowerShell + Win32 | Ready (macOS/Linux) |
| Dart/Flutter | `clients/dart/` | security CLI | secret-tool CLI | Not planned | Ready (macOS/Linux) |
| Swift | `clients/swift/` | Security.framework | secret-tool CLI | Not planned | Ready (macOS/Linux) |
| Java | `clients/java/` | security CLI | secret-tool CLI | JNA → Win32 API | Available (Windows: not fully tested) |
| C# (.NET) | `clients/csharp/` | security CLI | secret-tool CLI | P/Invoke → Win32 API | Available (Windows: not fully tested) |
| Rust | `clients/rust/` | security CLI | secret-tool CLI | windows-sys → Win32 API | Available (Windows: not fully tested) |
| C++ | `clients/cpp/` | security CLI | secret-tool CLI | Win32 API (wincred.h) | Available (Windows: not fully tested) |
| Ruby | `clients/ruby/` | security CLI | secret-tool CLI | PowerShell + inline C# | Available (Windows: not fully tested) |

Each library follows the same pattern:
1. Platform detection at import time
2. `getSecret(key, project?)` with project→global fallback
3. `listSecrets(scope?, project?)` with filtering
4. Service naming: `keysync/global`, `keysync/project/<name>`

## Helping users migrate from .env to keysync

When a user has run `keysync migrate` and needs their source code updated:

### Workflow

1. **Get the migrated key list** — If the user provides the `---MIGRATION_RESULT_START---` JSON block from `keysync migrate` output, use it. Otherwise ask: "What keys were migrated and what scope did you choose for each?"

2. **Find .env references** — Search the project for these patterns:
   ```bash
   rg "process\.env\.|os\.Getenv|os\.environ|ENV\[" --glob '*.{ts,tsx,js,jsx,go,py,rb}'
   ```

3. **Replace each reference** — Use the appropriate client library below. Match the scope: use global accessor for global secrets, project accessor for project-scoped ones.

4. **Remove dotenv imports** — Delete `import 'dotenv/config'`, `require('dotenv').config()`, `load_dotenv()`, `from dotenv import load_dotenv`.

5. **Clean up** — Add `.env*` to `.gitignore`. Tell the user the `.env` file is safe to delete (keysync never modifies it).

### Replacement patterns by language

**TypeScript / Node.js** — install `@keysync/node`, then:

```typescript
// Global secret
import { getSecret } from '@keysync/node';
const apiKey = await getSecret('API_KEY');

// Project-scoped secret (falls back to global if no project match)
const dbUrl = await getSecret('DATABASE_URL', 'myapp');

// Multiple secrets
const [key, url] = await Promise.all([
  getSecret('API_KEY'),
  getSecret('DATABASE_URL', 'myapp'),
]);
```

**Go** — add `github.com/dipockdas/keysync/clients/go` to `go.mod`, then:

```go
import "github.com/dipockdas/keysync/clients/go"

// Global secret
apiKey, err := keysync.GetGlobal("API_KEY")

// Project-scoped secret
dbUrl, err := keysync.GetSecret("myapp", "DATABASE_URL")
```

**Python** — install `keysync`, then:

```python
from keysync import get_secret

# Global secret
api_key = get_secret("API_KEY")

# Project-scoped secret
db_url = get_secret("DATABASE_URL", project="myapp")
```

**Swift** — add `KeySync` package dependency, then:

```swift
import KeySync

// Global secret
let apiKey = try KeySyncClient.shared.getSecret("API_KEY")

// Project-scoped secret
let dbUrl = try KeySyncClient.shared.getSecret("DATABASE_URL", project: "myapp")
```

**Java** — add `io.keysync:keysync` Maven dependency, then:

```java
import io.keysync.KeySyncClient;

KeySyncClient client = KeySyncClient.getInstance();

// Global secret
String apiKey = client.getSecret("API_KEY");

// Project-scoped secret
String dbUrl = client.getSecret("DATABASE_URL", "myapp");
```

**C# (.NET)** — add `KeySync` package reference, then:

```csharp
using KeySync;

// Global secret
string apiKey = KeySyncClient.GetSecret("API_KEY");

// Project-scoped secret
string dbUrl = KeySyncClient.GetSecret("DATABASE_URL", "myapp");
```

**Rust** — add `keysync` to `Cargo.toml`, then:

```rust
use keysync::get_secret;

// Global secret
let api_key = get_secret("API_KEY", None)?;

// Project-scoped secret
let db_url = get_secret("DATABASE_URL", Some("myapp"))?;
```

**C++** — add `#include <keysync/keysync.hpp>`, then:

```cpp
// Global secret
std::string apiKey = keysync::getSecret("API_KEY");

// Project-scoped secret
std::string dbUrl = keysync::getSecret("DATABASE_URL", "myapp");
```

**Ruby** — add `gem "keysync"` to Gemfile, then:

```ruby
require "keysync"

# Global secret
api_key = KeySync.get_secret("API_KEY")

# Project-scoped secret
db_url = KeySync.get_secret("DATABASE_URL", project: "myapp")
```

**Important:** Never read or expose secret values. Only key names and scope labels are needed for migration.

## Writing client libraries

When adding a new client library:

1. Use the platform's native keychain tooling — never depend on the `keysync` binary
2. Implement `getSecret` with project→global fallback
3. Implement `listSecrets` with scope/project filtering
4. Include a `MemoryStore` or equivalent for testing
5. Create README.md, CLAUDE.md, and AGENTS.md
6. Add to the directory listing in `clients/README.md`
