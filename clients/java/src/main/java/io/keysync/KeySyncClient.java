package io.keysync;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * KeySync client — retrieve secrets from the OS keychain.
 *
 * <p>This library reads secrets from the OS-native keychain with zero
 * dependency on the keysync binary. Each platform uses its native keychain
 * tooling:
 * <ul>
 *   <li><b>macOS</b>: {@code security} CLI (built-in)</li>
 *   <li><b>Linux</b>: {@code secret-tool} CLI (libsecret)</li>
 *   <li><b>Windows</b>: JNA + Win32 Credential Manager API</li>
 * </ul>
 *
 * <h3>Resolution order</h3>
 * Every call to {@link #getSecret(String)} or {@link #getSecret(String, String)}
 * follows this order:
 * <ol>
 *   <li>Check environment variable ({@code System.getenv(key)}) —
 *       for cloud/CI where the platform injects env vars</li>
 *   <li>If not found, fall back to OS keychain</li>
 *   <li>If a project is provided, check project scope first, then global scope</li>
 * </ol>
 *
 * <h3>Usage</h3>
 * <pre>{@code
 * // Singleton client
 * KeySyncClient client = KeySyncClient.getInstance();
 *
 * // Get a global secret
 * String apiKey = client.getSecret("API_KEY");
 *
 * // Get a project-scoped secret (falls back to global)
 * String dbUrl = client.getSecret("DATABASE_URL", "myapp");
 *
 * // List all global secrets
 * List<Credential> globals = client.listSecrets();
 *
 * // List project secrets (includes global fallback)
 * List<Credential> project = client.listSecrets("myapp");
 * }</pre>
 */
public class KeySyncClient {

    private static volatile KeySyncClient instance;

    private final KeychainProvider provider;
    private final boolean isSupported;

    /**
     * Returns the singleton KeySyncClient instance.
     *
     * <p>The client is thread-safe and safe to create from multiple threads.
     */
    public static KeySyncClient getInstance() {
        if (instance == null) {
            synchronized (KeySyncClient.class) {
                if (instance == null) {
                    instance = new KeySyncClient();
                }
            }
        }
        return instance;
    }

    private KeySyncClient() {
        String os = System.getProperty("os.name").toLowerCase();

        if (os.contains("mac")) {
            this.provider = new MacKeychain();
            this.isSupported = true;
        } else if (os.contains("linux")) {
            this.provider = new LinuxKeychain();
            this.isSupported = true;
        } else if (os.contains("win")) {
            this.provider = new WindowsKeychain();
            this.isSupported = true;
        } else {
            this.provider = null;
            this.isSupported = false;
        }
    }

    /**
     * Retrieves a global secret.
     *
     * <p>Equivalent to {@code getSecret(key, null)}.
     *
     * @param key the secret key name (e.g. "API_KEY")
     * @return the secret value
     * @throws KeySyncException if the secret is not found or the platform is
     *         unsupported
     */
    public String getSecret(String key) throws KeySyncException {
        return getSecret(key, null);
    }

    /**
     * Retrieves a secret, optionally scoped to a project.
     *
     * <p>Checks the environment variable {@code key} first. If set, returns it
     * immediately without touching the OS keychain. This is the primary path
     * for both local development (where secrets are injected via
     * {@code eval $(keysync export)}) and cloud deployments (where platforms
     * inject environment variables directly).
     *
     * <p>If the env var is not set, falls back to the OS keychain. When
     * {@code project} is non-null and non-empty, checks project scope first,
     * then global scope.
     *
     * @param key     the secret key name (e.g. "DATABASE_URL")
     * @param project optional project name; if provided, project scope is
     *                checked first, then global scope
     * @return the secret value
     * @throws KeySyncException if the secret is not found in any scope or
     *         the platform is unsupported
     */
    public String getSecret(String key, String project) throws KeySyncException {
        // 1. Check environment variable first (primary path for cloud/CI)
        String envVal = System.getenv(key);
        if (envVal != null) {
            return envVal;
        }

        // 2. Platform check
        if (!isSupported || provider == null) {
            throw new KeySyncException(
                    KeySyncError.UNSUPPORTED_PLATFORM,
                    "keysync client: unsupported platform (only macOS, Linux, and Windows are supported)"
            );
        }

        // 3. Try project scope first
        if (project != null && !project.isEmpty()) {
            String service = serviceName("project", project);
            try {
                return provider.getSecret(service, key);
            } catch (KeySyncException e) {
                if (e.getError() != KeySyncError.NOT_FOUND) {
                    throw e; // re-throw non-not-found errors
                }
                // Fall through to global scope
            }
        }

        // 4. Fall back to global scope
        String service = serviceName("global", "");
        return provider.getSecret(service, key);
    }

    /**
     * Lists all global secrets.
     *
     * <p>Equivalent to {@code listSecrets(null)}.
     *
     * @return a list of credentials (never null, may be empty)
     * @throws KeySyncException if the platform is unsupported
     */
    public List<Credential> listSecrets() throws KeySyncException {
        return listSecrets(null);
    }

    /**
     * Lists all secrets, optionally filtered by project.
     *
     * <p>When {@code project} is non-null and non-empty, returns only secrets
     * whose service matches {@code keysync/project/<project>} or
     * {@code keysync/global}.
     *
     * @param project optional project name to filter by (or null for all scopes)
     * @return a list of credentials (never null, may be empty)
     * @throws KeySyncException if the platform is unsupported
     */
    public List<Credential> listSecrets(String project) throws KeySyncException {
        if (!isSupported || provider == null) {
            throw new KeySyncException(
                    KeySyncError.UNSUPPORTED_PLATFORM,
                    "keysync client: unsupported platform (only macOS, Linux, and Windows are supported)"
            );
        }

        List<Credential> all = provider.listSecrets();

        if (project == null || project.isEmpty()) {
            return all;
        }

        // Filter: match either keysync/project/<project> or keysync/global
        String projectService = serviceName("project", project);
        String globalService = serviceName("global", "");

        List<Credential> filtered = new ArrayList<>();
        for (Credential c : all) {
            if (projectService.equals(c.getService()) || globalService.equals(c.getService())) {
                filtered.add(c);
            }
        }
        return filtered;
    }

    // --- Service name helpers ---

    /**
     * Builds the keychain service name.
     *
     * <ul>
     *   <li>Global:  {@code "keysync/global"}</li>
     *   <li>Project: {@code "keysync/project/<name>"}</li>
     * </ul>
     */
    static String serviceName(String scope, String project) {
        if (project == null || project.isEmpty() || "global".equals(scope)) {
            return "keysync/" + scope;
        }
        return "keysync/" + scope + "/" + project;
    }

    /**
     * Parses a service name back into scope and project.
     *
     * <pre>
     * "keysync/global"          → ("global", null)
     * "keysync/project/my-app"  → ("project", "my-app")
     * "keysync/project/a/b"     → ("project", "a/b")
     * </pre>
     */
    static String[] parseServiceName(String service) {
        if (service == null || service.length() < 8 || !service.startsWith("keysync/")) {
            return new String[]{"global", null};
        }

        String trimmed = service.substring("keysync/".length());
        int slashIdx = trimmed.indexOf('/');

        if (slashIdx < 0) {
            return new String[]{trimmed, null};
        }

        String scope = trimmed.substring(0, slashIdx);
        if (!"project".equals(scope)) {
            return new String[]{scope, null};
        }

        String project = trimmed.substring(slashIdx + 1);
        return new String[]{scope, project};
    }

    /**
     * Returns the OS name string for diagnostics (e.g. "mac os x", "linux").
     */
    public String getPlatform() {
        return System.getProperty("os.name").toLowerCase();
    }

    /**
     * Returns true if the current platform is supported.
     */
    public boolean isPlatformSupported() {
        return isSupported;
    }
}
