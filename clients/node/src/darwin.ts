import { execFile } from "node:child_process";
import { KeySyncError } from "./index.js";

/** macOS keychain access via the built-in `security` CLI. */
export async function darwinGet(service: string, account: string): Promise<string> {
  const { stdout, stderr } = await execFilePromise("security", [
    "find-generic-password",
    "-s", service,
    "-a", account,
    "-w",
  ]);
  if (stderr && !stdout) {
    throw new KeySyncError("notFound", `secret not found: ${service}/${account}`);
  }
  return stdout.trim();
}

/** List all keysync secrets on macOS by parsing `security dump-keychain`. */
export async function darwinList(): Promise<Array<{ service: string; account: string }>> {
  const { stdout } = await execFilePromise("security", ["dump-keychain"]);
  const records = stdout.split("\nkeychain:");
  const results: Array<{ service: string; account: string }> = [];

  for (const rec of records) {
    if (!rec.includes(`class: "genp"`)) continue;
    const svc = findAttrValue(rec, "svce");
    if (!svc?.startsWith("keysync/")) continue;
    const acct = findAttrValue(rec, "acct");
    if (acct) {
      results.push({ service: svc, account: acct });
    }
  }
  return results;
}

function findAttrValue(record: string, attrName: string): string | undefined {
  const idx = record.indexOf(`"${attrName}"`);
  if (idx < 0) return undefined;
  const after = record.slice(idx + attrName.length + 2);
  const eqIdx = after.indexOf("=");
  if (eqIdx < 0) return undefined;
  let val = after.slice(eqIdx + 1).trim();
  if (val === "<NULL>") return undefined;
  if (val.startsWith('"')) {
    const end = val.indexOf('"', 1);
    return end >= 0 ? val.slice(1, end) : val.slice(1);
  }
  return val.replace(/^"|"$/g, "");
}

/** Wrapper around child_process.execFile that returns a promise. */
function execFilePromise(
  cmd: string,
  args: string[]
): Promise<{ stdout: string; stderr: string }> {
  return new Promise((resolve, reject) => {
    execFile(cmd, args, (error, stdout, stderr) => {
      if (error) {
        // security returns exit code 44 for "item not found"
        if ((error as NodeJS.ErrnoException).code === 44) {
          resolve({ stdout: "", stderr: "not found" });
        } else {
          reject(error);
        }
      } else {
        resolve({ stdout, stderr });
      }
    });
  });
}
