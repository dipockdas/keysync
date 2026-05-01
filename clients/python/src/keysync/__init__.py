"""Keysync client — retrieve secrets from the OS keychain.

Each platform uses its native keychain tooling:
  - macOS:  security CLI (built-in)
  - Linux:  secret-tool CLI (libsecret)
  - Windows: ctypes Win32 API (Credential Manager)

Usage:
    from keysync import get_secret, list_secrets

    db_url = get_secret("DATABASE_URL", project="my-api")
    api_key = get_secret("GLOBAL_API_KEY")
"""

import sys as _sys

from keysync._errors import KeySyncError, SecretNotFoundError
from keysync._darwin import darwin_get, darwin_list
from keysync._linux import linux_get, linux_list
from keysync._windows import windows_get, windows_list


# Platform selection
if _sys.platform == "darwin":
    _platform_get = darwin_get
    _platform_list = darwin_list
elif _sys.platform == "linux":
    _platform_get = linux_get
    _platform_list = linux_list
elif _sys.platform == "win32":
    _platform_get = windows_get
    _platform_list = windows_list
else:
    def _platform_get(service, account):
        raise KeySyncError("unsupportedPlatform", f"unsupported platform: {_sys.platform}")
    _platform_list = lambda scope, project: []


def _service_name(scope: str, project: str | None = None) -> str:
    """Build a keychain service name from scope and project.
    Global:  "keysync/global"
    Project: "keysync/project/<name>"
    """
    if not project or scope == "global":
        return f"keysync/{scope}"
    return f"keysync/{scope}/{project}"


def _parse_service_name(svc: str) -> tuple[str, str | None]:
    """Parse a service name into (scope, project)."""
    if not svc.startswith("keysync/"):
        return ("global", None)
    trimmed = svc.removeprefix("keysync/")
    if "/" not in trimmed:
        return (trimmed or "global", None)
    scope, rest = trimmed.split("/", 1)
    if scope != "project":
        return (scope, None)
    return (scope, rest)


def get_secret(key: str, project: str | None = None) -> str:
    """Retrieve a secret from the OS keychain.

    If *project* is provided, checks project scope first, then falls back
    to global scope. If *project* is None, only global scope is checked.

    Raises SecretNotFoundError if the secret doesn't exist.
    """
    # Try project scope first
    if project:
        svc = _service_name("project", project)
        try:
            return _platform_get(svc, key)
        except SecretNotFoundError:
            pass  # fall through to global

    # Fall back to global scope
    svc = _service_name("global")
    try:
        return _platform_get(svc, key)
    except SecretNotFoundError:
        raise SecretNotFoundError(key)


def list_secrets(scope: str | None = None, project: str | None = None) -> list[dict]:
    """List all stored secret entries.

    Returns a list of dicts with keys: scope, project, key.

    Keyword arguments:
        scope -- filter by scope ("global" or "project")
        project -- filter by project name
    """
    entries = _platform_list()
    results = []
    for entry in entries:
        entry_scope, entry_project = _parse_service_name(entry["service"])
        if scope and entry_scope != scope:
            continue
        if project and entry_project != project:
            continue
        results.append({
            "scope": entry_scope,
            "project": entry_project,
            "key": entry["account"],
        })
    return results
