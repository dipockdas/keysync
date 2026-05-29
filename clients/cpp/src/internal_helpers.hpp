#include <string>
#include <string_view>

namespace keysync {
namespace internal {

/// Build the keychain service name.
///
/// Global:  "keysync/global"
/// Project: "keysync/project/<name>"
std::string serviceName(std::string_view scope, std::string_view project = "");

/// Build the keychain service name with optional environment.
///
/// Environment: "keysync/project/<name>/env/<env>"
std::string serviceName(
    std::string_view scope,
    std::string_view project,
    std::string_view environment);

/// Parse a service name back into scope, project, and environment.
///
/// "keysync/global"                     → ("global", "", "")
/// "keysync/project/my-app"             → ("project", "my-app", "")
/// "keysync/project/my-app/env/staging" → ("project", "my-app", "staging")
/// "keysync/project/a/b"                → ("project", "a/b", "")
void parseServiceName(
    std::string_view service,
    std::string& scope,
    std::string& project,
    std::string& environment);

/// Convert a service name to a Windows Credential Manager target.
/// Strips /env/ keyword and replaces slashes with underscores.
/// "keysync/global"                    → "keysync_global"
/// "keysync/project/my-app"            → "keysync_project_my-app"
/// "keysync/project/my-app/env/dev"    → "keysync_project_my-app_dev"
std::string serviceToTarget(std::string_view service);

/// Convert a Windows target back to a service name, inserting /env/
/// between project and environment segments.
/// "keysync_global"                       → "keysync/global"
/// "keysync_project_my-app"               → "keysync/project/my-app"
/// "keysync_project_my-app_dev"           → "keysync/project/my-app/env/dev"
std::string targetToService(std::string_view target);

/// Trim trailing whitespace from a string (returns new string).
std::string trimTrailing(std::string_view s);

/// Trim leading and trailing whitespace.
std::string trim(std::string_view s);

} // namespace internal
} // namespace keysync
