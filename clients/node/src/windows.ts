import { execFile } from "node:child_process";
import { KeySyncError } from "./index.js";

// ---------------------------------------------------------------------------
// Service ↔ target name conversion
// ---------------------------------------------------------------------------

/** Convert keysync service name to Windows Credential Manager target name.
 *  Strips /env/ keyword, then replaces slashes with underscores.
 *  "keysync/project/myapp/env/dev" → "keysync_project_myapp_dev"
 *  "keysync/project/myapp"         → "keysync_project_myapp"
 *  "keysync/global"                → "keysync_global"
 */
function serviceToTarget(service: string): string {
  return service.replace(/\/env\//g, "_").replace(/\//g, "_");
}

/** Convert Windows Credential Manager target back to a keysync service name.
 *  "keysync_global"                → "keysync/global"
 *  "keysync_project_myapp"         → "keysync/project/myapp"
 *  "keysync_project_myapp_dev"     → "keysync/project/myapp/env/dev"
 */
function targetToService(target: string): string {
  const parts = target.split("_");
  if (parts.length >= 2 && parts[1] === "global") {
    return "keysync/global";
  }
  if (parts.length >= 3 && parts[1] === "project") {
    const extra = parts.slice(2);
    if (extra.length >= 2) {
      // Last part is environment, rest is project name.
      const env = extra[extra.length - 1];
      const project = extra.slice(0, -1).join("_");
      return `keysync/project/${project}/env/${env}`;
    }
    return `keysync/project/${extra.join("_")}`;
  }
  return target;
}

// ---------------------------------------------------------------------------
// Windows Credential Manager via PowerShell (Win32 API)
// ---------------------------------------------------------------------------

/** PowerShell script that defines C# helpers and reads a generic credential.
 *  Uses CredReadW from advapi32.dll — no native module dependency. */
function readCredPS(target: string): string {
  const escaped = target.replace(/'/g, "''");
  return `Add-Type @'
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
try { [KSCred]::Read('${escaped}') } catch { "" }`;
}

/** PowerShell script that lists all keysync credentials via CredEnumerateW. */
function listCredsPS(): string {
  return `Add-Type @'
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
try { [KSCred]::List() } catch { "" }`;
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/** Read a secret from Windows Credential Manager by service + account name. */
export async function windowsGet(
  service: string,
  account: string
): Promise<string> {
  const target = serviceToTarget(service);
  const ps = readCredPS(target);

  let stdout: string;
  try {
    const result = await execFilePromise("powershell.exe", [
      "-NoProfile",
      "-NonInteractive",
      "-Command",
      ps,
    ]);
    stdout = result.stdout.trim();
  } catch (err) {
    throw new KeySyncError(
      "keychainError",
      `credential read failed: ${err}`
    );
  }

  if (!stdout) {
    throw new KeySyncError(
      "notFound",
      `secret not found: ${service}/${account}`
    );
  }

  // Output format: "userName\tsecret"
  const tabIdx = stdout.indexOf("\t");
  const userName = tabIdx >= 0 ? stdout.slice(0, tabIdx) : "";
  const secret = tabIdx >= 0 ? stdout.slice(tabIdx + 1) : stdout;

  if (!secret) {
    throw new KeySyncError(
      "notFound",
      `secret not found: ${service}/${account}`
    );
  }

  // The credential's UserName is the key name — verify it matches
  if (userName !== account) {
    throw new KeySyncError(
      "notFound",
      `account mismatch: expected ${account}, got ${userName}`
    );
  }

  return secret;
}

/** List all keysync secrets from Windows Credential Manager. */
export async function windowsList(): Promise<
  Array<{ service: string; account: string }>
> {
  const ps = listCredsPS();

  let stdout: string;
  try {
    const result = await execFilePromise("powershell.exe", [
      "-NoProfile",
      "-NonInteractive",
      "-Command",
      ps,
    ]);
    stdout = result.stdout.trim();
  } catch {
    return [];
  }

  if (!stdout) return [];

  const results: Array<{ service: string; account: string }> = [];
  for (const line of stdout.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    const tabIdx = trimmed.indexOf("\t");
    const target = tabIdx >= 0 ? trimmed.slice(0, tabIdx) : trimmed;
    const userName = tabIdx >= 0 ? trimmed.slice(tabIdx + 1) : "";

    if (!target.startsWith("keysync_")) continue;

    const service = targetToService(target);
    if (!service) continue;

    results.push({ service, account: userName });
  }

  return results;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function execFilePromise(
  cmd: string,
  args: string[]
): Promise<{ stdout: string; stderr: string }> {
  return new Promise((resolve, reject) => {
    execFile(cmd, args, { maxBuffer: 1024 * 1024 }, (error, stdout, stderr) => {
      if (error) {
        reject(error);
      } else {
        resolve({ stdout, stderr });
      }
    });
  });
}
