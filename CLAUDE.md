# keysync — Project Instructions for Claude Code

## Agent workflow

Read **[docs/coding-assistants.md](docs/coding-assistants.md)** and **[`.agents/skills/keysync-agent/SKILL.md`](.agents/skills/keysync-agent/SKILL.md)** before helping with keysync.

**Do not run `keysync set` or `keysync migrate` for the user** — secret values must not enter chat. Give terminal commands for the user to run. They may share migrate output JSON (key names/scopes only) for code updates.

**OK for assistants:** `.keysync.json` edits, `push --dry-run`, init scaffolding, client migration, export docs (user runs `eval $(keysync export …)` locally).

When editing **this repository**: do not run `keysync init` or `keysync push` here; `.keysync.json` is an example template only. Use `make test` — never store real user secrets in the keysync repo.

## Repository structure

```
cmd/keysync/main.go     # CLI entry point
internal/commands/      # All CLI commands (cobra)
internal/config/        # .keysync.json loader
internal/crypto/        # NaCl box encryption (Curve25519+XSalsa20-Poly1305)
internal/github/        # GitHub Secrets client via `gh` CLI
internal/platforms/     # Vercel, Railway, Supabase API clients
internal/store/         # OS keychain backends + fallback store
client/                 # Legacy Go client (shells out to `keysync get`)
clients/                # Native language client libraries
  go/                   #   Go client (build-tagged, direct keychain)
  node/                 #   TypeScript/Node.js client
  python/               #   Python client
  dart/                 #   Dart/Flutter client
  swift/                #   Swift client
  java/                 #   Java client (Maven, JNA for Windows)
  csharp/               #   C# client (.NET 8.0, P/Invoke for Windows)
  rust/                 #   Rust client (cargo, windows-sys crate)
  cpp/                  #   C++ client (CMake C++17, wincred.h)
  ruby/                 #   Ruby client (pure Ruby, PowerShell for Windows)
docs/                   # Tutorials and guides
```

## Build commands

```bash
make build          # go build -o ./bin/keysync ./cmd/keysync
make build-signed   # macOS: Developer ID sign (Always Allow persists across rebuilds)
export PATH="$PWD/bin:$PATH"
keysync trust       # macOS: after every make build or binary copy
make test           # Run all non-platform tests
make test-platform  # Run platform client tests
make clean          # rm -rf ./bin/
```

## Key conventions

- CLI uses `cobra` — commands in `internal/commands/`, registered in `root.go`
- OS keychain stores use build tags (`//go:build darwin`, `//go:build linux`, `//go:build windows`)
- Service naming: `keysync/global` for global scope, `keysync/project/<name>` for project scope
- Never commit `.env` files or secret values
- Client libraries in `clients/` access the OS keychain directly (no dependency on the keysync binary)
- Each client library has its own README.md, CLAUDE.md, and AGENTS.md

## Open-source safety

The `keysync push` command includes validation to prevent users from accidentally syncing secrets to the keysync repository itself:

- Rejects repo names matching `dipockdas/keysync` (`internal/commands/push.go`)
- Rejects placeholder values like `YOUR_ORG/YOUR_REPO`
- Example `.keysync.json` uses placeholder values, not real repo names

**Why this matters**: The sync command pushes secrets to GitHub Secrets via the `gh` CLI. Without validation, users who copy the example config without updating it would push their secrets to the keysync repo itself, creating a security risk.

**Push is local-only**: The push command runs on the user's machine and reads from their OS keychain. There is no CI/CD workflow that syncs from GitHub Secrets to platforms — all pushing happens locally.

## User quick start (three stages)

1. **Local** — `set` / `get` / `list` (no `.keysync.json` required)
2. **Project** — `init`, optional `migrate`
3. **Cloud** — `push --dry-run`, then `push` (needs `gh`)

