namespace KeySync;

/// <summary>
/// Abstraction over OS-native keychain backends.
///
/// Each platform (macOS, Linux, Windows) provides its own implementation.
/// </summary>
internal interface IKeychainProvider
{
    /// <summary>
    /// Retrieve a single secret from the keychain.
    /// </summary>
    /// <param name="service">Keychain service name (e.g. "keysync/global").</param>
    /// <param name="account">Account/key name (e.g. "DATABASE_URL").</param>
    /// <returns>The secret value.</returns>
    /// <exception cref="KeySyncException">Thrown on not-found or keychain errors.</exception>
    string GetSecret(string service, string account);

    /// <summary>
    /// List all keychain entries with a <c>keysync/*</c> service name.
    /// </summary>
    /// <returns>A list of (service, account) pairs.</returns>
    /// <exception cref="KeySyncException">Thrown on keychain errors.</exception>
    List<CredentialEntry> ListSecrets();
}
