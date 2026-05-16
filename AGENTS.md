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

**Step 1: Understand what was migrated**
The `keysync migrate` output lists every migrated key with its scope. For example:
```
API_KEY → stored as global
DB_URL → stored as project/myapp
```

**Step 2: Find all references in source code**
Search for `process.env.KEY`, `os.Getenv("KEY")`, `os.environ["KEY"]`, `ENV["KEY"]`.

**Step 3: Replace with the appropriate client library call**

| Language | Before | After |
|----------|--------|-------|
| TypeScript | `process.env.KEY` | `await getSecret("KEY")` from `@keysync/node` |
| Go | `os.Getenv("KEY")` | `keysync.GetGlobal("KEY")` or `keysync.GetSecret("project", "KEY")` |
| Python | `os.environ.get("KEY")` | `get_secret("KEY")` from `keysync` |

**Step 4: Remove dotenv imports**
Remove `import 'dotenv/config'`, `require('dotenv').config()`, `load_dotenv()` etc.

**Step 5: Clean up**
- Add `.env*` to `.gitignore`
- Delete or archive the `.env` file

Never read or expose the actual secret values — only the key names are needed.

## Writing client libraries

When adding a new client library:

1. Use the platform's native keychain tooling — never depend on the `keysync` binary
2. Implement `getSecret` with project→global fallback
3. Implement `listSecrets` with scope/project filtering
4. Include a `MemoryStore` or equivalent for testing
5. Create README.md, CLAUDE.md, and AGENTS.md
6. Add to the directory listing in `clients/README.md`
