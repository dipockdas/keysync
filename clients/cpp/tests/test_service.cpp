#include "internal_helpers.hpp"

#include <cstdlib>
#include <iostream>
#include <string>

using namespace keysync::internal;

// Simple assertion macro
static int g_failures = 0;

#define ASSERT_EQUAL(a, b) \
    do { \
        if ((a) != (b)) { \
            std::cerr << "FAIL [" << __LINE__ << "]: expected " << (b) << ", got " << (a) << std::endl; \
            ++g_failures; \
        } \
    } while (0)

#define ASSERT_TRUE(cond) \
    do { \
        if (!(cond)) { \
            std::cerr << "FAIL [" << __LINE__ << "]: expected true" << std::endl; \
            ++g_failures; \
        } \
    } while (0)

// ---------------------------------------------------------------------------
// Service name construction tests
// ---------------------------------------------------------------------------

static void test_service_name_global() {
    std::string svc = serviceName("global");
    ASSERT_EQUAL(svc, std::string("keysync/global"));
    std::cout << "  PASS: test_service_name_global" << std::endl;
}

static void test_service_name_project() {
    std::string svc = serviceName("project", "my-app");
    ASSERT_EQUAL(svc, std::string("keysync/project/my-app"));
    std::cout << "  PASS: test_service_name_project" << std::endl;
}

static void test_service_name_global_ignores_project() {
    std::string svc = serviceName("global", "my-app");
    ASSERT_EQUAL(svc, std::string("keysync/global"));
    std::cout << "  PASS: test_service_name_global_ignores_project" << std::endl;
}

static void test_service_name_project_empty_name() {
    std::string svc = serviceName("project", "");
    ASSERT_EQUAL(svc, std::string("keysync/project"));
    std::cout << "  PASS: test_service_name_project_empty_name" << std::endl;
}

static void test_service_name_environment() {
    std::string svc = serviceName("project", "my-app", "staging");
    ASSERT_EQUAL(svc, std::string("keysync/project/my-app/env/staging"));
    std::cout << "  PASS: test_service_name_environment" << std::endl;
}

static void test_service_name_environment_global_ignores_env() {
    std::string svc = serviceName("global", "", "staging");
    ASSERT_EQUAL(svc, std::string("keysync/global"));
    std::cout << "  PASS: test_service_name_environment_global_ignores_env" << std::endl;
}

// ---------------------------------------------------------------------------
// Service name parsing tests
// ---------------------------------------------------------------------------

static void test_parse_global() {
    std::string scope, project, environment;
    parseServiceName("keysync/global", scope, project, environment);
    ASSERT_EQUAL(scope, std::string("global"));
    ASSERT_EQUAL(project, std::string(""));
    ASSERT_EQUAL(environment, std::string(""));
    std::cout << "  PASS: test_parse_global" << std::endl;
}

static void test_parse_project() {
    std::string scope, project, environment;
    parseServiceName("keysync/project/my-app", scope, project, environment);
    ASSERT_EQUAL(scope, std::string("project"));
    ASSERT_EQUAL(project, std::string("my-app"));
    ASSERT_EQUAL(environment, std::string(""));
    std::cout << "  PASS: test_parse_project" << std::endl;
}

static void test_parse_environment() {
    std::string scope, project, environment;
    parseServiceName("keysync/project/my-app/env/staging",
                     scope, project, environment);
    ASSERT_EQUAL(scope, std::string("project"));
    ASSERT_EQUAL(project, std::string("my-app"));
    ASSERT_EQUAL(environment, std::string("staging"));
    std::cout << "  PASS: test_parse_environment" << std::endl;
}

static void test_parse_project_deep_path() {
    std::string scope, project, environment;
    parseServiceName("keysync/project/my/deep/app", scope, project, environment);
    ASSERT_EQUAL(scope, std::string("project"));
    ASSERT_EQUAL(project, std::string("my/deep/app"));
    ASSERT_EQUAL(environment, std::string(""));
    std::cout << "  PASS: test_parse_project_deep_path" << std::endl;
}

static void test_parse_unprefixed() {
    std::string scope, project, environment;
    parseServiceName("other/global", scope, project, environment);
    ASSERT_EQUAL(scope, std::string("other"));
    std::cout << "  PASS: test_parse_unprefixed" << std::endl;
}

static void test_parse_empty() {
    std::string scope, project, environment;
    parseServiceName("", scope, project, environment);
    ASSERT_EQUAL(scope, std::string(""));
    ASSERT_EQUAL(project, std::string(""));
    ASSERT_EQUAL(environment, std::string(""));
    std::cout << "  PASS: test_parse_empty" << std::endl;
}

