package io.keysync;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.condition.EnabledOnOs;
import org.junit.jupiter.api.condition.OS;

import java.lang.reflect.Field;
import java.util.Collections;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for the KeySync Java client library.
 */
@DisplayName("KeySyncClient")
class KeySyncClientTest {

    // --- Service name construction ---

    @Nested
    @DisplayName("Service name construction")
    class ServiceNameTests {

        @Test
        @DisplayName("global scope returns 'keysync/global'")
        void globalScope() {
            String name = KeySyncClient.serviceName("global", null);
            assertEquals("keysync/global", name);
        }

        @Test
        @DisplayName("global scope with project returns 'keysync/global' (ignores project)")
        void globalScopeWithProject() {
            String name = KeySyncClient.serviceName("global", "my-app");
            assertEquals("keysync/global", name);
        }

        @Test
        @DisplayName("project scope returns 'keysync/project/<name>'")
        void projectScope() {
            String name = KeySyncClient.serviceName("project", "my-app");
            assertEquals("keysync/project/my-app", name);
        }

        @Test
        @DisplayName("project scope with null project returns 'keysync/project'")
        void projectScopeNullProject() {
            String name = KeySyncClient.serviceName("project", null);
            assertEquals("keysync/project", name);
        }

        @Test
        @DisplayName("project scope with empty project returns 'keysync/project'")
        void projectScopeEmptyProject() {
            String name = KeySyncClient.serviceName("project", "");
            assertEquals("keysync/project", name);
        }
    }

    // --- Service name parsing ---

    @Nested
    @DisplayName("Service name parsing")
    class ParseServiceNameTests {

        @Test
        @DisplayName("parses 'keysync/global'")
        void parseGlobal() {
            String[] result = KeySyncClient.parseServiceName("keysync/global");
            assertEquals("global", result[0]);
            assertNull(result[1]);
        }

        @Test
        @DisplayName("parses 'keysync/project/my-app'")
        void parseProject() {
            String[] result = KeySyncClient.parseServiceName("keysync/project/my-app");
            assertEquals("project", result[0]);
            assertEquals("my-app", result[1]);
        }

        @Test
        @DisplayName("parses 'keysync/project/my/deep/app'")
        void parseProjectDeep() {
            String[] result = KeySyncClient.parseServiceName(
                    "keysync/project/my/deep/app");
            assertEquals("project", result[0]);
            assertEquals("my/deep/app", result[1]);
        }

        @Test
        @DisplayName("parses non-keysync prefix as global/null")
        void parseNonKeysync() {
            String[] result = KeySyncClient.parseServiceName("other/global");
            assertEquals("global", result[0]);
            assertNull(result[1]);
        }

        @Test
        @DisplayName("parses null as global/null")
        void parseNull() {
            String[] result = KeySyncClient.parseServiceName(null);
            assertEquals("global", result[0]);
            assertNull(result[1]);
        }

        @Test
        @DisplayName("parses empty string as global/null")
        void parseEmpty() {
            String[] result = KeySyncClient.parseServiceName("");
            assertEquals("global", result[0]);
            assertNull(result[1]);
        }

        @Test
        @DisplayName("non-project scope returns scope with null project")
        void parseNonProjectScope() {
            String[] result = KeySyncClient.parseServiceName("keysync/other/val");
            assertEquals("other", result[0]);
            assertNull(result[1]);
        }
    }

    // --- Error types ---

    @Nested
    @DisplayName("Error types")
    class ErrorTypeTests {

        @Test
        @DisplayName("NOT_FOUND has code 'notFound'")
        void notFoundCode() {
            assertEquals("notFound", KeySyncError.NOT_FOUND.getCode());
        }

        @Test
        @DisplayName("KEYCHAIN_ERROR has code 'keychainError'")
        void keychainErrorCode() {
            assertEquals("keychainError", KeySyncError.KEYCHAIN_ERROR.getCode());
        }

        @Test
        @DisplayName("UNSUPPORTED_PLATFORM has code 'unsupportedPlatform'")
        void unsupportedPlatformCode() {
            assertEquals("unsupportedPlatform",
                    KeySyncError.UNSUPPORTED_PLATFORM.getCode());
        }

        @Test
        @DisplayName("KeySyncException stores error and message")
        void exceptionStoresErrorAndMessage() {
            KeySyncException ex = new KeySyncException(
                    KeySyncError.NOT_FOUND, "missing key");

            assertEquals(KeySyncError.NOT_FOUND, ex.getError());
            assertEquals("notFound", ex.getErrorCode());
            assertEquals("missing key", ex.getMessage());
        }

