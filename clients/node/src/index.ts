import { serviceName, parseServiceName } from "./types.js";
import { darwinGet, darwinList } from "./darwin.js";
import { linuxGet, linuxList } from "./linux.js";
import { windowsGet, windowsList } from "./windows.js";

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

/** Errors thrown by the keysync client. */
export class KeySyncError extends Error {
  constructor(
    public readonly code: "notFound" | "keychainError" | "unsupportedPlatform",
    message: string
  ) {
    super(message);
    this.name = "KeySyncError";
  }
}

// ---------------------------------------------------------------------------
// Platform selection
// ---------------------------------------------------------------------------

type PlatformImpl = {
  get(service: string, account: string): Promise<string>;
  list(): Promise<Array<{ service: string; account: string }>>;
};

function selectPlatform(): PlatformImpl {
  const platform = process.platform;
  switch (platform) {
    case "darwin":
      return { get: darwinGet, list: darwinList };
    case "linux":
      return { get: linuxGet, list: linuxList };
    case "win32":
      return { get: windowsGet, list: windowsList };
    default:
      throw new KeySyncError(
        "unsupportedPlatform",
        `unsupported platform: ${platform}`
      );
  }
}

const platform: PlatformImpl = selectPlatform();

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Retrieve a secret from the OS keychain.
 *
 * If `project` is provided, checks project scope first, then falls back
 * to global scope. If `project` is omitted, only checks global scope.
 *
 * @param key - The secret key name (e.g. "DATABASE_URL").
 * @param project - Optional project name for project-scoped secrets.
 * @returns The secret value.
 * @throws {KeySyncError} with code "notFound" if the secret doesn't exist.
 */
export async function getSecret(key: string, project?: string): Promise<string> {
  // Try project scope first
  if (project) {
    const svc = serviceName("project", project);
    try {
      return await platform.get(svc, key);
    } catch (err) {
      if (err instanceof KeySyncError && err.code !== "notFound") {
        throw err;
      }
      // Fall through to global scope
    }
  }

  // Fall back to global scope
  const svc = serviceName("global");
  return await platform.get(svc, key);
}

/**
 * List all stored secret key names.
 *
 * @param scope - Filter by scope ("global" or "project").
 * @param project - Filter by project name.
 * @returns Array of { scope, project?, key } tuples.
 */
export async function listSecrets(
  scope?: string,
  project?: string
): Promise<Array<{ scope: string; project?: string; key: string }>> {
  const entries = await platform.list();
  return entries
    .map((entry) => {
      const parsed = parseServiceName(entry.service);
      return { scope: parsed.scope, project: parsed.project, key: entry.account };
    })
    .filter((entry) => {
      if (scope && entry.scope !== scope) return false;
      if (project && entry.project !== project) return false;
      return true;
    });
}
