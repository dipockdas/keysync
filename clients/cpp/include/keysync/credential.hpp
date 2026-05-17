#ifndef KEYSYNC_CREDENTIAL_HPP
#define KEYSYNC_CREDENTIAL_HPP

#include <string>

namespace keysync {

/// Represents a single secret entry with its scope, project, and key.
struct CredentialEntry {
    std::string scope;    // "global" or "project"
    std::string project;  // Project name (empty for global scope)
    std::string key;      // The secret key name (e.g. "DATABASE_URL")

    bool operator==(const CredentialEntry& other) const {
        return scope == other.scope && project == other.project && key == other.key;
    }

    bool operator!=(const CredentialEntry& other) const {
        return !(*this == other);
    }
};

} // namespace keysync

#endif // KEYSYNC_CREDENTIAL_HPP
