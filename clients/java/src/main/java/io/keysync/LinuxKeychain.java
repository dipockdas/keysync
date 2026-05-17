package io.keysync;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * Linux keychain access via libsecret's {@code secret-tool} CLI.
 *
 * <p>Requires the {@code libsecret-tools} package to be installed and a
 * running secret service (GNOME Keyring, KDE Wallet, KeePassXC, etc.).
 */
class LinuxKeychain implements KeychainProvider {

    @Override
    public String getSecret(String service, String account) throws KeySyncException {
        ProcessBuilder pb = new ProcessBuilder(
                "secret-tool", "lookup",
                "keysync-service", service,
                "keysync-key", account
        );
        pb.redirectErrorStream(false);

        try {
            Process process = pb.start();
            String stdout = readString(process.getInputStream());
            String stderr = readString(process.getErrorStream());
            int exitCode = process.waitFor();

            if (exitCode == 1) {
                // secret-tool returns 1 when the secret is not found
                throw new KeySyncException(
                        KeySyncError.NOT_FOUND,
                        "secret not found: " + service + "/" + account
                );
            }

            if (exitCode != 0) {
                throw new KeySyncException(
                        KeySyncError.KEYCHAIN_ERROR,
                        "secret-tool lookup failed with code " + exitCode
                            + ": " + stderr.trim()
                );
            }

            String value = stdout.trim();
            if (value.isEmpty()) {
                throw new KeySyncException(
                        KeySyncError.NOT_FOUND,
                        "secret not found: " + service + "/" + account
                );
            }
            return value;

        } catch (IOException e) {
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "failed to run secret-tool: " + e.getMessage()
                        + " (is libsecret-tools installed?)",
                    e
            );
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "interrupted while waiting for secret-tool",
                    e
            );
        }
    }

    @Override
    public List<Credential> listSecrets() throws KeySyncException {
        ProcessBuilder pb = new ProcessBuilder("secret-tool", "search", "keysync-service", "keysync");
        pb.redirectErrorStream(true);

        try {
            Process process = pb.start();
            String output = readString(process.getInputStream());
            int exitCode = process.waitFor();

            if (exitCode != 0) {
                // secret-tool not available or no results
                return Collections.emptyList();
            }

            List<Credential> results = new ArrayList<>();
            String currentService = null;
            String currentAccount = null;

            for (String line : output.split("\n")) {
                line = line.trim();
                if (line.isEmpty()) {
                    // Blank line = end of entry
                    if (currentService != null && currentAccount != null) {
                        results.add(new Credential(currentService, currentAccount));
                    }
                    currentService = null;
                    currentAccount = null;
                    continue;
                }

                if (line.startsWith("keysync-service")) {
                    currentService = parseValue(line);
                } else if (line.startsWith("keysync-key")) {
                    currentAccount = parseValue(line);
                }
            }

            // Handle last entry if no trailing blank line
            if (currentService != null && currentAccount != null) {
                results.add(new Credential(currentService, currentAccount));
            }

            return results;

        } catch (IOException e) {
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "failed to run secret-tool search: " + e.getMessage(),
                    e
            );
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "interrupted while searching keychain",
                    e
            );
        }
    }

    // --- helpers ---

    private static String readString(java.io.InputStream stream) throws IOException {
        StringBuilder sb = new StringBuilder();
        try (BufferedReader reader = new BufferedReader(
                new InputStreamReader(stream, StandardCharsets.UTF_8))) {
            String line;
            while ((line = reader.readLine()) != null) {
                if (sb.length() > 0) {
                    sb.append('\n');
                }
                sb.append(line);
            }
        }
        return sb.toString();
    }

    /**
     * Parses "keysync-service = keysync/global" and returns "keysync/global".
     */
    private static String parseValue(String line) {
        int eqIdx = line.indexOf('=');
        if (eqIdx < 0) {
            return "";
        }
        return line.substring(eqIdx + 1).trim();
    }
}
