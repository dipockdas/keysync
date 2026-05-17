using System.Runtime.InteropServices;

namespace KeySync;

/// <summary>
/// Windows Credential Manager access via Win32 P/Invoke to advapi32.dll.
///
/// Calls <c>CredReadW</c>, <c>CredEnumerateW</c>, and <c>CredFree</c> directly
/// with no external process or dependency.
///
/// Target name convention:
///   <c>"keysync/global"</c> is stored as <c>"keysync_global"</c>
///   <c>"keysync/project/my-app"</c> is stored as <c>"keysync_project_my-app"</c>
/// </summary>
internal sealed class WindowsKeychainProvider : IKeychainProvider
{
    // Win32 credential type: CRED_TYPE_GENERIC = 1
    private const uint CRED_TYPE_GENERIC = 1;

    // CredEnumerateW filter: return all credentials.
    private const string CRED_ENUM_ALL = "*";

    /// <inheritdoc />
    public string GetSecret(string service, string account)
    {
        string target = ServiceToTarget(service);

        if (!CredReadW(target, CRED_TYPE_GENERIC, 0, out IntPtr pCred))
        {
            int error = Marshal.GetLastWin32Error();
            if (error == 1168) // ERROR_NOT_FOUND
                throw KeySyncException.NotFound();

            throw KeySyncException.KeychainError(
                $"CredReadW failed for target \"{target}\" (error {error})");
        }

        try
        {
            var cred = Marshal.PtrToStructure<CREDENTIAL>(pCred)
                ?? throw KeySyncException.KeychainError("failed to marshal CREDENTIAL struct");

            string? userName = cred.UserName != IntPtr.Zero
                ? Marshal.PtrToStringUni(cred.UserName)
                : null;

            // Verify the account/key name matches.
            if (userName != account)
                throw KeySyncException.NotFound();

            string? password = cred.CredentialBlob != IntPtr.Zero && cred.CredentialBlobSize > 0
                ? Marshal.PtrToStringUni(cred.CredentialBlob, (int)cred.CredentialBlobSize / 2)
                : null;

            if (password == null)
                throw KeySyncException.NotFound();

            return password;
        }
        finally
        {
            CredFree(pCred);
        }
    }

    /// <inheritdoc />
    public List<CredentialEntry> ListSecrets()
    {
        var results = new List<CredentialEntry>();

        if (!CredEnumerateW(null, 0, out int count, out IntPtr pCredentials))
        {
            int error = Marshal.GetLastWin32Error();
            // ERROR_NOT_FOUND is expected when there are no credentials.
            if (error == 1168)
                return results;

            throw KeySyncException.KeychainError(
                $"CredEnumerateW failed (error {error})");
        }

        try
        {
            for (int i = 0; i < count; i++)
            {
                // pCredentials points to an array of PCREDENTIAL (pointers).
                // Each element is IntPtr.Size bytes. Read the i-th pointer,
                // then marshal the CREDENTIAL struct it points to.
                IntPtr pElemAddr = IntPtr.Add(pCredentials, i * IntPtr.Size);
                IntPtr pCred = Marshal.ReadIntPtr(pElemAddr);

                var cred = Marshal.PtrToStructure<CREDENTIAL>(pCred);
                string? targetName = cred.TargetName != IntPtr.Zero
                    ? Marshal.PtrToStringUni(cred.TargetName)
                    : null;

                if (targetName == null || !targetName.StartsWith("keysync_", StringComparison.Ordinal))
                    continue;

                string service = TargetToService(targetName);

                string? account = cred.UserName != IntPtr.Zero
                    ? Marshal.PtrToStringUni(cred.UserName)
                    : null;

                if (account == null)
                    continue;

                results.Add(new CredentialEntry(service, account));
            }

            return results;
        }
        finally
        {
            CredFree(pCredentials);
        }
    }

    // ────────────────────────────────────────────────────
    //  P/Invoke declarations
    // ────────────────────────────────────────────────────

    [DllImport("advapi32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool CredReadW(
        string targetName,
        uint type,
        uint flags,
        out IntPtr credential);

    [DllImport("advapi32.dll")]
    private static extern void CredFree(IntPtr buffer);

    [DllImport("advapi32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    private static extern bool CredEnumerateW(
        string? filter,
        uint flags,
        out int count,
        out IntPtr credentials);

    // ────────────────────────────────────────────────────
    //  CREDENTIAL struct (matches Win32 CREDENTIALW)
    // ────────────────────────────────────────────────────

    [StructLayout(LayoutKind.Sequential)]
    private struct CREDENTIAL
    {
        public uint Flags;
        public uint Type;
        public IntPtr TargetName;
        public IntPtr Comment;
        public long LastWritten;       // FILETIME packed as 64-bit
        public uint CredentialBlobSize;
        public IntPtr CredentialBlob;
        public uint Persist;
        public uint AttributeCount;
        public IntPtr Attributes;      // PCREDENTIAL_ATTRIBUTEW
        public IntPtr TargetAlias;
        public IntPtr UserName;
    }

    // ────────────────────────────────────────────────────
    //  Name conversions
    // ────────────────────────────────────────────────────

    /// <summary>
    /// Converts a keysync service name to a Windows Credential Manager
    /// target name by replacing slashes with underscores.
    /// </summary>
    /// <example>
    /// <c>"keysync/global"</c> → <c>"keysync_global"</c>
    /// <c>"keysync/project/my-app"</c> → <c>"keysync_project_my-app"</c>
    /// </example>
    internal static string ServiceToTarget(string service)
    {
        if (service.StartsWith("keysync/", StringComparison.Ordinal))
        {
            return "keysync_" + service[8..].Replace('/', '_');
        }
        return "keysync_" + service.Replace('/', '_');
    }

    /// <summary>
    /// Converts a Windows target name back to a keysync service name.
    /// </summary>
    /// <example>
    /// <c>"keysync_global"</c> → <c>"keysync/global"</c>
    /// <c>"keysync_project_my-app"</c> → <c>"keysync/project/my-app"</c>
    /// </example>
    internal static string TargetToService(string target)
    {
        if (target.StartsWith("keysync_", StringComparison.Ordinal))
        {
            return "keysync/" + target[8..].Replace('_', '/');
        }
        return target;
    }
}
