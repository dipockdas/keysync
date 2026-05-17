#ifndef KEYSYNC_KEYSYNC_HPP
#define KEYSYNC_KEYSYNC_HPP

/// @file keysync.hpp
/// @brief Public API for reading secrets from the OS-native keychain.
///
/// Usage:
/// @code{.cpp}
/// #include <keysync/keysync.hpp>
///
/// // Get a global secret
/// std::string apiKey = keysync::getSecret("API_KEY");
///
/// // Get a project-scoped secret (falls back to global)
/// std::string dbUrl = keysync::getSecret("DATABASE_URL", "myapp");
///
/// // List all global secrets
/// auto globals = keysync::listSecrets();
///
/// // List project secrets (includes global fallback)
/// auto project = keysync::listSecrets("myapp");
/// @endcode

#include <string>
#include <string_view>
#include <vector>

#include "credential.hpp"
#include "errors.hpp"

namespace keysync {

/// Retrieve a secret from the OS keychain.
///
/// Checks the environment variable identified by `key` first. If set, returns
/// it immediately without touching the OS keychain. This is the primary path
/// for both local development (where secrets are injected via
/// `eval $(keysync export)`) and cloud deployments (where platforms inject
/// environment variables directly).
///
/// If the env var is not set, falls back to the OS keychain. When `project` is
/// provided, checks project scope first, then global scope.
///
/// Service naming:
///   Global:  "keysync/global"
///   Project: "keysync/project/<name>"
///
/// @param key The secret key name (e.g. "DATABASE_URL").
/// @param project Optional project name for project-scoped secrets.
/// @return The secret value as a string.
/// @throws KeySyncError with ErrorCode::NotFound if the secret doesn't exist
///         in any scope.
/// @throws KeySyncError with ErrorCode::KeychainError on OS-level failures.
/// @throws KeySyncError with ErrorCode::UnsupportedPlatform if the platform
///         is not supported.
std::string getSecret(std::string_view key, std::string_view project = "");

/// List all stored secrets matching the given scope and/or project.
///
/// @param project Optional project filter. If empty, lists global secrets only.
/// @return A vector of CredentialEntry structs.
/// @throws KeySyncError on platform failures.
std::vector<CredentialEntry> listSecrets(std::string_view project = "");

} // namespace keysync

#endif // KEYSYNC_KEYSYNC_HPP
