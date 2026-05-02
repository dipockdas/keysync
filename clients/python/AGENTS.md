# keysync Python Client

## Overview

The Python client retrieves secrets managed by keysync from the OS keychain
with no dependency on the keysync binary.

## Platform support

| Platform | Mechanism | Status |
|----------|-----------|--------|
| macOS | `security` CLI via subprocess | Ready |
| Linux | `secret-tool` CLI via subprocess | Ready |
| Windows | ctypes Win32 API (CredReadW / CredEnumerateW) | Ready |

## API

```python
get_secret(key: str, project: str | None = None) -> str
list_secrets(scope: str | None = None, project: str | None = None) -> list[dict]
```

## Key design decisions

- Runtime platform detection via `sys.platform`
- Windows uses **ctypes Win32 API directly** — no external dependencies, no CGo,
  no helper binaries. This is the cleanest Windows story of any client library.
- Error classes in `_errors.py` to avoid circular imports between `__init__.py`
  and the platform modules.
- Service naming matches keysync convention: `keysync/global` and `keysync/project/<name>`.
