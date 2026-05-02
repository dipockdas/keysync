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
