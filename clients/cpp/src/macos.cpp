/// macOS implementation: uses the built-in `security` CLI via popen().
///
/// Stored entries use the macOS Keychain's "generic password" class.
///   Service: "keysync/global" or "keysync/project/<name>"
///   Account: the secret key name (e.g. "DATABASE_URL")

#ifdef __APPLE__

#include "internal_helpers.hpp"
#include "keysync/errors.hpp"
#include "keysync/credential.hpp"

#include <cstdio>
#include <cstdlib>
#include <memory>
#include <string>
#include <vector>

namespace keysync {
namespace internal {
namespace {

/// Execute a shell command with popen() and return stdout as string.
/// Returns empty string on failure.
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
    (void)status; // suppress unused warning

    return result;
}

/// Extract a named attribute value from a `security dump-keychain` record.
/// Records look like:
///     "svce"<NULL>
///     "acct"<NULL>
/// or
///     "svce"<blob>="keysync/global"
///     "acct"<blob>="DATABASE_URL"
std::string findAttrValue(const std::string& record, const std::string& attrName) {
    std::string search = "\"" + attrName + "\"";
    auto idx = record.find(search);
    if (idx == std::string::npos) {
        return "";
    }

    std::string after = record.substr(idx + search.size());

    auto eqIdx = after.find('=');
    if (eqIdx == std::string::npos) {
        return "";
    }

    std::string val = after.substr(eqIdx + 1);
    // Trim whitespace
    auto start = val.find_first_not_of(" \t\n\r");
    if (start == std::string::npos) {
        return "";
    }
    val = val.substr(start);

    if (val == "<NULL>") {
        return "";
    }

    if (!val.empty() && val[0] == '"') {
        auto end = val.find('"', 1);
        if (end != std::string::npos) {
            return val.substr(1, end - 1);
        }
    }

    // Trim trailing quotes
    while (!val.empty() && val.back() == '"') {
        val.pop_back();
    }
    return val;
}

} // anonymous namespace

std::string getSecretMacOS(const std::string& service, const std::string& account) {
    // Build command: security find-generic-password -s <service> -a <account> -w 2>&1
    std::string cmd = "security find-generic-password -s \"";
    cmd += service;
    cmd += "\" -a \"";
    cmd += account;
    cmd += "\" -w 2>&1";

    std::string output = execRead(cmd.c_str());

    if (output.empty()) {
        throw KeySyncError(ErrorCode::NotFound,
            "secret not found: " + service + "/" + account);
    }

    // Check if output contains "security:" error prefix
    if (output.find("security: ") == 0) {
        // Exit code 44 means item not found, anything else is a keychain error
        if (output.find("could not be found") != std::string::npos ||
            output.find("The specified item could not be found") != std::string::npos) {
            throw KeySyncError(ErrorCode::NotFound,
                "secret not found: " + service + "/" + account);
        }
        throw KeySyncError(ErrorCode::KeychainError, trimTrailing(output));
    }

    return trimTrailing(output);
}

std::vector<CredentialEntry> listSecretsMacOS(const std::string& scope_filter,
                                               const std::string& project_filter) {
    std::vector<CredentialEntry> entries;

    // Dump the keychain and parse generic password entries
    std::string output = execRead("security dump-keychain 2>&1");

    if (output.empty()) {
        return entries;
    }

    // Split on "keychain:" records
    size_t pos = 0;
    while (true) {
        auto nextRecord = output.find("\nkeychain:", pos);
        std::string record;

        if (nextRecord == std::string::npos) {
            record = output.substr(pos);
        } else {
            record = output.substr(pos, nextRecord - pos);
        }

        // Only process generic password ("genp") entries
        if (record.find("class: \"genp\"") != std::string::npos) {
            std::string svc = findAttrValue(record, "svce");
            if (svc.size() >= 8 && svc.substr(0, 8) == "keysync/") {
                std::string acct = findAttrValue(record, "acct");
                if (!acct.empty()) {
                    std::string entryScope, entryProject;
                    parseServiceName(svc, entryScope, entryProject);

                    bool scopeMatch = scope_filter.empty() || entryScope == scope_filter;
                    bool projectMatch = project_filter.empty() || entryProject == project_filter;

                    if (scopeMatch && projectMatch) {
                        entries.push_back({entryScope, entryProject, acct});
                    }
                }
            }
        }

        if (nextRecord == std::string::npos) {
            break;
        }
        pos = nextRecord + 1; // skip past \n
    }

    return entries;
}

bool isNotFoundMacOS(const std::string& service, const std::string& account) {
    std::string cmd = "security find-generic-password -s \"";
    cmd += service;
    cmd += "\" -a \"";
    cmd += account;
    cmd += "\" -w 2>&1";

    std::string output = execRead(cmd.c_str());

    if (output.empty()) {
        return true;
    }

    // security returns exit code 44 for not found
    if (output.find("could not be found") != std::string::npos ||
        output.find("The specified item could not be found") != std::string::npos) {
        return true;
    }

    return false;
}

} // namespace internal
} // namespace keysync

#endif // __APPLE__
