using KeySync;
using Xunit;

namespace KeySync.Tests;

public class ServiceNameTests
{
    [Fact]
    public void ForScope_Global_ReturnsCorrectName()
    {
        string name = ServiceName.ForScope("global", project: null);
        Assert.Equal("keysync/global", name);
    }

    [Fact]
    public void ForScope_Project_ReturnsCorrectName()
    {
        string name = ServiceName.ForScope("project", project: "my-app");
        Assert.Equal("keysync/project/my-app", name);
    }

    [Fact]
    public void ForScope_GlobalWithProject_IgnoresProject()
    {
        string name = ServiceName.ForScope("global", project: "my-app");
        Assert.Equal("keysync/global", name);
    }

    [Fact]
    public void ForScope_ProjectWithoutName_ReturnsWithoutProject()
    {
        string name = ServiceName.ForScope("project", project: null);
        Assert.Equal("keysync/project", name);
    }

    [Fact]
    public void ForScope_ProjectWithEmptyName_ReturnsWithoutProject()
    {
        string name = ServiceName.ForScope("project", project: "");
        Assert.Equal("keysync/project", name);
    }

    [Fact]
    public void Parse_Global_ReturnsScopeOnly()
    {
        var (scope, project) = ServiceName.Parse("keysync/global");
        Assert.Equal("global", scope);
        Assert.Null(project);
    }

    [Fact]
    public void Parse_Project_ReturnsScopeAndProject()
    {
        var (scope, project) = ServiceName.Parse("keysync/project/my-app");
        Assert.Equal("project", scope);
        Assert.Equal("my-app", project);
    }

    [Fact]
    public void Parse_ProjectDeepNesting_ReturnsScopeAndProject()
    {
        var (scope, project) = ServiceName.Parse("keysync/project/my/deep/app");
        Assert.Equal("project", scope);
        Assert.Equal("my/deep/app", project);
    }

    [Fact]
    public void Parse_JustKeysync_ReturnsGlobal()
    {
        var (scope, project) = ServiceName.Parse("keysync");
        Assert.Equal("global", scope);
        Assert.Null(project);
    }

    [Fact]
    public void Parse_UnrecognizedScope_ReturnsScopeWithoutProject()
    {
        var (scope, project) = ServiceName.Parse("keysync/other/val");
        Assert.Equal("other", scope);
        Assert.Null(project);
    }

    [Fact]
    public void Parse_EmptyString_ReturnsGlobal()
    {
        var (scope, project) = ServiceName.Parse("");
        Assert.Equal("global", scope);
        Assert.Null(project);
    }

    [Fact]
    public void Parse_Whitespace_ReturnsGlobal()
    {
        var (scope, project) = ServiceName.Parse("   ");
        Assert.Equal("global", scope);
        Assert.Null(project);
    }

    [Fact]
    public void RoundTrip_Global()
    {
        string built = ServiceName.ForScope("global", null);
        var (scope, project) = ServiceName.Parse(built);
        Assert.Equal("global", scope);
        Assert.Null(project);
    }

    [Fact]
    public void RoundTrip_Project()
    {
        string built = ServiceName.ForScope("project", "my-app");
        var (scope, project) = ServiceName.Parse(built);
        Assert.Equal("project", scope);
        Assert.Equal("my-app", project);
    }
}

public class KeySyncErrorTests
{
    [Fact]
    public void NotFoundException_HasCorrectProperties()
    {
        var ex = KeySyncException.NotFound();
        Assert.Equal(KeySyncError.NotFound, ex.ErrorCode);
        Assert.Equal("secret not found", ex.Message);
    }

    [Fact]
    public void KeychainErrorException_HasCorrectProperties()
    {
        var ex = KeySyncException.KeychainError("something broke");
        Assert.Equal(KeySyncError.KeychainError, ex.ErrorCode);
        Assert.Contains("something broke", ex.Message);
    }

    [Fact]
    public void UnsupportedPlatformException_HasCorrectProperties()
    {
        var ex = KeySyncException.UnsupportedPlatform();
        Assert.Equal(KeySyncError.UnsupportedPlatform, ex.ErrorCode);
        Assert.Contains("keychain access not available", ex.Message);
    }

    [Fact]
    public void KeychainErrorException_WithInnerException()
    {
        var inner = new InvalidOperationException("bad state");
        var ex = new KeySyncException(
            KeySyncError.KeychainError,
            "something broke",
            inner);
        Assert.Equal(KeySyncError.KeychainError, ex.ErrorCode);
        Assert.Contains("something broke", ex.Message);
        Assert.Same(inner, ex.InnerException);
    }

    [Fact]
    public void ErrorCodeEnum_ValuesAreDistinct()
    {
        Assert.NotEqual((int)KeySyncError.NotFound, (int)KeySyncError.KeychainError);
        Assert.NotEqual((int)KeySyncError.NotFound, (int)KeySyncError.UnsupportedPlatform);
        Assert.NotEqual((int)KeySyncError.KeychainError, (int)KeySyncError.UnsupportedPlatform);
    }

    [Fact]
    public void CanCatchByErrorCode()
    {
        bool caughtNotFound = false;
        try
        {
            throw KeySyncException.NotFound();
        }
        catch (KeySyncException ex) when (ex.ErrorCode == KeySyncError.NotFound)
        {
            caughtNotFound = true;
        }
        Assert.True(caughtNotFound);
    }
}

