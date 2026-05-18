# frozen_string_literal: true

require "open3"
require_relative "errors"

module KeySync
  # Windows keychain access via PowerShell with inline C#.
  #
  # Uses CredReadW and CredEnumerateW from advapi32.dll via
  # PowerShell scripts that define C# P/Invoke helpers at runtime.
  # This approach requires no native gem dependencies or compilation.
  module Windows
    module_function

    # Convert a keysync service name to a Windows Credential Manager
    # target name suitable for storage. Strips the /env/ keyword and
    # replaces slashes with underscores.
    #
    # "keysync/global"                    => "keysync_global"
    # "keysync/project/app"               => "keysync_project_app"
    # "keysync/project/app/env/staging"   => "keysync_project_app_staging"
    #
    # @param service [String] keysync service name
    # @return [String] target name with environment marker removed
    def service_to_target(service)
      # Strip /env/ keyword so environment is just part of the path
      processed = service.gsub("/env/", "/")
      processed.tr("/", "_")
    end

    # Convert a Windows target name back to a keysync service name.
    #
    # "keysync_global"                   => "keysync/global"
    # "keysync_project_myapp"            => "keysync/project/myapp"
    # "keysync_project_myapp_staging"    => "keysync/project/myapp/env/staging"
    #
    # @param target [String] the Windows credential target name
    # @return [String, nil] the keysync service name, or nil if not a keysync target
    def target_to_service(target)
      parts = target.split("_")
      if parts.length >= 2 && parts[1] == "global"
        "keysync/global"
      elsif parts.length >= 3 && parts[1] == "project"
        if parts.length >= 4
          # 3+ segments: project + env
          "keysync/project/#{parts[2]}/env/#{parts[3..].join('_')}"
        else
          # Exactly 2 segments: just project
          "keysync/project/#{parts[2]}"
        end
      end
    end

    # PowerShell script that defines C# helpers and reads a generic credential
    # using CredReadW from advapi32.dll.
    #
    # @param target [String] the Windows credential target name
    # @return [String] PowerShell script text
    def read_cred_ps(target)
      escaped = target.gsub("'", "''")
      <<~PWSH
        Add-Type @'
        using System;
        using System.Runtime.InteropServices;

        [StructLayout(LayoutKind.Sequential)]
        public struct CREDENTIAL {
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

        public static class KSCred {
            [DllImport("advapi32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
            public static extern bool CredReadW(string target, int type, int flags, out IntPtr credential);

            [DllImport("advapi32.dll", SetLastError = true)]
            public static extern void CredFree(IntPtr buffer);

            public static string Read(string target) {
                IntPtr ptr;
                if (!CredReadW(target, 1, 0, out ptr)) return "";
                try {
                    CREDENTIAL cred = (CREDENTIAL)Marshal.PtrToStructure(ptr, typeof(CREDENTIAL));
                    string userName = "";
                    string secret = "";
                    if (cred.UserName != IntPtr.Zero) userName = Marshal.PtrToStringUni(cred.UserName) ?? "";
                    if (cred.CredentialBlobSize > 0 && cred.CredentialBlob != IntPtr.Zero)
                        secret = Marshal.PtrToStringUni(cred.CredentialBlob, (int)cred.CredentialBlobSize / 2) ?? "";
                    return userName + "\\t" + secret;
                } finally { CredFree(ptr); }
            }
        }
        '@
        try { [KSCred]::Read('#{escaped}') } catch { "" }
      PWSH
    end

    # PowerShell script that lists all keysync credentials via CredEnumerateW.
    #
    # @return [String] PowerShell script text
    def list_creds_ps
      <<~PWSH
        Add-Type @'
        using System;
        using System.Runtime.InteropServices;

        [StructLayout(LayoutKind.Sequential)]
        public struct CREDENTIAL {
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

        public static class KSCred {
            [DllImport("advapi32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
            public static extern bool CredEnumerateW(string filter, int flags, out int count, out IntPtr credentials);

            [DllImport("advapi32.dll", SetLastError = true)]
            public static extern void CredFree(IntPtr buffer);

            public static string List() {
                IntPtr ptr;
                int count;
                if (!CredEnumerateW("keysync_*", 0, out count, out ptr)) return "";
                try {
                    var results = new System.Collections.Generic.List<string>();
                    int ptrSize = Marshal.SizeOf(typeof(IntPtr));
                    for (int i = 0; i < count; i++) {
                        IntPtr itemPtr = Marshal.ReadIntPtr(ptr, i * ptrSize);
                        CREDENTIAL cred = (CREDENTIAL)Marshal.PtrToStructure(itemPtr, typeof(CREDENTIAL));
                        string target = cred.TargetName != IntPtr.Zero ? Marshal.PtrToStringUni(cred.TargetName) ?? "" : "";
                        string user = cred.UserName != IntPtr.Zero ? Marshal.PtrToStringUni(cred.UserName) ?? "" : "";
                        results.Add(target + "\\t" + user);
                    }
                    return string.Join("\\n", results);
                } finally { CredFree(ptr); }
            }
        }
        '@
        try { [KSCred]::List() } catch { "" }
      PWSH
    end

    # Run a PowerShell command and return its stdout.
    #
    # @param ps_script [String] the PowerShell script to execute
    # @return [String] stdout output
    def run_powershell(ps_script)
      stdout, stderr, status = Open3.capture3(
        "powershell.exe",
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        ps_script
      )
      return stdout
    rescue Errno::ENOENT
      raise KeySyncError.new("keychain_error",
        "powershell.exe not found"
      )
    end

    # Retrieve a secret from Windows Credential Manager.
    #
    # @param service [String] keychain service name
    # @param account [String] key/account name
    # @return [String] the secret value
    # @raise [SecretNotFoundError] if the item does not exist
    # @raise [KeySyncError] if the PowerShell call fails
    def get(service, account)
      target = service_to_target(service)
      ps_script = read_cred_ps(target)
      stdout = run_powershell(ps_script)

      output = stdout.strip
      if output.empty?
        raise SecretNotFoundError, "#{service}/#{account}"
      end

      # Output format: "userName\tsecret"
      tab_idx = output.index("\t")
      if tab_idx.nil?
        raise SecretNotFoundError, "#{service}/#{account}"
      end

      user_name = output[0...tab_idx]
      secret = output[(tab_idx + 1)..]

      if secret.empty?
        raise SecretNotFoundError, "#{service}/#{account}"
      end

      # The credential's UserName is the key name — verify it matches
      if user_name != account
        raise SecretNotFoundError, "account mismatch: expected #{account}, got #{user_name}"
      end

      secret
    end

    # List all keysync secrets from Windows Credential Manager.
    #
    # @return [Array<Hash>] array of {service:, account:} hashes
    def list
      ps_script = list_creds_ps
      stdout = run_powershell(ps_script)
      return [] if stdout.strip.empty?

      entries = []
      stdout.split("\n").each do |line|
        trimmed = line.strip
        next if trimmed.empty?

        tab_idx = trimmed.index("\t")
        if tab_idx.nil?
          next
        end

        target = trimmed[0...tab_idx]
        user_name = trimmed[(tab_idx + 1)..]

        svc = target_to_service(target)
        next if svc.nil?

        entries << { "service" => svc, "account" => user_name }
      end

      entries
    end
  end
end
