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
 *   <li>If an environment is provided, check environment scope first
 *       ({@code keysync/project/<project>/env/<env>})</li>
 *   <li>If a project is provided, check project scope next</li>
 *   <li>Finally, fall back to global scope</li>
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
 * // Get an environment-scoped secret (falls back to project, then global)
 * String stagingDb = client.getSecret("DATABASE_URL", "myapp", "staging");
 *
 * // List all global secrets
 * List<Credential> globals = client.listSecrets();
 *
 * // List project secrets (includes global fallback)
 * List<Credential> project = client.listSecrets("myapp");
 *
 * // List environment-scoped secrets
 * List<Credential> envSecrets = client.listSecrets("myapp", "staging");
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
     * <p>Equivalent to {@code getSecret(key, null, null)}.
     *
     * @param key the secret key name (e.g. "API_KEY")
     * @return the secret value
     * @throws KeySyncException if the secret is not found or the platform is
     *         unsupported
     */
    public String getSecret(String key) throws KeySyncException {
        return getSecret(key, null, null);
    }

    /**
     * Retrieves a secret, optionally scoped to a project.
     *
     * <p>Equivalent to {@code getSecret(key, project, null)}.
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
        return getSecret(key, project, null);
    }

    /**
     * Retrieves a secret, optionally scoped to a project and environment.
     *
     * <p>Checks the environment variable {@code key} first. If set, returns it
     * immediately without touching the OS keychain. This is the primary path
     * for both local development (where secrets are injected via
     * {@code eval $(keysync export)}) and cloud deployments (where platforms
     * inject environment variables directly).
     *
     * <p>If the env var is not set, falls back to the OS keychain with the
     * following resolution order:
     * <ol>
     *   <li>Environment-scoped ({@code keysync/project/<project>/env/<env>})
     *       -- if both project and environment are provided</li>
     *   <li>Project-scoped ({@code keysync/project/<project>})
     *       -- if project is provided</li>
     *   <li>Global scope ({@code keysync/global})</li>
     * </ol>
     *
     * @param key         the secret key name (e.g. "DATABASE_URL")
     * @param project     optional project name; if provided, project scope is
     *                    checked before global scope
     * @param environment optional environment name (e.g. "staging", "production");
     *                    if both project and environment are provided,
     *                    environment scope is checked first
     * @return the secret value
     * @throws KeySyncException if the secret is not found in any scope or
     *         the platform is unsupported
     */
    public String getSecret(String key, String project, String environment) throws KeySyncException {
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

        // 3. If project is provided, check environment scope first, then project scope
        if (project != null && !project.isEmpty()) {
            // 3a. Try environment-scoped (if environment is provided)
            if (environment != null && !environment.isEmpty()) {
                String envService = serviceName("project", project, environment);
                try {
                    return provider.getSecret(envService, key);
                } catch (KeySyncException e) {
                    if (e.getError() != KeySyncError.NOT_FOUND) {
                        throw e; // re-throw non-not-found errors
                    }
                    // Fall through to project scope
                }
            }

            // 3b. Try project scope
            String projectService = serviceName("project", project);
            try {
                return provider.getSecret(projectService, key);
            } catch (KeySyncException e) {
                if (e.getError() != KeySyncError.NOT_FOUND) {
                    throw e; // re-throw non-not-found errors
                }
                // Fall through to global scope
            }
        }

        // 4. Fall back to global scope
        String globalService = serviceName("global", "");
        return provider.getSecret(globalService, key);
    }

    /**
     * Lists all global secrets.
     *
     * <p>Equivalent to {@code listSecrets(null, null)}.
     *
     * @return a list of credentials (never null, may be empty)
     * @throws KeySyncException if the platform is unsupported
     */
    public List<Credential> listSecrets() throws KeySyncException {
        return listSecrets(null, null);
    }

    /**
     * Lists all secrets, optionally filtered by project.
     *
     * <p>Equivalent to {@code listSecrets(project, null)}.
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
        return listSecrets(project, null);
    }

    /**
     * Lists all secrets, optionally filtered by project and environment.
     *
     * <p>When {@code project} is non-null and non-empty, filters to secrets
     * whose service matches the given project. When {@code environment} is
     * also non-null and non-empty, further filters to environment-scoped
     * secrets ({@code keysync/project/<project>/env/<env>}).
     *
     * <p>Global secrets ({@code keysync/global}) are always included when
     * a project filter is provided.
     *
     * @param project     optional project name to filter by (or null for all scopes)
     * @param environment optional environment name to filter by (or null for all)
     * @return a list of credentials (never null, may be empty)
     * @throws KeySyncException if the platform is unsupported
     */
    public List<Credential> listSecrets(String project, String environment) throws KeySyncException {
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

        // Build matching services
        String globalService = serviceName("global", "");
        String projectService = serviceName("project", project);
        String envService = (environment != null && !environment.isEmpty())
                ? serviceName("project", project, environment)
                : null;

        List<Credential> filtered = new ArrayList<>();
        for (Credential c : all) {
            String svc = c.getService();
            if (globalService.equals(svc)
                    || projectService.equals(svc)
                    || (envService != null && envService.equals(svc))) {
                // Parse service name to attach environment to credential
                String[] parsed = parseServiceName(svc);
                String credEnv = parsed.length > 2 ? parsed[2] : null;
                Credential enriched = new Credential(svc, c.getAccount(), credEnv);
                filtered.add(enriched);
            }
        }
        return filtered;
    }

    // --- Service name helpers ---

    /**
     * Builds the keychain service name.
     *
     * <ul>
     *   <li>Global:       {@code "keysync/global"}</li>
     *   <li>Project:      {@code "keysync/project/<name>"}</li>
     *   <li>Environment:  {@code "keysync/project/<name>/env/<env>"}</li>
     * </ul>
     */
    static String serviceName(String scope, String project) {
        return serviceName(scope, project, null);
    }

    /**
     * Builds the keychain service name with optional environment.
     *
     * @param scope       "global" or "project"
     * @param project     project name (ignored for global scope)
     * @param environment optional environment name (appended as /env/<env>)
     */
    static String serviceName(String scope, String project, String environment) {
        if (project == null || project.isEmpty() || "global".equals(scope)) {
            return "keysync/" + scope;
        }
        if (environment != null && !environment.isEmpty()) {
            return "keysync/" + scope + "/" + project + "/env/" + environment;
        }
        return "keysync/" + scope + "/" + project;
    }

    /**
     * Parses a service name back into scope, project, and environment.
     *
     * <pre>
     * "keysync/global"                      → ("global", null, null)
     * "keysync/project/my-app"              → ("project", "my-app", null)
     * "keysync/project/my-app/env/staging"  → ("project", "my-app", "staging")
     * "keysync/project/a/b"                 → ("project", "a/b", null)
     * </pre>
     */
    static String[] parseServiceName(String service) {
        if (service == null || service.length() < 8 || !service.startsWith("keysync/")) {
            return new String[]{"global", null, null};
        }

        String trimmed = service.substring("keysync/".length());
        int slashIdx = trimmed.indexOf('/');

        if (slashIdx < 0) {
            return new String[]{trimmed, null, null};
        }

        String scope = trimmed.substring(0, slashIdx);
        String rest = trimmed.substring(slashIdx + 1);

        if (!"project".equals(scope)) {
            return new String[]{scope, null, null};
        }

        // Check for /env/ segment to detect environment
        int envIdx = rest.indexOf("/env/");
        if (envIdx > 0) {
            String project = rest.substring(0, envIdx);
            String environment = rest.substring(envIdx + "/env/".length());
            return new String[]{scope, project, environment};
        }

        return new String[]{scope, rest, null};
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
