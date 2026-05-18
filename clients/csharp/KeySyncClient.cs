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
///   <item>If environment is provided, check environment scope first
///       (<c>keysync/project/&lt;name&gt;/env/&lt;env&gt;</c>).</item>
///   <item>If <c>project</c> is provided, check project scope next
///       (<c>keysync/project/&lt;name&gt;</c>).</item>
///   <item>Fall back to global scope (<c>keysync/global</c>).</item>
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
/// // Get an environment-scoped secret
/// string stagingDb = KeySyncClient.GetSecret("DATABASE_URL", "myapp", "staging");
///
/// // List all secrets
/// var globals = KeySyncClient.ListSecrets();
///
/// // List project secrets (includes global fallback)
/// var project = KeySyncClient.ListSecrets("myapp");
///
/// // List environment-scoped secrets
/// var envSecrets = KeySyncClient.ListSecrets("myapp", "staging");
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
    /// then global scope. When <paramref name="environment"/> is also
    /// provided, checks environment scope before project scope.
    /// </summary>
    /// <param name="key">
    /// The secret key name (e.g. <c>"DATABASE_URL"</c>).
    /// </param>
    /// <param name="project">
    /// Optional project name. If provided, checks project scope first,
    /// then falls back to global scope.
    /// </param>
    /// <param name="environment">
    /// Optional environment name (e.g. <c>"staging"</c>, <c>"production"</c>).
    /// If both project and environment are provided, environment-scoped
    /// secrets are checked before project-scoped secrets.
    /// </param>
    /// <returns>The secret value.</returns>
    /// <exception cref="KeySyncException">
    /// Thrown with <see cref="KeySyncError.NotFound"/> if the secret
    /// doesn't exist in any checked scope.</exception>
    public static string GetSecret(string key, string? project = null, string? environment = null)
    {
        // 1. Check environment variable first (cloud/CI + local dev).
        string? envVal = Environment.GetEnvironmentVariable(key);
        if (envVal != null)
        {
            return envVal;
        }

        // 2. If project is provided, check environment scope first, then project scope.
        if (!string.IsNullOrEmpty(project))
        {
            // 2a. Try environment-scoped (if environment is provided)
            if (!string.IsNullOrEmpty(environment))
            {
                string envService = ServiceName.ForScope("project", project, environment);
                try
                {
                    return _provider.GetSecret(envService, key);
                }
                catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.NotFound)
                {
                    // Fall through to project scope.
                }
            }

            // 2b. Try project scope
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
    /// When <paramref name="environment"/> is also provided, environment-scoped
    /// entries are additionally included.
    /// When omitted or null, returns all global-scoped keys only.
    /// </summary>
    /// <param name="project">
    /// Optional project name to filter by. If omitted, returns global entries.
    /// </param>
    /// <param name="environment">
    /// Optional environment name to filter by.
    /// </param>
    /// <returns>A list of <see cref="CredentialEntry"/> objects.</returns>
    public static List<CredentialEntry> ListSecrets(string? project = null, string? environment = null)
    {
        var allEntries = _provider.ListSecrets();

        if (string.IsNullOrEmpty(project))
        {
            // Return only global-scoped entries.
            return allEntries
                .Where(e => e.Service == "keysync/global")
                .Select(e => new CredentialEntry(e.Service, e.Account, null))
                .ToList();
        }

        // Build matching services
        string projectService = ServiceName.ForScope("project", project);
        string? envService = !string.IsNullOrEmpty(environment)
            ? ServiceName.ForScope("project", project, environment)
            : null;

        return allEntries
            .Where(e => e.Service == projectService ||
                        e.Service == "keysync/global" ||
                        (envService != null && e.Service == envService))
            .Select(e =>
            {
                var (scope, _, env) = ServiceName.Parse(e.Service);
                return new CredentialEntry(e.Service, e.Account, env);
            })
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
/// Secrets are stored with a service name that encodes scope, project, and environment:
/// <list type="bullet">
///   <item>Global:       <c>"keysync/global"</c></item>
///   <item>Project:      <c>"keysync/project/&lt;name&gt;"</c></item>
///   <item>Environment:  <c>"keysync/project/&lt;name&gt;/env/&lt;env&gt;"</c></item>
/// </list>
/// </summary>
internal static class ServiceName
{
    /// <summary>
    /// Builds a service name from a scope, optional project, and optional environment.
    /// </summary>
    internal static string ForScope(string scope, string? project, string? environment = null)
    {
        if (!string.IsNullOrEmpty(project) && scope == "project")
        {
            if (!string.IsNullOrEmpty(environment))
                return $"keysync/{scope}/{project}/env/{environment}";

            return $"keysync/{scope}/{project}";
        }

        return $"keysync/{scope}";
    }

    /// <summary>
    /// Parses a service name into its scope, project, and environment components.
    /// </summary>
    /// <param name="service">The full service name (e.g. "keysync/project/my-app/env/staging").</param>
    /// <returns>A tuple of (scope, project, environment). Environment is null when not present.</returns>
    internal static (string Scope, string? Project, string? Environment) Parse(string service)
    {
        ReadOnlySpan<char> span = service.AsSpan().Trim();
        const string prefix = "keysync/";

        if (!span.StartsWith(prefix))
            return ("global", null, null);

        ReadOnlySpan<char> remainder = span[prefix.Length..];
        int slashIdx = remainder.IndexOf('/');

        if (slashIdx < 0)
            return (remainder.ToString(), null, null);

        string scope = remainder[..slashIdx].ToString();
        string rest = remainder[(slashIdx + 1)..].ToString();

        if (scope != "project")
            return (scope, null, null);

        // Check for /env/ segment to detect environment
        int envIdx = rest.IndexOf("/env/", StringComparison.Ordinal);
        if (envIdx > 0)
        {
            string project = rest[..envIdx];
            string environment = rest[(envIdx + "/env/".Length)..];
            return (scope, project, environment);
        }

        return (scope, rest, null);
    }
}
