import { execFile } from "node:child_process";
import { KeySyncError } from "./index.js";

/** Linux keychain access via the `secret-tool` CLI (libsecret). */
export async function linuxGet(service: string, account: string): Promise<string> {
  try {
    const { stdout } = await execFilePromise("secret-tool", [
      "lookup", "service", service, "account", account,
    ]);
    const val = stdout.trim();
    if (!val) throw new KeySyncError("notFound", "secret not found");
    return val;
  } catch (err) {
    if (err instanceof KeySyncError) throw err;
    throw new KeySyncError("keychainError", `secret-tool lookup failed: ${err}`);
  }
}

/** List all keysync secrets on Linux by parsing `secret-tool search`. */
export async function linuxList(): Promise<Array<{ service: string; account: string }>> {
  let stdout: string;
  try {
    const result = await execFilePromise("secret-tool", [
      "search", "service", "keysync",
    ]);
    stdout = result.stdout;
  } catch {
    return []; // secret-tool not available or no results
  }

  const results: Array<{ service: string; account: string }> = [];
  let currentSvc = "";
  let currentAcct = "";

  for (const line of stdout.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) {
      if (currentSvc && currentAcct && currentSvc.startsWith("keysync/")) {
        results.push({ service: currentSvc, account: currentAcct });
      }
      currentSvc = "";
      currentAcct = "";
      continue;
    }
    if (trimmed.startsWith("service")) {
      currentSvc = parseAttrValue(trimmed);
    } else if (trimmed.startsWith("account")) {
      currentAcct = parseAttrValue(trimmed);
    }
  }
  // Handle last entry if no trailing blank line
  if (currentSvc && currentAcct && currentSvc.startsWith("keysync/")) {
    results.push({ service: currentSvc, account: currentAcct });
  }
  return results;
}

function parseAttrValue(line: string): string {
  const eqIdx = line.indexOf("=");
  return eqIdx >= 0 ? line.slice(eqIdx + 1).trim() : "";
}

function execFilePromise(
  cmd: string,
  args: string[]
): Promise<{ stdout: string; stderr: string }> {
  return new Promise((resolve, reject) => {
    execFile(cmd, args, (error, stdout, stderr) => {
      if (error) {
        reject(error);
      } else {
        resolve({ stdout, stderr });
      }
    });
  });
}
