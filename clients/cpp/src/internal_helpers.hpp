#include <string>
#include <string_view>

namespace keysync {
namespace internal {

/// Build the keychain service name.
///
/// Global:  "keysync/global"
/// Project: "keysync/project/<name>"
std::string serviceName(std::string_view scope, std::string_view project = "");

/// Parse a service name back into scope and project.
///
/// "keysync/global"         → ("global", "")
/// "keysync/project/my-app" → ("project", "my-app")
/// "keysync/project/a/b"    → ("project", "a/b")
void parseServiceName(
    std::string_view service,
    std::string& scope,
    std::string& project);

/// Convert a service name to a Windows Credential Manager target.
/// "keysync/global"         → "keysync_global"
/// "keysync/project/my-app" → "keysync_project_my-app"
std::string serviceToTarget(std::string_view service);

/// Convert a Windows target back to a service name.
/// "keysync_global"          → "keysync/global"
/// "keysync_project_my-app"  → "keysync/project/my-app"
std::string targetToService(std::string_view target);

/// Trim trailing whitespace from a string (returns new string).
std::string trimTrailing(std::string_view s);

/// Trim leading and trailing whitespace.
std::string trim(std::string_view s);

} // namespace internal
} // namespace keysync
