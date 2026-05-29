"""Error types for the keysync client."""


class KeySyncError(Exception):
    """Base error for keysync client operations."""

    def __init__(self, code: str, message: str):
        self.code = code
        super().__init__(message)


class SecretNotFoundError(KeySyncError):
    """The requested secret was not found in any scope."""

    def __init__(self, key: str):
        self.key = key
        super().__init__("notFound", f"secret not found: {key}")
