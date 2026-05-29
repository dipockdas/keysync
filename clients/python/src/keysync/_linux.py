"""Linux keychain access via the secret-tool CLI (libsecret)."""

import subprocess
from keysync._errors import SecretNotFoundError


def linux_get(service: str, account: str) -> str:
    """Retrieve a secret from libsecret."""
    try:
        result = subprocess.run(
            ["secret-tool", "lookup", "service", service, "account", account],
            capture_output=True, text=True, check=True,
        )
        val = result.stdout.strip()
        if not val:
            raise SecretNotFoundError(f"{service}/{account}")
        return val
    except subprocess.CalledProcessError:
        raise SecretNotFoundError(f"{service}/{account}")


def linux_list() -> list[dict]:
    """List all keysync secrets by searching libsecret."""
    try:
        result = subprocess.run(
            ["secret-tool", "search", "service", "keysync"],
            capture_output=True, text=True, check=True,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        return []

    entries = []
    current_svc = ""
    current_acct = ""

    for line in result.stdout.split("\n"):
        trimmed = line.strip()
        if not trimmed:
            if current_svc and current_acct and current_svc.startswith("keysync/"):
                entries.append({"service": current_svc, "account": current_acct})
            current_svc = ""
            current_acct = ""
            continue
        if trimmed.startswith("service"):
            current_svc = _parse_attr(trimmed)
        elif trimmed.startswith("account"):
            current_acct = _parse_attr(trimmed)

    # Handle last entry if no trailing blank line
    if current_svc and current_acct and current_svc.startswith("keysync/"):
        entries.append({"service": current_svc, "account": current_acct})

    return entries


def _parse_attr(line: str) -> str:
    eq_idx = line.find("=")
    return line[eq_idx + 1:].strip() if eq_idx >= 0 else ""