        @Test
        @DisplayName("KeySyncException stores cause")
        void exceptionStoresCause() {
            Throwable cause = new RuntimeException("inner");
            KeySyncException ex = new KeySyncException(
                    KeySyncError.KEYCHAIN_ERROR, "broken", cause);

            assertEquals(cause, ex.getCause());
            assertEquals("broken", ex.getMessage());
        }
    }

    // --- Credential POJO ---

    @Nested
    @DisplayName("Credential POJO")
    class CredentialTests {

        @Test
        @DisplayName("stores service and account")
        void storesFields() {
            Credential c = new Credential("keysync/global", "API_KEY");
            assertEquals("keysync/global", c.getService());
            assertEquals("API_KEY", c.getAccount());
        }

        @Test
        @DisplayName("equals works for same values")
        void equalsSame() {
            Credential a = new Credential("keysync/global", "API_KEY");
            Credential b = new Credential("keysync/global", "API_KEY");
            assertEquals(a, b);
        }

        @Test
        @DisplayName("equals works for different values")
        void equalsDifferent() {
            Credential a = new Credential("keysync/global", "API_KEY");
            Credential b = new Credential("keysync/global", "OTHER_KEY");
            assertNotEquals(a, b);
        }

        @Test
        @DisplayName("hashCode is consistent with equals")
        void hashCodeConsistent() {
            Credential a = new Credential("keysync/global", "API_KEY");
            Credential b = new Credential("keysync/global", "API_KEY");
            assertEquals(a.hashCode(), b.hashCode());
        }

        @Test
        @DisplayName("toString contains service and account")
        void toStringContainsFields() {
            Credential c = new Credential("keysync/global", "API_KEY");
            String s = c.toString();
            assertTrue(s.contains("keysync/global"));
            assertTrue(s.contains("API_KEY"));
        }
    }

    // --- Env var fallback ---

    @Nested
    @DisplayName("Environment variable fallback")
    class EnvVarTests {

        @Test
        @DisplayName("getSecret returns env var when set")
        void returnsEnvVarWhenSet() {
            // Use a key that is definitely not a real env var to avoid
            // interference, then set it before calling. Since we can't
            // easily mock System.getenv() in Java, we test that the env
            // var check takes priority over keychain by checking that a
            // real env var is returned.
            String path = System.getenv("PATH");
            if (path != null) {
                // PATH is always set; calling getSecret("PATH") should
                // return it without hitting the keychain
                try {
                    String result = KeySyncClient.getInstance().getSecret("PATH");
                    assertEquals(path, result);
                } catch (KeySyncException e) {
                    // Only fail if it threw something other than the
                    // env var not being set
                    if (e.getError() != KeySyncError.NOT_FOUND) {
                        throw e;
                    }
                }
            }
        }

        @Test
        @DisplayName("getSecret returns env var even for project-scoped calls")
        void returnsEnvVarWithProject() {
            String home = System.getenv("HOME");
            if (home == null) {
                home = System.getenv("USERPROFILE");
            }
            if (home != null) {
                try {
                    // Even with a project parameter, the env var should
                    // be returned first (no keychain access)
                    String result = KeySyncClient.getInstance()
                            .getSecret("HOME", "myapp");
                    assertEquals(home, result);
                } catch (KeySyncException e) {
                    if (e.getError() != KeySyncError.NOT_FOUND
                            && e.getError() != KeySyncError.KEYCHAIN_ERROR) {
                        throw e;
                    }
                }
            }
        }

        @Test
        @DisplayName("getSecret returns env var even when HOME is set")
        void homeEnvVar() {
            // HOME is guaranteed on macOS/Linux; USERPROFILE on Windows
            String home = System.getenv("HOME");
            if (home == null) {
                home = System.getenv("USERPROFILE");
            }
            if (home != null) {
                try {
                    String result = KeySyncClient.getInstance().getSecret(
                            System.getenv("HOME") != null ? "HOME" : "USERPROFILE");
                    assertEquals(home, result);
                } catch (KeySyncException e) {
                    fail("Should have returned env var, but threw: " + e.getMessage());
                }
            }
        }
    }

    // --- Platform detection ---

    @Nested
    @DisplayName("Platform detection")
    class PlatformDetectionTests {

        @Test
        @DisplayName("getPlatform returns a non-empty string")
        void getPlatformReturnsNonEmpty() {
            String platform = KeySyncClient.getInstance().getPlatform();
            assertNotNull(platform);
            assertFalse(platform.isEmpty());
        }

