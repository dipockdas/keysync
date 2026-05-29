#ifndef KEYSYNC_CREDENTIAL_HPP
#define KEYSYNC_CREDENTIAL_HPP

#include <string>

namespace keysync {

/// Represents a single secret entry with its scope, project, environment, and key.
struct CredentialEntry {
    std::string scope;        // "global" or "project"
    std::string project;      // Project name (empty for global scope)
    std::string environment;  // Environment name (empty if not environment-scoped)
    std::string key;          // The secret key name (e.g. "DATABASE_URL")

    bool operator==(const CredentialEntry& other) const {
        return scope == other.scope
            && project == other.project
            && environment == other.environment
            && key == other.key;
    }

    bool operator!=(const CredentialEntry& other) const {
        return !(*this == other);
    }
};

} // namespace keysync

#endif // KEYSYNC_CREDENTIAL_HPP