public class EnvVarFallbackTests
{
    [Fact]
    public void GetSecret_ReturnsEnvVar_WhenSet()
    {
        string key = "KEY_SYNC_TEST_VAR_1";
        string expected = "env-var-value";
        try
        {
            Environment.SetEnvironmentVariable(key, expected);
            string result = KeySyncClient.GetSecret(key);
            Assert.Equal(expected, result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(key, null);
        }
    }

    [Fact]
    public void GetSecret_EnvVarTakesPriority_EvenWithProject()
    {
        string key = "KEY_SYNC_TEST_VAR_2";
        string expected = "priority-env-value";
        try
        {
            Environment.SetEnvironmentVariable(key, expected);
            string result = KeySyncClient.GetSecret(key, project: "some-project");
            Assert.Equal(expected, result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(key, null);
        }
    }

    [Fact]
    public void GetSecret_ReturnsEnvVar_WhenKeychainWouldFail()
    {
        // When the env var is set, it should be returned regardless
        // of whether the keychain entry exists.
        string key = "KEY_SYNC_TEST_VAR_3";
        string expected = "only-in-env";
        try
        {
            Environment.SetEnvironmentVariable(key, expected);
            string result = KeySyncClient.GetSecret(key);
            Assert.Equal(expected, result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(key, null);
        }
    }

    [Fact]
    public void GetSecret_ReturnsNullEnvVar()
    {
        string key = "KEY_SYNC_TEST_VAR_EMPTY";
        try
        {
            Environment.SetEnvironmentVariable(key, "");
            string result = KeySyncClient.GetSecret(key);
            Assert.Equal("", result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(key, null);
        }
    }
}

public class WindowsTargetConversionTests
{
    [Fact]
    public void ServiceToTarget_Global()
    {
        string result = WindowsKeychainProvider.ServiceToTarget("keysync/global");
        Assert.Equal("keysync_global", result);
    }

    [Fact]
    public void ServiceToTarget_Project()
    {
        string result = WindowsKeychainProvider.ServiceToTarget("keysync/project/my-app");
        Assert.Equal("keysync_project_my-app", result);
    }

    [Fact]
    public void ServiceToTarget_DeeplyNested()
    {
        string result = WindowsKeychainProvider.ServiceToTarget("keysync/project/my/deep/app");
        Assert.Equal("keysync_project_my_deep_app", result);
    }

    [Fact]
    public void TargetToService_Global()
    {
        string result = WindowsKeychainProvider.TargetToService("keysync_global");
        Assert.Equal("keysync/global", result);
    }

    [Fact]
    public void TargetToService_Project()
    {
        string result = WindowsKeychainProvider.TargetToService("keysync_project_my-app");
        Assert.Equal("keysync/project/my-app", result);
    }

    [Fact]
    public void ServiceToTarget_RoundTrip()
    {
        string original = "keysync/project/my-app";
        string target = WindowsKeychainProvider.ServiceToTarget(original);
        string back = WindowsKeychainProvider.TargetToService(target);
        Assert.Equal(original, back);
    }

    [Fact]
    public void ServiceToTarget_WithoutKeysyncPrefix()
    {
        // Service names that don't start with "keysync/" still get prefixed.
        string result = WindowsKeychainProvider.ServiceToTarget("some/service");
        Assert.Equal("keysync_some_service", result);
    }

    [Fact]
    public void TargetToService_WithoutKeysyncPrefix()
    {
        // Target names that don't start with "keysync_" are returned as-is.
        string result = WindowsKeychainProvider.TargetToService("some_service");
        Assert.Equal("some_service", result);
    }
}

public class CredentialEntryTests
{
    [Fact]
    public void Records_AreValueEqual()
    {
        var a = new CredentialEntry("keysync/global", "MY_KEY");
        var b = new CredentialEntry("keysync/global", "MY_KEY");
        Assert.Equal(a, b);
        Assert.True(a == b);
    }

    [Fact]
    public void Records_AreNotEqual_WhenDifferent()
    {
        var a = new CredentialEntry("keysync/global", "MY_KEY");
        var b = new CredentialEntry("keysync/global", "OTHER_KEY");
        Assert.NotEqual(a, b);
    }

    [Fact]
    public void Records_CanBeUsedInHashSets()
    {
        var set = new HashSet<CredentialEntry>
        {
            new CredentialEntry("keysync/global", "KEY_A"),
            new CredentialEntry("keysync/global", "KEY_B"),
            new CredentialEntry("keysync/global", "KEY_A"), // duplicate
        };
        Assert.Equal(2, set.Count);
    }
}

public class CREDENTIALStructTests
{
    [Fact]
    public void CredentialStructLayout_MatchesExpectedSize()
    {
        // On 64-bit Windows the CREDENTIAL struct should be 80 bytes:
        //   Flags(4) + Type(4) + TargetName(8) + Comment(8) +
        //   LastWritten(8) + CredentialBlobSize(4) + 4 padding +
        //   CredentialBlob(8) + Persist(4) + AttributeCount(4) +
        //   Attributes(8) + TargetAlias(8) + UserName(8) = 80
        int size = System.Runtime.InteropServices.Marshal.SizeOf<CREDENTIAL>();
        Assert.True(size == 80, $"Expected 80 bytes on 64-bit, got {size}");
    }
}

// Mirrors the internal CREDENTIAL struct for size verification.
// We must redeclare it here since the internal one is not accessible.
[System.Runtime.InteropServices.StructLayout(System.Runtime.InteropServices.LayoutKind.Sequential)]
file struct CREDENTIAL
{
    public uint Flags;
    public uint Type;
    public IntPtr TargetName;
    public IntPtr Comment;
    public long LastWritten;
    public uint CredentialBlobSize;
    public IntPtr CredentialBlob;
    public uint Persist;
    public uint AttributeCount;
    public IntPtr Attributes;
    public IntPtr TargetAlias;
    public IntPtr UserName;
}