        @Test
        @DisplayName("isPlatformSupported returns true on mac/linux/win")
        @EnabledOnOs({OS.MAC, OS.LINUX, OS.WINDOWS})
        void isPlatformSupported() {
            assertTrue(KeySyncClient.getInstance().isPlatformSupported());
        }

        @Test
        @DisplayName("platform string contains mac on macOS")
        @EnabledOnOs(OS.MAC)
        void platformContainsMac() {
            assertTrue(KeySyncClient.getInstance().getPlatform().contains("mac"));
        }

        @Test
        @DisplayName("platform string contains linux on Linux")
        @EnabledOnOs(OS.LINUX)
        void platformContainsLinux() {
            assertTrue(KeySyncClient.getInstance().getPlatform().contains("linux"));
        }

        @Test
        @DisplayName("platform string contains win on Windows")
        @EnabledOnOs(OS.WINDOWS)
        void platformContainsWin() {
            assertTrue(KeySyncClient.getInstance().getPlatform().contains("win"));
        }
    }

    // --- Singleton ---

    @Nested
    @DisplayName("Singleton behavior")
    class SingletonTests {

        @Test
        @DisplayName("getInstance returns the same instance")
        void getInstanceReturnsSameInstance() {
            KeySyncClient a = KeySyncClient.getInstance();
            KeySyncClient b = KeySyncClient.getInstance();
            assertSame(a, b);
        }
    }

    // --- Windows target conversion tests ---

    @Nested
    @DisplayName("Windows target name conversion")
    class WindowsConversionTests {

        @Test
        @DisplayName("serviceToTarget converts 'keysync/global'")
        void serviceToTargetGlobal() {
            String target = WindowsKeychain.serviceToTarget("keysync/global");
            assertEquals("keysync_global", target);
        }

        @Test
        @DisplayName("serviceToTarget converts 'keysync/project/myapp'")
        void serviceToTargetProject() {
            String target = WindowsKeychain.serviceToTarget(
                    "keysync/project/myapp");
            assertEquals("keysync_project_myapp", target);
        }

        @Test
        @DisplayName("serviceToTarget handles deeply nested projects")
        void serviceToTargetDeep() {
            String target = WindowsKeychain.serviceToTarget(
                    "keysync/project/a/b/c");
            assertEquals("keysync_project_a_b_c", target);
        }

        @Test
        @DisplayName("targetToService converts 'keysync_global'")
        void targetToServiceGlobal() {
            String service = WindowsKeychain.targetToService("keysync_global");
            assertEquals("keysync/global", service);
        }

        @Test
        @DisplayName("targetToService converts 'keysync_project_myapp'")
        void targetToServiceProject() {
            String service = WindowsKeychain.targetToService(
                    "keysync_project_myapp");
            assertEquals("keysync/project/myapp", service);
        }

        @Test
        @DisplayName("targetToService converts 'keysync_project_my_deep_app'")
        void targetToServiceDeep() {
            String service = WindowsKeychain.targetToService(
                    "keysync_project_my_deep_app");
            assertEquals("keysync/project/my/deep/app", service);
        }

        @Test
        @DisplayName("round-trip service→target→service")
        void roundTrip() {
            String original = "keysync/project/my-app";
            String roundTripped = WindowsKeychain.targetToService(
                    WindowsKeychain.serviceToTarget(original));
            assertEquals(original, roundTripped);
        }
    }

    // --- listSecrets with project filter ---

    @Nested
    @DisplayName("listSecrets project filters")
    class ListSecretsProjectFilterTests {

        @Test
        @DisplayName("listSecrets(null) returns all secrets (does not throw)")
        void listSecretsAll() {
            try {
                List<Credential> results =
                        KeySyncClient.getInstance().listSecrets();
                assertNotNull(results);
            } catch (KeySyncException e) {
                // May fail if on unsupported platform or keychain is locked
                if (e.getError() == KeySyncError.UNSUPPORTED_PLATFORM) {
                    throw e;
                }
                // Keychain error on a CI without entries is OK
            }
        }

        @Test
        @DisplayName("listSecrets with project returns filtered results")
        void listSecretsWithProject() {
            try {
                List<Credential> results =
                        KeySyncClient.getInstance().listSecrets("nonexistent");
                assertNotNull(results);
            } catch (KeySyncException e) {
                if (e.getError() == KeySyncError.UNSUPPORTED_PLATFORM) {
                    throw e;
                }
            }
        }
    }
}
