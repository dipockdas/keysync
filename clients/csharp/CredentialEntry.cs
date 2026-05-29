namespace KeySync;

/// <summary>
/// Represents a single secret entry discovered by <see cref="KeySyncClient.ListSecrets"/>.
/// </summary>
/// <param name="Service">
/// The keychain service name (e.g. <c>"keysync/global"</c> or
/// <c>"keysync/project/my-app"</c>).
/// </param>
/// <param name="Account">
/// The account/key name (e.g. <c>"DATABASE_URL"</c>).
/// </param>
/// <param name="Environment">
/// The environment name (e.g. <c>"staging"</c>), or null if not set.
/// </param>
public record CredentialEntry(string Service, string Account, string? Environment = null);
