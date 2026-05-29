# keysync Python Client — Claude Instructions

## Build & Test

```bash
cd clients/python
uv venv                        # Create virtual environment
uv pip install pytest -q        # Install test dependencies
PYTHONPATH=src uv run pytest    # Run tests
```

## Key files

```
src/keysync/
  __init__.py    # Public API (get_secret, list_secrets), platform selection
  _errors.py     # KeySyncError, SecretNotFoundError
  _darwin.py     # macOS: subprocess → security CLI
  _linux.py      # Linux: subprocess → secret-tool CLI
  _windows.py    # Windows: ctypes Win32 API (CredReadW, CredEnumerateW)
tests/
  test_client.py # Tests for service names and error types
```

## Conventions

- `sys.platform` for runtime platform detection
- Private modules prefixed with `_` (e.g. `_darwin.py`)
- Error types in a separate `_errors.py` module to avoid circular imports
- `SecretNotFoundError` for missing secrets, `KeySyncError` for platform/OS errors
- Python 3.11+ with type annotations
- uv for package management

## Migration: replacing os.environ with keysync

```python
from keysync import get_secret, list_secrets

# Global secret (shared across projects)
api_key = get_secret("API_KEY")

# Project-scoped secret (falls back to global if no project match)
db_url = get_secret("DATABASE_URL", project="myapp")

# List all secrets
globals = list_secrets()                    # global only
project = list_secrets(project="myapp")     # project + global fallback
```
