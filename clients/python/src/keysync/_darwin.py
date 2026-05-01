"""macOS keychain access via the built-in security CLI."""

import subprocess
from keysync._errors import SecretNotFoundError


def darwin_get(service: str, account: str) -> str:
    """Retrieve a secret from the macOS Keychain."""
    try:
        result = subprocess.run(
            ["security", "find-generic-password",
             "-s", service, "-a", account, "-w"],
            capture_output=True, text=True, check=True,
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        if e.returncode == 44:  # security: item not found
            raise SecretNotFoundError(f"{service}/{account}")
        raise


def darwin_list() -> list[dict]:
    """List all keysync secrets by parsing `security dump-keychain`."""
    try:
        result = subprocess.run(
            ["security", "dump-keychain"],
            capture_output=True, text=True, check=True,
        )
    except subprocess.CalledProcessError:
        return []

    records = result.stdout.split("\nkeychain:")
    entries = []
    for rec in records:
        if 'class: "genp"' not in rec:
            continue
        svc = _find_attr(rec, "svce")
        if not svc or not svc.startswith("keysync/"):
            continue
        acct = _find_attr(rec, "acct")
        if acct:
            entries.append({"service": svc, "account": acct})
    return entries


def _find_attr(record: str, attr_name: str) -> str | None:
    idx = record.find(f'"{attr_name}"')
    if idx < 0:
        return None
    after = record[idx + len(attr_name) + 2:]
    eq_idx = after.find("=")
    if eq_idx < 0:
        return None
    val = after[eq_idx + 1:].strip()
    if val == "<NULL>":
        return None
    if val.startswith('"'):
        end = val.find('"', 1)
        return val[1:end] if end >= 0 else val[1:]
    return val.strip('"')
