package io.keysync;

/**
 * Error codes for KeySync exceptions.
 */
public enum KeySyncError {
    /** The requested secret was not found in any scope. */
    NOT_FOUND("notFound"),

    /** An OS-level keychain error occurred. */
    KEYCHAIN_ERROR("keychainError"),

    /** The current platform is not supported. */
    UNSUPPORTED_PLATFORM("unsupportedPlatform");

    private final String code;

    KeySyncError(String code) {
        this.code = code;
    }

    /** Returns the error code string (e.g. "notFound"). */
    public String getCode() {
        return code;
    }

    @Override
    public String toString() {
        return code;
    }
}
