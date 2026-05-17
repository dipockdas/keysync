/// Linux implementation: uses the `secret-tool` CLI (libsecret) via popen().
///
/// Stored entries use libsecret's key-value lookup:
///     service="keysync/global" or "keysync/project/<name>"
///     account=the secret key name (e.g. "DATABASE_URL")

#ifdef __linux__

#include "internal_helpers.hpp"
#include "keysync/errors.hpp"
#include "keysync/credential.hpp"

#include <cstdio>
#include <cstdlib>
#include <string>
#include <vector>

namespace keysync {
namespace internal {
namespace {

/// Execute a shell command via popen() and return stdout.
std::string execRead(const char* cmd) {
    std::string result;
    FILE* pipe = popen(cmd, "r");
    if (!pipe) {
        return result;
    }

    char buffer[256];
    while (fgets(buffer, sizeof(buffer), pipe) != nullptr) {
        result += buffer;
    }

    int status = pclose(pipe);
    (void)status;

    return result;
}

/// Extract value from a "key = value" line.
std::string parseAttr(std::string_view line) {
    auto eqIdx = line.find('=');
    if (eqIdx == std::string_view::npos) {
        return "";
    }
    return trim(line.substr(eqIdx + 1));
}

} // anonymous namespace

std::string getSecretLinux(const std::string& service, const std::string& account) {
    // Build: secret-tool lookup service <service> account <account>
    std::string cmd = "secret-tool lookup service \"";
    cmd += service;
    cmd += "\" account \"";
    cmd += account;
    cmd += "\" 2>&1";

    std::string output = execRead(cmd.c_str());

    // secret-tool returns empty output + exit code 1 when not found
    output = trimTrailing(output);

    if (output.empty()) {
        throw KeySyncError(ErrorCode::NotFound,
            "secret not found: " + service + "/" + account);
    }

    // secret-tool might also print error messages to stdout via 2>&1
    if (output.find("secret-tool: ") == 0) {
        throw KeySyncError(ErrorCode::KeychainError, output);
    }

    return output;
}

std::vector<CredentialEntry> listSecretsLinux(const std::string& scope_filter,
                                               const std::string& project_filter) {
    std::vector<CredentialEntry> entries;

    // Search for all keysync entries
    std::string output = execRead("secret-tool search service keysync 2>&1");

    if (output.empty()) {
        return entries;
    }

    // Parse output:
    // [/N]
    // label = My Label
    // secret = <value>
    // created = 2024-01-01 12:00:00
    // modified = 2024-01-01 12:00:00
    // service = keysync/global
    // account = MY_KEY
    //
    // [/N+1]
    // ...

    std::string currentService;
    std::string currentAccount;
    bool inEntry = false;

    std::string line;
    size_t pos = 0;
    while (pos < output.size()) {
        auto nlPos = output.find('\n', pos);
        if (nlPos == std::string::npos) {
            line = output.substr(pos);
        } else {
            line = output.substr(pos, nlPos - pos);
        }

        // Check for blank line (entry separator)
        if (line.empty()) {
            if (!currentService.empty() && !currentAccount.empty() &&
                currentService.size() >= 8 && currentService.substr(0, 8) == "keysync/") {
                std::string entryScope, entryProject;
                parseServiceName(currentService, entryScope, entryProject);

                bool scopeMatch = scope_filter.empty() || entryScope == scope_filter;
                bool projectMatch = project_filter.empty() || entryProject == project_filter;

                if (scopeMatch && projectMatch) {
                    entries.push_back({entryScope, entryProject, currentAccount});
                }
            }
            currentService.clear();
            currentAccount.clear();
        } else {
            // Parse key = value lines
            auto eq = line.find('=');
            if (eq != std::string::npos) {
                std::string key = trimTrailing(line.substr(0, eq));
                std::string value = trim(line.substr(eq + 1));
                if (key == "service") {
                    currentService = value;
                } else if (key == "account") {
                    currentAccount = value;
                }
            }
        }

        if (nlPos == std::string::npos) {
            break;
        }
        pos = nlPos + 1;
    }

    // Handle last entry if no trailing blank line
    if (!currentService.empty() && !currentAccount.empty() &&
        currentService.size() >= 8 && currentService.substr(0, 8) == "keysync/") {
        std::string entryScope, entryProject;
        parseServiceName(currentService, entryScope, entryProject);

        bool scopeMatch = scope_filter.empty() || entryScope == scope_filter;
        bool projectMatch = project_filter.empty() || entryProject == project_filter;

        if (scopeMatch && projectMatch) {
            entries.push_back({entryScope, entryProject, currentAccount});
        }
    }

    return entries;
}

bool isNotFoundLinux(const std::string& service, const std::string& account) {
    std::string cmd = "secret-tool lookup service \"";
    cmd += service;
    cmd += "\" account \"";
    cmd += account;
    cmd += "\" 2>&1";

    std::string output = execRead(cmd.c_str());
    output = trimTrailing(output);

    // secret-tool returns exit code 1 and empty output for not found
    return output.empty();
}

} // namespace internal
} // namespace keysync

#endif // __linux__
