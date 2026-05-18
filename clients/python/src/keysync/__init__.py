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

import os as _os
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


def _service_name(scope: str, project: str | None = None, environment: str | None = None) -> str:
    """Build a keychain service name from scope, project, and environment.
    Global:  "keysync/global"
    Project: "keysync/project/<name>"
    Project with env: "keysync/project/<name>/env/<env>"
    """
    if not project or scope == "global":
        return f"keysync/{scope}"
    if environment:
        return f"keysync/{scope}/{project}/env/{environment}"
    return f"keysync/{scope}/{project}"


def _parse_service_name(svc: str) -> tuple[str, str | None, str | None]:
    """Parse a service name into (scope, project, environment)."""
    if not svc.startswith("keysync/"):
        return ("global", None, None)
    trimmed = svc.removeprefix("keysync/")
    if "/" not in trimmed:
        return (trimmed or "global", None, None)
    scope, rest = trimmed.split("/", 1)
    if scope != "project":
        return (scope, None, None)
    # Check for /env/ segment
    if "/env/" in rest:
        project, env = rest.split("/env/", 1)
        return (scope, project, env)
    return (scope, rest, None)


def get_secret(key: str, project: str | None = None, environment: str | None = None) -> str:
    """Retrieve a secret from the OS keychain.

    Resolution order:
    1. Check environment variable first
    2. If *environment* is provided, try keysync/project/<project>/env/<env>
    3. Try keysync/project/<project> (project scope, no env)
    4. Fall back to keysync/global

    Raises SecretNotFoundError if the secret doesn't exist.
    Raises KeySyncError if the platform is unsupported.
    """
    # Primary path: check environment variable first.
    # In local dev the user runs eval $(keysync export) at shell startup;
    # in cloud/CI the platform injects env vars directly.
    val = _os.environ.get(key)
    if val is not None:
        return val

    # Try project + environment scope first
    if project and environment:
        svc = _service_name("project", project, environment)
        try:
            return _platform_get(svc, key)
        except SecretNotFoundError:
            pass  # fall through to project scope

    # Try project scope (no env)
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


def list_secrets(scope: str | None = None, project: str | None = None, environment: str | None = None) -> list[dict]:
    """List all stored secret entries.

    Returns a list of dicts with keys: scope, project, key.
    If an environment is present, 'environment' key is also included.

    Keyword arguments:
        scope -- filter by scope ("global" or "project")
        project -- filter by project name
        environment -- filter by environment name
    """
    entries = _platform_list()
    results = []
    for entry in entries:
        entry_scope, entry_project, entry_env = _parse_service_name(entry["service"])
        if scope and entry_scope != scope:
            continue
        if project and entry_project != project:
            continue
        if environment and entry_env != environment:
            continue
        item = {
            "scope": entry_scope,
            "project": entry_project,
            "key": entry["account"],
        }
        if entry_env:
            item["environment"] = entry_env
        results.append(item)
    return results
