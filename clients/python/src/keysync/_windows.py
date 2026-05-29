"""Windows keychain access via ctypes Win32 API (Credential Manager).

Uses the CredEnumerateW / CredReadW Win32 API directly via ctypes.
No external dependencies required.
"""

import ctypes
import ctypes.wintypes
from keysync._errors import SecretNotFoundError, KeySyncError


# Win32 Credential Manager constants
CRED_TYPE_GENERIC = 1
CRED_ENUMERATE_ALL_CREDENTIALS = 0


class CREDENTIALW(ctypes.Structure):
    _fields_ = [
        ("Flags", ctypes.wintypes.DWORD),
        ("Type", ctypes.wintypes.DWORD),
        ("TargetName", ctypes.wintypes.LPCWSTR),
        ("Comment", ctypes.wintypes.LPCWSTR),
        ("LastWritten", ctypes.wintypes.FILETIME),
        ("CredentialBlobSize", ctypes.wintypes.DWORD),
        ("CredentialBlob", ctypes.wintypes.LPBYTE),
        ("Persist", ctypes.wintypes.DWORD),
        ("AttributeCount", ctypes.wintypes.DWORD),
        ("Attributes", ctypes.c_void_p),
        ("TargetAlias", ctypes.wintypes.LPCWSTR),
        ("UserName", ctypes.wintypes.LPCWSTR),
    ]


PCREDENTIALW = ctypes.POINTER(CREDENTIALW)


def _load_win32():
    """Load Win32 Credential Manager functions."""
    advapi32 = ctypes.windll.advapi32

    advapi32.CredEnumerateW.argtypes = [
        ctypes.wintypes.LPCWSTR,
        ctypes.wintypes.DWORD,
        ctypes.POINTER(ctypes.wintypes.DWORD),
        ctypes.POINTER(PCREDENTIALW),
    ]
    advapi32.CredEnumerateW.restype = ctypes.wintypes.BOOL

    advapi32.CredReadW.argtypes = [
        ctypes.wintypes.LPCWSTR,
        ctypes.wintypes.DWORD,
        ctypes.wintypes.DWORD,
        ctypes.POINTER(PCREDENTIALW),
    ]
    advapi32.CredReadW.restype = ctypes.wintypes.BOOL

    advapi32.CredFree.argtypes = [ctypes.c_void_p]
    advapi32.CredFree.restype = None

    return advapi32


def _target_name(service: str) -> str:
    """Convert a keysync service name to a Win32 Credential Manager target.

    Regular:  "keysync/global"        → "keysync_global"
    Regular:  "keysync/project/X"     → "keysync_project_X"
    With env: "keysync/project/X/env/Y" → "keysync_project_X_Y"  (env keyword stripped)
    """
    if service.startswith("keysync/"):
        # Strip /env/ keyword then replace remaining / with _
        path = service[8:].replace("/env/", "/")
        return "keysync_" + path.replace("/", "_")
    return "keysync_" + service


def windows_get(service: str, account: str) -> str:
    """Retrieve a secret from Windows Credential Manager."""
    advapi32 = _load_win32()
    target = _target_name(service)
    pcred = PCREDENTIALW()

    if not advapi32.CredReadW(target, CRED_TYPE_GENERIC, 0, ctypes.byref(pcred)):
        raise SecretNotFoundError(f"{service}/{account}")

    try:
        cred = pcred.contents
        if cred.UserName != account:
            raise SecretNotFoundError(f"{service}/{account}")
        blob_size = cred.CredentialBlobSize
        blob_data = ctypes.string_at(cred.CredentialBlob, blob_size)
        return blob_data.decode("utf-16-le").rstrip("\x00")
    finally:
        advapi32.CredFree(pcred)


def windows_list() -> list[dict]:
    """List all keysync secrets from Credential Manager."""
    advapi32 = _load_win32()
    count = ctypes.wintypes.DWORD(0)
    pcred_array = PCREDENTIALW()

    if not advapi32.CredEnumerateW(
        "keysync*",
        CRED_ENUMERATE_ALL_CREDENTIALS,
        ctypes.byref(count),
        ctypes.byref(pcred_array),
    ):
        return []

    try:
        entries = []
        for i in range(count.value):
            cred = pcred_array[i]
            svc = cred.TargetName
            if not svc or not svc.startswith("keysync_"):
                continue
            # Reverse the _target_name conversion.
            # 4+ underscore parts: last part is environment.
            parts = svc.split("_")
            if len(parts) >= 4:
                # keysync_project_myapp_dev → keysync/project/myapp/env/dev
                name = "/".join(parts[:2]) + "/" + "/".join(parts[2:-1]) + "/env/" + parts[-1]
            elif len(parts) >= 3:
                # keysync_project_myapp → keysync/project/myapp
                name = "/".join(parts[:2]) + "/" + "/".join(parts[2:])
            else:
                # keysync_global → keysync/global
                name = "/".join(parts)
            entries.append({"service": name, "account": cred.UserName})
        return entries
    finally:
        advapi32.CredFree(pcred_array)
