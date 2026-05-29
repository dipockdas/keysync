package
 io.keysync;

import java.util.List;

/**
 * Platform-specific keychain backend.
 *
 * <p>Implementations for macOS (security CLI), Linux (secret-tool), and
 * Windows (JNA + Credential Manager) are provided.
 */
public interface KeychainProvider {

    /**
     * Retrieves a secret from the platform keychain.
     *
     * @param service the keychain service name (e.g. "keysync/global")
     * @param account the account/key name (e.g. "API_KEY")
     * @return the secret value
     * @throws KeySyncException if the secret is not found or the keychain
     *         operation fails
     */
    String getSecret(String service, String account) throws KeySyncException;

    /**
     * Lists all keysync-managed secrets in the keychain.
     *
     * <p>Only returns credentials whose service name starts with "keysync/".
     *
     * @return a list of credential entries (never null, may be empty)
     * @throws KeySyncException if the keychain operation fails
     */
    List<Credential> listSecrets() throws KeySyncException;
}
