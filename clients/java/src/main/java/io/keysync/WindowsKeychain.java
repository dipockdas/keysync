package io.keysync;

import com.sun.jna.Native;
import com.sun.jna.Pointer;
import com.sun.jna.Structure;
import com.sun.jna.WString;
import com.sun.jna.ptr.IntByReference;
import com.sun.jna.ptr.PointerByReference;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * Windows keychain access via the Win32 Credential Manager API using JNA.
 *
 * <p>Uses {@code CredReadW} and {@code CredEnumerateW} from advapi32.dll to
 * read and list credentials. The target name uses underscores instead of
 * slashes:
 * <ul>
 *   <li>{@code "keysync/global"} becomes {@code "keysync_global"}</li>
 *   <li>{@code "keysync/project/myapp"} becomes {@code "keysync_project_myapp"}</li>
 * </ul>
 *
 * <p>Requires the JNA library ({@code net.java.dev.jna:jna}) on the classpath.
 */
class WindowsKeychain implements KeychainProvider {

    /**
     * JNA interface for advapi32.dll Win32 Credential Manager functions.
     */
    public interface Advapi32 extends com.sun.jna.Library {
        Advapi32 INSTANCE = com.sun.jna.Native.load("Advapi32", Advapi32.class,
                Collections.singletonMap(com.sun.jna.Library.OPTION_STRING_ENCODING, "UTF-8"));

        /**
         * Reads a credential from the user's credential set.
         *
         * @param targetName the name of the credential to read
         * @param type       the credential type (1 = CRED_TYPE_GENERIC)
         * @param flags      reserved, must be 0
         * @param credential receives a pointer to the credential
         * @return true on success, false on failure (call GetLastError)
         */
        boolean CredReadW(WString targetName, int type, int flags,
                          PointerByReference credential);

        /**
         * Frees a credential buffer returned by CredReadW.
         *
         * @param credential pointer to the credential to free
         */
        void CredFree(Pointer credential);

        /**
         * Enumerates credentials matching a filter.
         *
         * @param filter   the filter string (e.g. "keysync_*")
         * @param flags    reserved, must be 0
         * @param count    receives the number of credentials returned
         * @param creds    receives a pointer to an array of credential pointers
         * @return true on success, false on failure
         */
        boolean CredEnumerateW(WString filter, int flags, IntByReference count,
                               PointerByReference creds);
    }

    /**
     * JNA Structure that mirrors the Win32 CREDENTIALW struct.
     *
     * <pre>
     * typedef struct _CREDENTIALW {
     *     DWORD   Flags;
     *     DWORD   Type;
     *     LPWSTR  TargetName;
     *     LPWSTR  Comment;
     *     FILETIME LastWritten;
     *     DWORD   CredentialBlobSize;
     *     LPBYTE  CredentialBlob;
     *     DWORD   Persist;
     *     DWORD   AttributeCount;
     *     PCREDENTIAL_ATTRIBUTEW Attributes;
     *     LPWSTR  TargetAlias;
     *     LPWSTR  UserName;
     * } CREDENTIALW, *PCREDENTIALW;
     * </pre>
     *
     * Note: FILETIME is a 64-bit value, mapped to a single long. JNA handles
     * the alignment correctly via {@code Structure.ALIGN_DEFAULT}.
     */
    @Structure.FieldOrder({
            "Flags", "Type", "TargetName", "Comment", "LastWritten",
            "CredentialBlobSize", "CredentialBlob", "Persist",
            "AttributeCount", "Attributes", "TargetAlias", "UserName"
    })
    public static class CREDENTIALW extends Structure {

        public int Flags;
        public int Type;
        public Pointer TargetName;       // LPWSTR
        public Pointer Comment;          // LPWSTR
        public long LastWritten;         // FILETIME (8 bytes)
        public int CredentialBlobSize;
        public Pointer CredentialBlob;   // LPBYTE
        public int Persist;
        public int AttributeCount;
        public Pointer Attributes;       // PCREDENTIAL_ATTRIBUTE
        public Pointer TargetAlias;      // LPWSTR
        public Pointer UserName;         // LPWSTR

        public CREDENTIALW() {
            super();
        }

        public CREDENTIALW(Pointer ptr) {
            super(ptr);
            read();
        }
    }

    // --- Public API ---

