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

```bash
# The migrate output lists migrated keys — reference these
keysync migrate --file .env --project <name>

# Search for .env usage patterns
rg "process\.env\.|os\.Getenv|os\.environ" --type-add 'src:*.{ts,js,go,py}' -t src
```

**Replacement patterns** (never expose secret values, only use key names):

```typescript
// TypeScript: replace process.env.KEY
- const key = process.env.API_KEY;
+ import { getSecret } from '@keysync/node';
+ const key = await getSecret('API_KEY');
```

```go
// Go: replace os.Getenv("KEY")
- key := os.Getenv("API_KEY")
+ import "github.com/dipockdas/keysync/clients/go"
+ key, err := keysync.GetGlobal("API_KEY")
```

```python
# Python: replace os.environ.get("KEY")
- key = os.environ.get("API_KEY")
+ from keysync import get_secret
+ key = get_secret("API_KEY")
```

Also remove dotenv imports and add `.env*` to `.gitignore`.
