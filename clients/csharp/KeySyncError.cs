namespace KeySync;

/// <summary>
/// Error type codes used by <see cref="KeySyncException"/>.
/// </summary>
public enum KeySyncError
{
    /// <summary>The requested secret was not found in any scope.</summary>
    NotFound,

    /// <summary>An OS-level keychain error occurred.</summary>
    KeychainError,

    /// <summary>The current platform does not have keychain support.</summary>
    UnsupportedPlatform,
}
