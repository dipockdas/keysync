# keysync TypeScript Client — Claude Instructions

## Build & Test

```bash
cd clients/node
npm install          # Install dependencies
npm run build        # Compile TypeScript → dist/
npm test             # Run vitest tests
```

## Key files

```
src/
  index.ts         # Public API (getSecret, listSecrets), platform selection
  types.ts         # Service name helpers
  darwin.ts        # macOS: child_process execFile → security CLI
  linux.ts         # Linux: child_process execFile → secret-tool CLI
  windows.ts       # Windows: throws unsupportedPlatform
src/__tests__/
  index.test.ts    # Error type tests
  types.test.ts    # Service name parsing tests
```

## Conventions

- `process.platform` for runtime platform detection
- `KeySyncError` class with `code` property (notFound, keychainError, unsupportedPlatform)
- Async API (Promise-based) for all operations
- Compiled with `tsc` to `dist/`, NodeNext module resolution
- vitest for testing
