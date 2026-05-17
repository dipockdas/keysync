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
  windows.ts       # Windows: PowerShell with inline C# (CredReadW/CredEnumerateW)
src/__tests__/
  index.test.ts    # Error type tests
  types.test.ts    # Service name parsing tests
  windows.test.ts  # Windows target conversion tests
```

## Conventions

- `process.platform` for runtime platform detection
- `KeySyncError` class with `code` property (notFound, keychainError, unsupportedPlatform)
- Async API (Promise-based) for all operations
- Compiled with `tsc` to `dist/`, NodeNext module resolution
- vitest for testing
- Windows: PowerShell with inline C# compiles at runtime to P/Invoke CredReadW / CredEnumerateW
  from advapi32.dll — no native compilation or helper binary needed
- Windows target names: slashes replaced with underscores (keysync/global → keysync_global)

## Migration: replacing process.env with keysync

```typescript
import { getSecret, listSecrets } from '@keysync/node';

// Global secret (shared across projects)
const apiKey = await getSecret('API_KEY');

// Project-scoped secret (falls back to global if no project match)
const dbUrl = await getSecret('DATABASE_URL', 'myapp');

// List all secrets
const globals = await listSecrets();
const project = await listSecrets('myapp');
```
