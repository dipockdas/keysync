using System.Diagnostics;

namespace KeySync;

/// <summary>
/// Linux keychain access via libsecret's <c>secret-tool</c> CLI.
///
/// Requires <c>libsecret-tools</c> to be installed and a running secret
/// service (GNOME Keyring, KDE Wallet, KeePassXC, etc.).
///
/// Uses the <c>service</c> and <c>account</c> attributes matching the
/// convention used by the keysync CLI and all other client libraries.
/// </summary>
internal sealed class LinuxKeychainProvider : IKeychainProvider
{
    /// <inheritdoc />
    public string GetSecret(string service, string account)
    {
        var startInfo = new ProcessStartInfo
        {
            FileName = "secret-tool",
            ArgumentList = { "lookup", "service", service, "account", account },
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true,
        };

        using var process = Process.Start(startInfo)
            ?? throw KeySyncException.KeychainError("failed to start secret-tool");

        string stdout = process.StandardOutput.ReadToEnd().Trim();
        string stderr = process.StandardError.ReadToEnd().Trim();

        process.WaitForExit();

        if (process.ExitCode != 0)
        {
            // secret-tool exits with code 1 when the secret is not found.
            if (process.ExitCode == 1)
                throw KeySyncException.NotFound();

            throw KeySyncException.KeychainError(
                $"secret-tool exited with code {process.ExitCode}: {stderr}");
        }

        if (string.IsNullOrEmpty(stdout))
            throw KeySyncException.NotFound();

        return stdout;
    }

    /// <inheritdoc />
    public List<CredentialEntry> ListSecrets()
    {
        var startInfo = new ProcessStartInfo
        {
            FileName = "secret-tool",
            ArgumentList = { "search", "service", "keysync" },
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true,
        };

        using var process = Process.Start(startInfo)
            ?? throw KeySyncException.KeychainError("failed to start secret-tool");

        string stdout = process.StandardOutput.ReadToEnd();
        process.WaitForExit();

        if (process.ExitCode != 0)
            return new List<CredentialEntry>(); // no results or secret-tool unavailable

        // Parse key-value output. Each entry is separated by a blank line.
        // Lines look like:
        //   service = keysync/global
        //   account = MY_KEY
        //   password = <value>
        var results = new List<CredentialEntry>();
        string? currentService = null;
        string? currentAccount = null;

        foreach (string rawLine in stdout.Split('\n'))
        {
            string line = rawLine.Trim();
            if (line.Length == 0)
            {
                // Blank line signals end of an entry.
                if (currentService != null && currentAccount != null)
                {
                    results.Add(new CredentialEntry(currentService, currentAccount));
                }
                currentService = null;
                currentAccount = null;
                continue;
            }

            if (line.StartsWith("service", StringComparison.OrdinalIgnoreCase))
            {
                currentService = ExtractValue(line);
            }
            else if (line.StartsWith("account", StringComparison.OrdinalIgnoreCase))
            {
                currentAccount = ExtractValue(line);
            }
        }

        // Handle the last entry if the output had no trailing blank line.
        if (currentService != null && currentAccount != null)
        {
            results.Add(new CredentialEntry(currentService, currentAccount));
        }

        return results;
    }

    /// <summary>
    /// Extracts the value portion of a <c>key = value</c> line.
    /// </summary>
    private static string? ExtractValue(string line)
    {
        int eqIdx = line.IndexOf('=');
        if (eqIdx < 0)
            return null;

        string val = line[(eqIdx + 1)..].Trim();
        return val.Length > 0 ? val : null;
    }
}
