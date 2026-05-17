using System.Runtime.InteropServices;

namespace KeySync;

/// <summary>
/// Keysync client — retrieve secrets from the OS keychain.
///
/// Every <see cref="GetSecret"/> call follows this resolution order:
/// <list type="number">
///   <item>Check environment variable first — for cloud/CI where the
///       platform injects env vars directly.</item>
///   <item>If not found, fall back to the OS keychain.</item>
///   <item>If <c>project</c> is provided, check project scope first,
///       then global scope.</item>
/// </list>
///
/// <para>Usage:</para>
/// <code>
/// using KeySync;
///
/// // Get a global secret
/// string apiKey = KeySyncClient.GetSecret("API_KEY");
///
/// // Get a project-scoped secret (falls back to global)
/// string dbUrl = KeySyncClient.GetSecret("DATABASE_URL", "myapp");
///
/// // List all global secrets
/// var globals = KeySyncClient.ListSecrets();
///
/// // List project secrets (includes global fallback)
/// var project = KeySyncClient.ListSecrets("myapp");
/// </code>
/// </summary>
public static class KeySyncClient
{
    private static readonly IKeychainProvider _provider = ResolveProvider();

    // ────────────────────────────────────────────────────
    //  Public API
    // ────────────────────────────────────────────────────

    /// <summary>
    /// Retrieve a secret from the OS keychain.
    ///
    /// Checks the environment variable identified by <paramref name="key"/>
    /// first. If set, returns it immediately without touching the OS keychain.
    /// This is the primary path for both local development (where secrets are
    /// injected via <c>eval $(keysync export)</c>) and cloud deployments
    /// (where platforms inject environment variables directly).
    ///
    /// If the env var is not set, falls back to the OS keychain. When
    /// <paramref name="project"/> is provided, checks project scope first,
    /// then global scope.
    /// </summary>
    /// <param name="key">
    /// The secret key name (e.g. <c>"DATABASE_URL"</c>).
    /// </param>
    /// <param name="project">
    /// Optional project name. If provided, checks project scope first,
    /// then falls back to global scope.
    /// </param>
    /// <returns>The secret value.</returns>
    /// <exception cref="KeySyncException">
    /// Thrown with <see cref="KeySyncError.NotFound"/> if the secret
    /// doesn't exist in any checked scope.</exception>
    public static string GetSecret(string key, string? project = null)
    {
        // 1. Check environment variable first (cloud/CI + local dev).
        string? envVal = Environment.GetEnvironmentVariable(key);
        if (envVal != null)
        {
            return envVal;
        }

        // 2. Try project scope first (if a project is specified).
        if (!string.IsNullOrEmpty(project))
        {
            string projectService = ServiceName.ForScope("project", project);
            try
            {
                return _provider.GetSecret(projectService, key);
            }
            catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.NotFound)
            {
                // Fall through to global scope.
            }
        }

        // 3. Fall back to global scope.
        string globalService = ServiceName.ForScope("global", project: null);
        return _provider.GetSecret(globalService, key);
    }

    /// <summary>
    /// List all stored secret entries.
    ///
    /// When <paramref name="project"/> is provided, returns both project-scoped
    /// and global-scoped keys that are relevant to that project.
    /// When omitted or null, returns all global-scoped keys only.
    /// </summary>
    /// <param name="project">
    /// Optional project name to filter by. If omitted, returns global entries.
    /// </param>
    /// <returns>A list of <see cref="CredentialEntry"/> objects.</returns>
    public static List<CredentialEntry> ListSecrets(string? project = null)
    {
        var allEntries = _provider.ListSecrets();

        if (string.IsNullOrEmpty(project))
        {
            // Return only global-scoped entries.
            return allEntries
                .Where(e => e.Service == "keysync/global")
                .ToList();
        }

        // Return project entries plus global entries.
        string projectService = ServiceName.ForScope("project", project);
        return allEntries
            .Where(e => e.Service == projectService || e.Service == "keysync/global")
            .ToList();
    }

    // ────────────────────────────────────────────────────
    //  Platform resolution
    // ────────────────────────────────────────────────────

    private static IKeychainProvider ResolveProvider()
    {
        if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
            return new MacKeychainProvider();

        if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
            return new LinuxKeychainProvider();

        if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
            return new WindowsKeychainProvider();

        throw KeySyncException.UnsupportedPlatform();
    }
}

// ────────────────────────────────────────────────────────
//  Service name helpers
// ────────────────────────────────────────────────────────

/// <summary>
/// Service name helpers matching the keysync keychain convention.
///
/// Secrets are stored with a service name that encodes scope and project:
/// <list type="bullet">
///   <item>Global:  <c>"keysync/global"</c></item>
///   <item>Project: <c>"keysync/project/&lt;name&gt;"</c></item>
/// </list>
/// </summary>
internal static class ServiceName
{
    /// <summary>
    /// Builds a service name from a scope and optional project.
    /// </summary>
    internal static string ForScope(string scope, string? project)
    {
        if (!string.IsNullOrEmpty(project) && scope == "project")
            return $"keysync/{scope}/{project}";

        return $"keysync/{scope}";
    }

    /// <summary>
    /// Parses a service name into its scope and project components.
    /// </summary>
    /// <param name="service">The full service name (e.g. "keysync/project/my-app").</param>
    /// <returns>A tuple of (scope, project). Project is null for global entries.</returns>
    internal static (string Scope, string? Project) Parse(string service)
    {
        ReadOnlySpan<char> span = service.AsSpan().Trim();
        const string prefix = "keysync/";

        if (!span.StartsWith(prefix))
            return ("global", null);

        ReadOnlySpan<char> remainder = span[prefix.Length..];
        int slashIdx = remainder.IndexOf('/');

        if (slashIdx < 0)
            return (remainder.ToString(), null);

        string scope = remainder[..slashIdx].ToString();
        if (scope != "project")
            return (scope, null);

        string project = remainder[(slashIdx + 1)..].ToString();
        return (scope, project);
    }
}
