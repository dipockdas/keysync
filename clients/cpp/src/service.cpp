#include "internal_helpers.hpp"
#include <algorithm>
#include <cctype>

namespace keysync {
namespace internal {

std::string serviceName(std::string_view scope, std::string_view project) {
    std::string result = "keysync/";
    result += scope;
    if (!project.empty() && scope == "project") {
        result += "/";
        result += project;
    }
    return result;
}

void parseServiceName(std::string_view service, std::string& scope, std::string& project) {
    // Strip "keysync/" prefix
    std::string_view trimmed = service;
    constexpr std::string_view prefix = "keysync/";
    if (trimmed.size() >= prefix.size() && trimmed.substr(0, prefix.size()) == prefix) {
        trimmed = trimmed.substr(prefix.size());
    }

    // Find the first slash
    auto slashPos = trimmed.find('/');
    if (slashPos == std::string_view::npos) {
        scope = trimmed;
        project.clear();
        return;
    }

    scope = trimmed.substr(0, slashPos);
    project = trimmed.substr(slashPos + 1);
}

std::string serviceToTarget(std::string_view service) {
    std::string target;
    if (service.size() >= 8 && service.substr(0, 8) == "keysync/") {
        target = "keysync_";
        std::string_view rest = service.substr(8);
        for (char ch : rest) {
            if (ch == '/') {
                target += '_';
            } else {
                target += ch;
            }
        }
    } else {
        target = "keysync_";
        for (char ch : service) {
            if (ch == '/') {
                target += '_';
            } else {
                target += ch;
            }
        }
    }
    return target;
}

std::string targetToService(std::string_view target) {
    std::string service;
    constexpr std::string_view prefix = "keysync_";
    if (target.size() >= prefix.size() && target.substr(0, prefix.size()) == prefix) {
        service = "keysync/";
        std::string_view rest = target.substr(prefix.size());
        bool firstUnderscore = true;
        for (char ch : rest) {
            if (ch == '_' && firstUnderscore) {
                service += '/';
                firstUnderscore = false;
            } else {
                service += ch;
            }
        }
    } else {
        service = target;
    }
    return service;
}

std::string trimTrailing(std::string_view s) {
    auto end = s.find_last_not_of(" \t\n\r\f\v");
    if (end == std::string_view::npos) {
        return "";
    }
    return std::string(s.substr(0, end + 1));
}

std::string trim(std::string_view s) {
    auto start = s.find_first_not_of(" \t\n\r\f\v");
    if (start == std::string_view::npos) {
        return "";
    }
    auto end = s.find_last_not_of(" \t\n\r\f\v");
    return std::string(s.substr(start, end - start + 1));
}

} // namespace internal
} // namespace keysync
