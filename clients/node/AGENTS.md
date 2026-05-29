# keysync TypeScript / Node.js Client

## Overview

The TypeScript client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via child_process | Ready |
| Linux | `secret-tool` CLI via child_process | Ready |
| Windows | PowerShell with inline C# (CredReadW / CredEnumerateW) | Available (not fully tested on Windows) |

## API

```typescript
getSecret(key: string, project?: string): Promise<string>
listSecrets(scope?: string, project?: string): Promise<Array<{scope, project?, key}>>
```

## Key design decisions

- Runtime platform detection via `process.platform`
- Async API (all keychain calls are I/O-bound subprocess calls)
- Windows uses PowerShell with inline C# to P/Invoke CredReadW / CredEnumerateW
  from advapi32.dll — no native compilation or helper binary required.
- Service naming matches keysync convention: `keysync/global` and `keysync/project/<name>`
