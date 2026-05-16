# keysync — Project Instructions for Claude Code

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
  swift/                #   Swift client
docs/                   # Tutorials and guides
```

## Build commands

```bash
make build          # go build -o ./bin/keysync ./cmd/keysync
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

**Never inspect or log secret values.** Only key names are needed for migration.
