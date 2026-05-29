package io.keysync;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * macOS keychain access via the built-in {@code security} CLI.
 *
 * <p>Uses {@code security find-generic-password} to retrieve secrets stored
 * by keysync. No native library required — the {@code security} tool ships
 * with every macOS installation.
 *
 * <p>On newer macOS versions (13+), the password is written to stderr.
 * This implementation captures both stdout and stderr as a result.
 */
class MacKeychain implements KeychainProvider {

    @Override
    public String getSecret(String service, String account) throws KeySyncException {
        ProcessBuilder pb = new ProcessBuilder(
                "security", "find-generic-password",
                "-s", service,
                "-a", account,
                "-w"
        );
        pb.redirectErrorStream(false);

        try {
            Process process = pb.start();

            // Read stdout
            String stdout = readString(process.getInputStream());
            // Read stderr (on macOS 13+ the password may appear here)
            String stderr = readString(process.getErrorStream());

            int exitCode = process.waitFor();

            if (exitCode == 44) {
                // security returns exit code 44 for "item not found"
                throw new KeySyncException(
                        KeySyncError.NOT_FOUND,
                        "secret not found: " + service + "/" + account
                );
            }

            if (exitCode != 0) {
                String errorDetail = stderr.isEmpty() ? stdout : stderr;
                throw new KeySyncException(
                        KeySyncError.KEYCHAIN_ERROR,
                        "security find-generic-password failed with code " + exitCode
                            + ": " + errorDetail.trim()
                );
            }

            // Combine stdout and stderr; trim whitespace
            String value = stdout.trim();
            if (value.isEmpty()) {
                value = stderr.trim();
            }
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
                    "failed to run security command: " + e.getMessage(),
                    e
            );
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "interrupted while waiting for security command",
                    e
            );
        }
    }

    @Override
    public List<Credential> listSecrets() throws KeySyncException {
        // security find-generic-password -s keysync/ returns items matching
        // that service prefix. We scan both "keysync/global" and all
        // "keysync/project/*" entries by enumerating the keychain.

        // Use security dump-keychain to get all generic passwords
        ProcessBuilder pb = new ProcessBuilder("security", "dump-keychain");
        pb.redirectErrorStream(true);

        try {
            Process process = pb.start();
            String output = readString(process.getInputStream());
            int exitCode = process.waitFor();

            if (exitCode != 0) {
                // Return empty list on failure; keychain may be locked
                return Collections.emptyList();
            }

            List<Credential> results = new ArrayList<>();
            String[] records = output.split("\\nkeychain:");

            for (String record : records) {
                record = record.trim();
                if (record.isEmpty() || !record.contains("class: \"genp\"")) {
                    continue;
                }

                String service = extractAttribute(record, "svce");
                if (!service.startsWith("keysync/")) {
                    continue;
                }

                String account = extractAttribute(record, "acct");
                if (account.isEmpty()) {
                    continue;
                }

                results.add(new Credential(service, account));
            }

            return results;

        } catch (IOException e) {
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "failed to run security dump-keychain: " + e.getMessage(),
                    e
            );
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR,
                    "interrupted while dumping keychain",
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
     * Extracts an attribute value from a dump-keychain record.
     * Records contain lines like: "svce"<blob>=0x...  "keysync/global"
     * or:                         0x00000007 <blob>="keysync/global"
     */
    static String extractAttribute(String record, String attrName) {
        // Look for the quoted attribute name
        String quotedAttr = "\"" + attrName + "\"";
        int idx = record.indexOf(quotedAttr);
        if (idx < 0) {
            return "";
        }

        String after = record.substring(idx + quotedAttr.length());

        // Find the value: skip past the blob part and '='
        int eqIdx = after.indexOf('=');
        if (eqIdx < 0) {
            return "";
        }

        String raw = after.substring(eqIdx + 1).trim();

        // Handle <NULL>
        if ("<NULL>".equals(raw)) {
            return "";
        }

        // If value starts with a hex blob prefix like "0x00000007 ", skip it
        int spaceIdx = raw.indexOf("  ");
        if (spaceIdx >= 0) {
            raw = raw.substring(spaceIdx).trim();
        }

        // Strip surrounding quotes
        if (raw.startsWith("\"") && raw.endsWith("\"")) {
            raw = raw.substring(1, raw.length() - 1);
        }

        return raw;
    }
}