Full guide: [docs/getting-started.md](docs/getting-started.md). README: [README.md](README.md#first-steps-any-install-method).

**Scopes:** With `-p`, `set`/`push`/`list` default to `dev` unless `--env` is passed; `--env ""` = project-wide. `get`/`export` use `--env` only when explicitly passed.

## Platform configuration

All platforms use the **generic engine** in `.keysync.json` with `"type": "cli"` or `"type": "http"` (including Vercel, Railway, Supabase). First-party configs: [docs/platform-configs/](docs/platform-configs/). Examples: [docs/platform-examples.md](docs/platform-examples.md).

GitHub can use the structured `platforms.github` block (secrets vs variables) — see [docs/configuration.md](docs/configuration.md).

**Example `.keysync.json` with mixed platforms**:
```json
{
  "repos": {
    "myorg/myapp": {
      "project": "myapp",
      "platforms": {
        "github": {
          "type": "cli",
          "command": "gh secret set {KEY} --repo {REPO}",
          "stdin": "{VALUE}",
          "token_env": "GH_TOKEN",
          "config": {"REPO": "myorg/myapp"}
        },
        "vercel": {
          "type": "http",
          "endpoint": "https://api.vercel.com/v9/projects/{PROJECT_ID}/env",
          "method": "POST",
          "token_env": "VERCEL_TOKEN",
          "headers": { "Authorization": "Bearer {TOKEN}", "Content-Type": "application/json" },
          "body": { "key": "{KEY}", "value": "{VALUE}", "target": ["production"], "type": "encrypted" },
          "template_vars": { "PROJECT_ID": "prj_abc123" }
        },
        "cloudflare": {
          "type": "cli",
          "command": "wrangler secret put {KEY}",
          "stdin": "{VALUE}",
          "token_env": "CLOUDFLARE_API_TOKEN"
        }
      }
    }
  }
}
```

Config schema: `internal/config/config.go` defines:
- `PlatformConfig` - legacy struct with Vercel/Railway/Supabase fields
- `GenericPlatformConfig` - new struct for CLI/HTTP platforms with type, command, endpoint, etc.

## Migration: replacing .env with keysync

When helping a user who has run `keysync migrate` and needs to update source code:

### Workflow

1. **Get the migrated keys** from the `---MIGRATION_RESULT_START---` JSON block in the user's `keysync migrate` output. Each entry has `key`, `scope` ("global" or "project"), and `project`.

2. **Find references** across the project:
   ```bash
   rg "process\.env\.|os\.Getenv|os\.environ|ENV\[" --glob '*.{ts,tsx,js,jsx,go,py,rb,swift}'
   ```

3. **Replace** each reference using the patterns below. Use global accessor for `scope: "global"`, project accessor for `scope: "project"`.

4. **Remove dotenv imports** — `import 'dotenv/config'`, `require('dotenv').config()`, `load_dotenv()`, `from dotenv import load_dotenv`.

5. **Clean up** — add `.env*` to `.gitignore`. The `.env` file is safe to delete (keysync never modifies it).

### Replacement patterns

**TypeScript** (install `@keysync/node`):
```typescript
import { getSecret } from '@keysync/node';
const key = await getSecret('API_KEY');                    // global
const key = await getSecret('DB_URL', 'myapp');            // project
const [a, b] = await Promise.all([getSecret('A'), getSecret('B')]);
```

**Go** (`go get github.com/dipockdas/keysync/clients/go`):
```go
import "github.com/dipockdas/keysync/clients/go"
val, err := keysync.GetGlobal("API_KEY")                  // global
val, err := keysync.GetSecret("myapp", "DB_URL")          // project
```

**Python** (`pip install keysync`):
```python
from keysync import get_secret
key = get_secret("API_KEY")                               # global
key = get_secret("DB_URL", project="myapp")               # project
```

**Java** (Maven: `io.keysync:keysync`):
```java
KeySyncClient client = KeySyncClient.getInstance();
String key = client.getSecret("API_KEY");                    // global
String key = client.getSecret("DB_URL", "myapp");            // project
```

**C# (.NET)** (NuGet: `KeySync`):
```csharp
using KeySync;
string key = KeySyncClient.GetSecret("API_KEY");             // global
string key = KeySyncClient.GetSecret("DB_URL", "myapp");     // project
```

**Rust** (Cargo: `keysync`):
```rust
use keysync::get_secret;
let key = get_secret("API_KEY", None)?;                      // global
let key = get_secret("DB_URL", Some("myapp"))?;              // project
```

**C++** (`#include <keysync/keysync.hpp>`):
```cpp
std::string key = keysync::getSecret("API_KEY");             // global
std::string key = keysync::getSecret("DB_URL", "myapp");     // project
```

**Ruby** (Gemfile: `gem "keysync"`):
```ruby
require "keysync"
key = KeySync.get_secret("API_KEY")                          # global
key = KeySync.get_secret("DB_URL", project: "myapp")         # project
```

**Never inspect or log secret values.** Only key names are needed for migration.
