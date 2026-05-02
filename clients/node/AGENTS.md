# keysync TypeScript / Node.js Client

## Overview

The TypeScript client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via child_process | Ready |
| Linux | `secret-tool` CLI via child_process | Ready |
| Windows | Not yet supported | Stub |

## API

```typescript
getSecret(key: string, project?: string): Promise<string>
listSecrets(scope?: string, project?: string): Promise<Array<{scope, project?, key}>>
```

## Key design decisions

- Runtime platform detection via `process.platform`
- Async API (all keychain calls are I/O-bound subprocess calls)
- Windows support is planned via a companion Go helper binary (the Go client
  can be compiled into a small wincred-helper executable)
- Service naming matches keysync convention: `keysync/global` and `keysync/project/<name>`
