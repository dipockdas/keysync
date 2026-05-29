namespace KeySync;

/// <summary>
/// Exception thrown by the KeySync client library.
/// </summary>
public class KeySyncException : Exception
{
    /// <summary>
    /// The error code identifying the type of failure.
    /// </summary>
    public KeySyncError ErrorCode { get; }

    internal KeySyncException(KeySyncError errorCode, string message)
        : base(message)
    {
        ErrorCode = errorCode;
    }

    internal KeySyncException(KeySyncError errorCode, string message, Exception innerException)
        : base(message, innerException)
    {
        ErrorCode = errorCode;
    }

    /// <summary>
    /// Creates a <see cref="KeySyncException"/> for "secret not found".
    /// </summary>
    internal static KeySyncException NotFound()
        => new(KeySyncError.NotFound, "secret not found");

    /// <summary>
    /// Creates a <see cref="KeySyncException"/> for a keychain-level error.
    /// </summary>
    internal static KeySyncException KeychainError(string detail)
        => new(KeySyncError.KeychainError, $"keychain error: {detail}");

    /// <summary>
    /// Creates a <see cref="KeySyncException"/> for a keychain-level error
    /// wrapping an inner exception.
    /// </summary>
    internal static KeySyncException KeychainError(string detail, Exception innerException)
        => new(KeySyncError.KeychainError, $"keychain error: {detail}", innerException);

    /// <summary>
    /// Creates a <see cref="KeySyncException"/> for unsupported platforms.
    /// </summary>
    internal static KeySyncException UnsupportedPlatform()
        => new(KeySyncError.UnsupportedPlatform, "keychain access not available on this platform");
}
