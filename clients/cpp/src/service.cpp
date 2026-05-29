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

std::string serviceName(std::string_view scope, std::string_view project,
                         std::string_view environment) {
    std::string result = "keysync/";
    result += scope;
    if (!project.empty() && scope == "project") {
        result += "/";
        result += project;
        if (!environment.empty()) {
            result += "/env/";
            result += environment;
        }
    }
    return result;
}

void parseServiceName(std::string_view service, std::string& scope,
                       std::string& project, std::string& environment) {
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
        environment.clear();
        return;
    }

    scope = trimmed.substr(0, slashPos);
    std::string_view rest = trimmed.substr(slashPos + 1);

    // Check for /env/ segment to detect environment
    constexpr std::string_view envMarker = "/env/";
    auto envIdx = rest.find(envMarker);
    if (envIdx != std::string_view::npos && envIdx > 0) {
        project = rest.substr(0, envIdx);
        environment = rest.substr(envIdx + envMarker.size());
    } else {
        project = rest;
        environment.clear();
    }
}

std::string serviceToTarget(std::string_view service) {
    // Strip /env/ keyword before converting
    std::string processed(service);
    constexpr std::string_view envMarker = "/env/";
    size_t pos = processed.find(envMarker);
    if (pos != std::string::npos) {
        processed.erase(pos + 1, envMarker.size() - 1); // remove "env" keeping the "/"
    }

    std::string target;
    constexpr std::string_view svcPrefix = "keysync/";
    if (processed.size() >= svcPrefix.size() && processed.substr(0, svcPrefix.size()) == svcPrefix) {
        target = "keysync_";
        std::string_view rest(processed.data() + svcPrefix.size(),
                              processed.size() - svcPrefix.size());
        for (char ch : rest) {
            if (ch == '/') {
                target += '_';
            } else {
                target += ch;
            }
        }
    } else {
        target = "keysync_";
        for (char ch : processed) {
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
        std::string_view rest = target.substr(prefix.size());
        // Find the first underscore to get the scope
        auto firstUnderscore = rest.find('_');
        if (firstUnderscore == std::string_view::npos) {
            return "keysync/" + std::string(rest);
        }

        std::string scope(rest.substr(0, firstUnderscore));
        std::string_view restStr = rest.substr(firstUnderscore + 1);

        if (scope == "global") {
            return "keysync/global";
        }

        // For project scope: convert underscores to slashes
        // Check for 3+ segments (project_env_more => project/env/more)
        auto secondUnderscore = restStr.find('_');
        if (secondUnderscore != std::string_view::npos) {
            // Has 3 segments: project + env + possibly more
            std::string projectPart(restStr.substr(0, secondUnderscore));
            std::string envPart(restStr.substr(secondUnderscore + 1));
            // Replace remaining underscores with slashes in env part
            std::replace(envPart.begin(), envPart.end(), '_', '/');
            return "keysync/" + scope + "/" + projectPart + "/env/" + envPart;
        }

        // Only 2 segments: just project
        std::string projectPart(restStr);
        return "keysync/" + scope + "/" + projectPart;
    }

    return std::string(target);
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