// ---------------------------------------------------------------------------
// Windows target name conversion tests (always testable)
// ---------------------------------------------------------------------------

static void test_service_to_target_global() {
    std::string target = serviceToTarget("keysync/global");
    ASSERT_EQUAL(target, std::string("keysync_global"));
    std::cout << "  PASS: test_service_to_target_global" << std::endl;
}

static void test_service_to_target_project() {
    std::string target = serviceToTarget("keysync/project/my-app");
    ASSERT_EQUAL(target, std::string("keysync_project_my-app"));
    std::cout << "  PASS: test_service_to_target_project" << std::endl;
}

static void test_service_to_target_multi_slash() {
    std::string target = serviceToTarget("keysync/project/a/b/c");
    ASSERT_EQUAL(target, std::string("keysync_project_a_b_c"));
    std::cout << "  PASS: test_service_to_target_multi_slash" << std::endl;
}

static void test_service_to_target_environment() {
    std::string target = serviceToTarget("keysync/project/my-app/env/dev");
    ASSERT_EQUAL(target, std::string("keysync_project_my-app_dev"));
    std::cout << "  PASS: test_service_to_target_environment" << std::endl;
}

static void test_target_to_service_global() {
    std::string svc = targetToService("keysync_global");
    ASSERT_EQUAL(svc, std::string("keysync/global"));
    std::cout << "  PASS: test_target_to_service_global" << std::endl;
}

static void test_target_to_service_project() {
    std::string svc = targetToService("keysync_project_my-app");
    // Two segments: just project
    ASSERT_EQUAL(svc, std::string("keysync/project/my-app"));
    std::cout << "  PASS: test_target_to_service_project" << std::endl;
}

static void test_target_to_service_environment() {
    std::string svc = targetToService("keysync_project_my-app_dev");
    // Three segments: project + env
    ASSERT_EQUAL(svc, std::string("keysync/project/my-app/env/dev"));
    std::cout << "  PASS: test_target_to_service_environment" << std::endl;
}

static void test_roundtrip_global() {
    std::string svc = "keysync/global";
    ASSERT_EQUAL(targetToService(serviceToTarget(svc)), svc);
    std::cout << "  PASS: test_roundtrip_global" << std::endl;
}

static void test_roundtrip_environment() {
    std::string svc = "keysync/project/my-app/env/staging";
    ASSERT_EQUAL(targetToService(serviceToTarget(svc)), svc);
    std::cout << "  PASS: test_roundtrip_environment" << std::endl;
}

// ---------------------------------------------------------------------------
// Trim tests
// ---------------------------------------------------------------------------

static void test_trim_trailing_newline() {
    std::string result = trimTrailing("hello\n");
    ASSERT_EQUAL(result, std::string("hello"));
    std::cout << "  PASS: test_trim_trailing_newline" << std::endl;
}

static void test_trim_trailing_spaces() {
    std::string result = trimTrailing("hello   ");
    ASSERT_EQUAL(result, std::string("hello"));
    std::cout << "  PASS: test_trim_trailing_spaces" << std::endl;
}

static void test_trim_trailing_empty() {
    std::string result = trimTrailing("   ");
    ASSERT_TRUE(result.empty());
    std::cout << "  PASS: test_trim_trailing_empty" << std::endl;
}

static void test_trim_leading_trailing() {
    std::string result = trim("  hello  ");
    ASSERT_EQUAL(result, std::string("hello"));
    std::cout << "  PASS: test_trim_leading_trailing" << std::endl;
}

static void test_trim_empty() {
    std::string result = trim("   ");
    ASSERT_TRUE(result.empty());
    std::cout << "  PASS: test_trim_empty" << std::endl;
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

int test_service_names() {
    std::cout << "Running service name tests..." << std::endl;

    test_service_name_global();
    test_service_name_project();
    test_service_name_global_ignores_project();
    test_service_name_project_empty_name();
    test_service_name_environment();
    test_service_name_environment_global_ignores_env();

    test_parse_global();
    test_parse_project();
    test_parse_environment();
    test_parse_project_deep_path();
    test_parse_unprefixed();
    test_parse_empty();

    test_service_to_target_global();
    test_service_to_target_project();
    test_service_to_target_multi_slash();
    test_service_to_target_environment();
    test_target_to_service_global();
    test_target_to_service_project();
    test_target_to_service_environment();
    test_roundtrip_global();
    test_roundtrip_environment();

    test_trim_trailing_newline();
    test_trim_trailing_spaces();
    test_trim_trailing_empty();
    test_trim_leading_trailing();
    test_trim_empty();

    if (g_failures == 0) {
        std::cout << "  All service name tests passed." << std::endl;
    }

    return g_failures;
}
