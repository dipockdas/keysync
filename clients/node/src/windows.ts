import { KeySyncError } from "./index.js";

/**
 * Windows keychain access.
 *
 * The Windows Credential Manager cannot be accessed via command-line tools
 * for reading values (cmdkey only lists names, not values). A companion
 * helper binary (compiled from the Go client library) is needed.
 *
 * For now, this throws an error directing users to the Go/Python clients.
 */
export async function windowsGet(_service: string, _account: string): Promise<string> {
  throw new KeySyncError(
    "unsupportedPlatform",
    "Windows is not yet supported in the Node client. " +
    "Use the Go client (github.com/dipockdas/keysync/clients/go) or " +
    "the Python client."
  );
}

export async function windowsList(): Promise<Array<{ service: string; account: string }>> {
  throw new KeySyncError(
    "unsupportedPlatform",
    "Windows is not yet supported in the Node client."
  );
}
