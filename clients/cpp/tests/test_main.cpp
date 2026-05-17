#include <cstdlib>
#include <cstring>
#include <iostream>
#include <string>
#include <vector>

#include "keysync/keysync.hpp"
#include "keysync/errors.hpp"
#include "keysync/credential.hpp"

// Simple assertion macro – no external test framework needed.
static int g_failures = 0;

#define ASSERT_TRUE(cond) \
    do { \
        if (!(cond)) { \
            std::cerr << "FAIL [" << __LINE__ << "]: expected true, got false" << std::endl; \
            ++g_failures; \
        } \
    } while (0)

#define ASSERT_EQUAL(a, b) \
    do { \
        if ((a) != (b)) { \
            std::cerr << "FAIL [" << __LINE__ << "]: expected " << (b) << ", got " << (a) << std::endl; \
            ++g_failures; \
        } \
    } while (0)

#define ASSERT_THROWS_CODE(expr, code) \
    do { \
        bool caught = false; \
        try { \
            expr; \
        } catch (const keysync::KeySyncError& e) { \
            caught = true; \
            if (e.code() != code) { \
                std::cerr << "FAIL [" << __LINE__ << "]: expected error code " \
                          << static_cast<int>(code) << ", got " \
                          << static_cast<int>(e.code()) << " (" << e.what() << ")" << std::endl; \
                ++g_failures; \
            } \
        } catch (const std::exception& e) { \
            std::cerr << "FAIL [" << __LINE__ << "]: expected KeySyncError, got " \
                      << typeid(e).name() << ": " << e.what() << std::endl; \
            ++g_failures; \
        } \
        if (!caught) { \
            std::cerr << "FAIL [" << __LINE__ << "]: expected KeySyncError but no exception was thrown" << std::endl; \
            ++g_failures; \
        } \
    } while (0)

// ---------------------------------------------------------------------------
// Error type tests
// ---------------------------------------------------------------------------

static void test_error_not_found() {
    keysync::KeySyncError err(keysync::ErrorCode::NotFound, "secret not found: API_KEY");
    ASSERT_EQUAL(static_cast<int>(err.code()), static_cast<int>(keysync::ErrorCode::NotFound));
    ASSERT_TRUE(std::string(err.what()).find("secret not found") != std::string::npos);
    std::cout << "  PASS: test_error_not_found" << std::endl;
}

static void test_error_keychain_error() {
    keysync::KeySyncError err(keysync::ErrorCode::KeychainError, "something broke");
    ASSERT_EQUAL(static_cast<int>(err.code()), static_cast<int>(keysync::ErrorCode::KeychainError));
    ASSERT_TRUE(std::string(err.what()).find("something broke") != std::string::npos);
    std::cout << "  PASS: test_error_keychain_error" << std::endl;
}

static void test_error_unsupported_platform() {
    keysync::KeySyncError err(keysync::ErrorCode::UnsupportedPlatform, "bad platform");
    ASSERT_EQUAL(static_cast<int>(err.code()), static_cast<int>(keysync::ErrorCode::UnsupportedPlatform));
    std::cout << "  PASS: test_error_unsupported_platform" << std::endl;
}

// ---------------------------------------------------------------------------
// Env var fallback test
// ---------------------------------------------------------------------------

static void test_env_var_fallback() {
    // Set an environment variable
    setenv("KEYSYNC_TEST_VAR", "env_value", 1);

    // getSecret should pick up the env var first
    std::string result = keysync::getSecret("KEYSYNC_TEST_VAR");
    ASSERT_EQUAL(result, std::string("env_value"));

    // Clean up
    unsetenv("KEYSYNC_TEST_VAR");
    std::cout << "  PASS: test_env_var_fallback" << std::endl;
}

static void test_env_var_not_set_triggers_keychain() {
    // For a key that doesn't exist as an env var and likely doesn't exist
    // in the keychain, we expect a NotFound error.
    // This tests that the env var check succeeds and the code proceeds to
    // the keychain fallback.
    ASSERT_THROWS_CODE(
        keysync::getSecret("KEYSYNC_NONEXISTENT_XYZ_999"),
        keysync::ErrorCode::NotFound
    );
    std::cout << "  PASS: test_env_var_not_set_triggers_keychain" << std::endl;
}

// ---------------------------------------------------------------------------
// CredentialEntry tests
// ---------------------------------------------------------------------------

static void test_credential_entry() {
    keysync::CredentialEntry entry{"global", "", "API_KEY"};
    ASSERT_EQUAL(entry.scope, std::string("global"));
    ASSERT_EQUAL(entry.project, std::string(""));
    ASSERT_EQUAL(entry.key, std::string("API_KEY"));

    keysync::CredentialEntry entry2{"project", "myapp", "DATABASE_URL"};
    ASSERT_EQUAL(entry2.scope, std::string("project"));
    ASSERT_EQUAL(entry2.project, std::string("myapp"));
    ASSERT_EQUAL(entry2.key, std::string("DATABASE_URL"));

    // Equality
    ASSERT_TRUE(entry == entry);
    ASSERT_TRUE(entry != entry2);

    std::cout << "  PASS: test_credential_entry" << std::endl;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

int main() {
    std::cout << "Running keysync C++ client tests..." << std::endl;
    std::cout << std::endl;

    test_error_not_found();
    test_error_keychain_error();
    test_error_unsupported_platform();
    test_env_var_fallback();
    test_env_var_not_set_triggers_keychain();
    test_credential_entry();
    // Tests from test_service.cpp are linked together

    std::cout << std::endl;
    extern int test_service_names();
    int svcFailures = test_service_names();
    g_failures += svcFailures;

    std::cout << std::endl;
    if (g_failures == 0) {
        std::cout << "All tests passed." << std::endl;
        return 0;
    } else {
        std::cout << g_failures << " test(s) FAILED." << std::endl;
        return 1;
    }
}
