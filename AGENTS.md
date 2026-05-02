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

## Writing client libraries

When adding a new client library:

1. Use the platform's native keychain tooling — never depend on the `keysync` binary
2. Implement `getSecret` with project→global fallback
3. Implement `listSecrets` with scope/project filtering
4. Include a `MemoryStore` or equivalent for testing
5. Create README.md, CLAUDE.md, and AGENTS.md
6. Add to the directory listing in `clients/README.md`
