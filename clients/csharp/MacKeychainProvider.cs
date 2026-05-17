using System.Diagnostics;
using System.Runtime.InteropServices;

namespace KeySync;

/// <summary>
/// macOS keychain access via the built-in <c>security</c> CLI.
///
/// Executes <c>security find-generic-password</c> to retrieve secrets
/// stored by keysync. No framework dependency required.
///
/// On newer macOS versions the password may be written to stderr instead
/// of stdout, so both streams are captured and checked.
/// </summary>
internal sealed class MacKeychainProvider : IKeychainProvider
{
    /// <inheritdoc />
    public string GetSecret(string service, string account)
    {
        var startInfo = new ProcessStartInfo
        {
            FileName = "/usr/bin/security",
            ArgumentList = { "find-generic-password", "-s", service, "-a", account, "-w" },
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true,
        };

        using var process = Process.Start(startInfo)
            ?? throw KeySyncException.KeychainError("failed to start security CLI");

        string stdout = process.StandardOutput.ReadToEnd().Trim();
        string stderr = process.StandardError.ReadToEnd().Trim();

        process.WaitForExit();

        // Treat non-zero exit codes with meaningful stderr as keychain errors.
        if (process.ExitCode != 0)
        {
            if (process.ExitCode == 44)
            {
                // Exit code 44 from `security` means "item not found".
                throw KeySyncException.NotFound();
            }

            throw KeySyncException.KeychainError(
                $"security exited with code {process.ExitCode}: {stderr}");
        }

        // Newer macOS sometimes writes the password to stderr.
        // Check stdout first, then fall back to stderr.
        if (!string.IsNullOrEmpty(stdout))
        {
            return stdout;
        }

        if (!string.IsNullOrEmpty(stderr))
        {
            return stderr;
        }

        throw KeySyncException.NotFound();
    }

    /// <inheritdoc />
    public List<CredentialEntry> ListSecrets()
    {
        var results = new List<CredentialEntry>();

        // Query global secrets.
        CollectEntries(results, "keysync/global");

        // Query project secrets by scanning for any service starting with "keysync/project/".
        // We do a broad dump-keychain and parse results.
        var startInfo = new ProcessStartInfo
        {
            FileName = "/usr/bin/security",
            Arguments = "dump-keychain",
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true,
        };

        using var process = Process.Start(startInfo)
            ?? throw KeySyncException.KeychainError("failed to start security dump-keychain");

        string stdout = process.StandardOutput.ReadToEnd();
        process.WaitForExit();

        if (process.ExitCode != 0)
        {
            // dump-keychain might fail on some configurations; return whatever we have.
            return results;
        }

        // Parse the output. Records are separated by "keychain: " lines.
        string[] records = stdout.Split("\nkeychain:", StringSplitOptions.None);
        foreach (string record in records)
        {
            if (!record.Contains("class: \"genp\""))
                continue;

            string? svc = ExtractAttribute(record, "svce");
            if (svc == null || !svc.StartsWith("keysync/", StringComparison.Ordinal))
                continue;

            string? acct = ExtractAttribute(record, "acct");
            if (acct == null)
                continue;

            results.Add(new CredentialEntry(svc, acct));
        }

        return results;
    }

    /// <summary>
    /// Finds all accounts stored under a specific service and adds them to the list.
    /// </summary>
    private static void CollectEntries(List<CredentialEntry> results, string service)
    {
        var startInfo = new ProcessStartInfo
        {
            FileName = "/usr/bin/security",
            Arguments = $"find-generic-password -s \"{service}\"",
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true,
        };

        using var process = Process.Start(startInfo);
        if (process == null)
            return;

        string stdout = process.StandardOutput.ReadToEnd();
        process.WaitForExit();

        if (process.ExitCode != 0)
            return;

        // Parse "acct" lines from the output.
        // Each line looks like:      "acct"<blob>="MY_KEY"
        // or in hex form:            "acct"<blob>=0x4D595F4B4559  "MY_KEY"
        foreach (string line in stdout.Split('\n'))
        {
            string trimmed = line.Trim();
            if (!trimmed.Contains("acct"))
                continue;

            int eqIndex = trimmed.LastIndexOf('=');
            if (eqIndex < 0)
                continue;

            // Value after the last '='
            string afterEq = trimmed[(eqIndex + 1)..].Trim();

            // Strip surrounding double quotes if present.
            string account = afterEq.Trim('"');
            if (account.Length > 0)
            {
                results.Add(new CredentialEntry(service, account));
            }
        }
    }

    /// <summary>
    /// Extracts a named attribute value from a dump-keychain record line.
    /// </summary>
    private static string? ExtractAttribute(string record, string attrName)
    {
        int idx = record.IndexOf($"\"{attrName}\"", StringComparison.Ordinal);
        if (idx < 0)
            return null;

        string after = record[(idx + attrName.Length + 2)..];
        int eqIdx = after.IndexOf('=');
        if (eqIdx < 0)
            return null;

        string val = after[(eqIdx + 1)..].Trim();
        if (val == "<NULL>")
            return null;

        if (val.StartsWith('"'))
        {
            int end = val.IndexOf('"', 1);
            if (end >= 0)
                return val[1..end];
        }

        return val.Trim('"');
    }
}
