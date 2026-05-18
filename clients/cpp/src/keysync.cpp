#include "internal_helpers.hpp"
#include "keysync/keysync.hpp"
#include "keysync/errors.hpp"
#include "keysync/credential.hpp"

#include <cstdlib>
#include <string>
#include <string_view>
#include <vector>

// ---------------------------------------------------------------------------
// Platform-specific function declarations (defined in per-platform .cpp files)
// ---------------------------------------------------------------------------

namespace keysync {
namespace internal {

#if defined(__APPLE__)
std::string getSecretMacOS(const std::string& service, const std::string& account);
std::vector<CredentialEntry> listSecretsMacOS(const std::string& scope_filter,
                                               const std::string& project_filter,
                                               const std::string& environment_filter);
bool isNotFoundMacOS(const std::string& service, const std::string& account);
#elif defined(__linux__)
std::string getSecretLinux(const std::string& service, const std::string& account);
std::vector<CredentialEntry> listSecretsLinux(const std::string& scope_filter,
                                                const std::string& project_filter,
                                                const std::string& environment_filter);
bool isNotFoundLinux(const std::string& service, const std::string& account);
#elif defined(_WIN32)
std::string getSecretWindows(const std::string& service, const std::string& account);
std::vector<CredentialEntry> listSecretsWindows(const std::string& scope_filter,
                                                  const std::string& project_filter,
                                                  const std::string& environment_filter);
bool isNotFoundWindows(const std::string& service, const std::string& account);
#endif

} // namespace internal

// ---------------------------------------------------------------------------
// Public API implementation
// ---------------------------------------------------------------------------

std::string getSecret(std::string_view key, std::string_view project,
                      std::string_view environment) {
#if !defined(__APPLE__) && !defined(__linux__) && !defined(_WIN32)
    throw KeySyncError(ErrorCode::UnsupportedPlatform,
        "keychain access not available on this platform");
#else
    // Primary path: check environment variable first.
    // In local dev the user runs `eval $(keysync export)` at shell startup;
    // in cloud/CI the platform injects env vars directly.
    std::string keyStr(key);
    const char* envVal = std::getenv(keyStr.c_str());
    if (envVal != nullptr) {
        return std::string(envVal);
    }

    // If project is provided, check environment scope first, then project scope
    if (!project.empty()) {
        // 1. Try environment-scoped (if environment is provided)
        if (!environment.empty()) {
            std::string svc = internal::serviceName("project", project, environment);
            try {
#if defined(__APPLE__)
                return internal::getSecretMacOS(svc, keyStr);
#elif defined(__linux__)
                return internal::getSecretLinux(svc, keyStr);
#elif defined(_WIN32)
                return internal::getSecretWindows(svc, keyStr);
#endif
            } catch (const KeySyncError& e) {
                if (e.code() != ErrorCode::NotFound) {
                    throw; // Re-throw non-NotFound errors
                }
                // Fall through to project scope
            }
        }

        // 2. Try project scope
        std::string svc = internal::serviceName("project", project);
        try {
#if defined(__APPLE__)
            return internal::getSecretMacOS(svc, keyStr);
#elif defined(__linux__)
            return internal::getSecretLinux(svc, keyStr);
#elif defined(_WIN32)
            return internal::getSecretWindows(svc, keyStr);
#endif
        } catch (const KeySyncError& e) {
            if (e.code() != ErrorCode::NotFound) {
                throw; // Re-throw non-NotFound errors
            }
            // Fall through to global scope
        }
    }

    // 3. Fall back to global scope
    std::string svc = internal::serviceName("global");
#if defined(__APPLE__)
    return internal::getSecretMacOS(svc, keyStr);
#elif defined(__linux__)
    return internal::getSecretLinux(svc, keyStr);
#elif defined(_WIN32)
    return internal::getSecretWindows(svc, keyStr);
#endif
#endif // platform check
}

std::vector<CredentialEntry> listSecrets(std::string_view project,
                                          std::string_view environment) {
#if !defined(__APPLE__) && !defined(__linux__) && !defined(_WIN32)
    throw KeySyncError(ErrorCode::UnsupportedPlatform,
        "keychain access not available on this platform");
#else
    std::string projectStr(project);
    std::string envStr(environment);

#if defined(__APPLE__)
    if (project.empty()) {
        return internal::listSecretsMacOS("global", "", "");
    } else {
        return internal::listSecretsMacOS("project", projectStr, envStr);
    }
#elif defined(__linux__)
    if (project.empty()) {
        return internal::listSecretsLinux("global", "", "");
    } else {
        return internal::listSecretsLinux("project", projectStr, envStr);
    }
#elif defined(_WIN32)
    if (project.empty()) {
        return internal::listSecretsWindows("global", "", "");
    } else {
        return internal::listSecretsWindows("project", projectStr, envStr);
    }
#endif
#endif // platform check
}

} // namespace keysync
