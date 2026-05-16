# keysync — Agent Instructions

## Project overview

keysync is a unified secret management tool that stores secrets in OS-native
keychains (macOS Keychain, Linux libsecret, Windows Credential Manager) and
syncs them to GitHub Secrets and deployment platforms (Vercel, Railway,
Supabase).

## Architecture

```
CLI (cobra) → Store (OS keychain) → GitHub Sync → Platform Sync
```

Secrets have two scopes:
- **Global**: shared across all projects (`keysync/global`)
- **Project**: scoped to a specific project (`keysync/project/<name>`)

Project scope overrides global when both exist with the same key.

## Client libraries

The `clients/` directory contains per-language libraries that read secrets
directly from the OS keychain with no dependency on the keysync binary:

| Library | Location | macOS | Linux | Windows |
|---------|----------|-------|-------|---------|
| Go | `clients/go/` | security CLI | secret-tool CLI | wincred |
| Python | `clients/python/` | security CLI | secret-tool CLI | ctypes Win32 API |
| TypeScript | `clients/node/` | security CLI | secret-tool CLI | Planned |
| Swift | `clients/swift/` | Security.framework | secret-tool CLI | Stub |

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

**Important:** Never read or expose secret values. Only key names and scope labels are needed for migration.

## Writing client libraries

When adding a new client library:

1. Use the platform's native keychain tooling — never depend on the `keysync` binary
2. Implement `getSecret` with project→global fallback
3. Implement `listSecrets` with scope/project filtering
4. Include a `MemoryStore` or equivalent for testing
5. Create README.md, CLAUDE.md, and AGENTS.md
6. Add to the directory listing in `clients/README.md`
