package io.keysync;

/**
 * Exception thrown by KeySync operations.
 *
 * <p>Contains an error code ({@link KeySyncError}) and a human-readable message.
 *
 * <ul>
 *   <li>{@code NOT_FOUND} — the secret does not exist in any scope</li>
 *   <li>{@code KEYCHAIN_ERROR} — the native keychain tool failed</li>
 *   <li>{@code UNSUPPORTED_PLATFORM} — the current OS is not supported</li>
 * </ul>
 */
public class KeySyncException extends RuntimeException {

    private final KeySyncError error;

    /**
     * Creates a new KeySyncException.
     *
     * @param error   the error type
     * @param message a human-readable description
     */
    public KeySyncException(KeySyncError error, String message) {
        super(message);
        this.error = error;
    }

    /**
     * Creates a new KeySyncException with a cause.
     *
     * @param error   the error type
     * @param message a human-readable description
     * @param cause   the underlying cause
     */
    public KeySyncException(KeySyncError error, String message, Throwable cause) {
        super(message, cause);
        this.error = error;
    }

    /** Returns the error type. */
    public KeySyncError getError() {
        return error;
    }

    /** Returns the error code string (e.g. "notFound"). */
    public String getErrorCode() {
        return error.getCode();
    }
}