    @Override
    public String getSecret(String service, String account) throws KeySyncException {
        String target = serviceToTarget(service);

        PointerByReference pCredRef = new PointerByReference();
        try {
            boolean ok = Advapi32.INSTANCE.CredReadW(
                    new WString(target), 1 /* CRED_TYPE_GENERIC */, 0, pCredRef);
            if (!ok) {
                throw new KeySyncException(
                        KeySyncError.NOT_FOUND,
                        "secret not found: " + service + "/" + account
                );
            }

            CREDENTIALW cred = new CREDENTIALW(pCredRef.getValue());

            // Read UserName (the account / key name)
            String userName = pointerToWideString(cred.UserName);
            if (userName == null || !userName.equals(account)) {
                throw new KeySyncException(
                        KeySyncError.NOT_FOUND,
                        "account mismatch: expected " + account
                            + ", got " + userName
                );
            }

            // Read CredentialBlob (the password / secret value)
            if (cred.CredentialBlobSize == 0 || cred.CredentialBlob == null) {
                throw new KeySyncException(
                        KeySyncError.NOT_FOUND,
                        "secret not found: " + service + "/" + account
                );
            }

            // The blob is stored as a wide (UTF-16LE) string.
            // CredentialBlobSize is in bytes; divide by 2 for char count.
            int charCount = cred.CredentialBlobSize / 2;
            char[] chars = cred.CredentialBlob.getCharArray(0, charCount);
            return new String(chars);

        } finally {
            Pointer credPtr = pCredRef.getValue();
            if (credPtr != null) {
                Advapi32.INSTANCE.CredFree(credPtr);
            }
        }
    }

    @Override
    public List<Credential> listSecrets() throws KeySyncException {
        IntByReference countRef = new IntByReference();
        PointerByReference credsRef = new PointerByReference();

        try {
            boolean ok = Advapi32.INSTANCE.CredEnumerateW(
                    new WString("keysync_*"), 0, countRef, credsRef);
            if (!ok || countRef.getValue() == 0) {
                return Collections.emptyList();
            }

            int count = countRef.getValue();
            List<Credential> results = new ArrayList<>(count);
            int pointerSize = Native.POINTER_SIZE;

            for (int i = 0; i < count; i++) {
                // The creds pointer points to an array of PCREDENTIALW
                // (pointers to CREDENTIALW structs). Each pointer is
                // Pointer.SIZE bytes.
                long offset = (long) i * pointerSize;
                Pointer credPtr = credsRef.getValue().getPointer(offset);
                if (credPtr == null) {
                    continue;
                }

                CREDENTIALW cred = new CREDENTIALW(credPtr);
                String target = pointerToWideString(cred.TargetName);
                String userName = pointerToWideString(cred.UserName);

                if (target == null || !target.startsWith("keysync_")) {
                    continue;
                }

                String service = targetToService(target);
                results.add(new Credential(service, userName != null ? userName : ""));
            }

            return results;

        } finally {
            Pointer credsPtr = credsRef.getValue();
            if (credsPtr != null) {
                Advapi32.INSTANCE.CredFree(credsPtr);
            }
        }
    }

    // --- Service/target name conversion ---

    /**
     * Converts a keysync service name to a Windows Credential Manager target
     * name by replacing slashes with underscores.
     *
     * <pre>
     * "keysync/global"      → "keysync_global"
     * "keysync/project/app" → "keysync_project_app"
     * </pre>
     */
    static String serviceToTarget(String service) {
        return service.replace('/', '_');
    }

    /**
     * Converts a Windows target name back to a keysync service name.
     *
     * <pre>
     * "keysync_global"             → "keysync/global"
     * "keysync_project_myapp"      → "keysync/project/myapp"
     * "keysync_project_my_deep"    → "keysync/project/my/deep"
     * </pre>
     *
     * The first underscore after "keysync_" separates the scope from the
     * project. For "global" there is no project part. For "project" the
     * remaining underscores become slashes.
     */
    static String targetToService(String target) {
        // Strip the "keysync_" prefix
        String remainder = target.substring("keysync_".length());
        int firstUnderscore = remainder.indexOf('_');

        if (firstUnderscore < 0) {
            // Just scope, e.g. "global"
            return "keysync/" + remainder;
        }

        String scope = remainder.substring(0, firstUnderscore);
        String projectPart = remainder.substring(firstUnderscore + 1);

        if ("global".equals(scope)) {
            return "keysync/global";
        }

        // project scope: replace remaining underscores with slashes
        return "keysync/" + scope + "/" + projectPart.replace('_', '/');
    }

    // --- helpers ---

    /**
     * Reads a null-terminated wide (UTF-16LE) string from a native pointer.
     */
    private static String pointerToWideString(Pointer ptr) {
        if (ptr == null) {
            return null;
        }
        // Walk until we hit a null wchar (two zero bytes)
        int maxLen = 4096; // safety limit
        int len = 0;
        while (len < maxLen && ptr.getShort((long) len * 2) != 0) {
            len++;
        }
        if (len == 0) {
            return "";
        }
        char[] chars = ptr.getCharArray(0, len);
        return new String(chars);
    }
}
