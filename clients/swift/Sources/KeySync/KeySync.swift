import Foundation

/// Service name helpers matching the keysync convention.
///
/// Secrets are stored with a service name that encodes scope, project, and environment:
///   Global:      "keysync/global"
///   Project:     "keysync/project/<name>"
///   Environment: "keysync/project/<name>/env/<env>"
enum ServiceName {
    static func forScope(_ scope: String, project: String?, environment: String? = nil) -> String {
        if let project = project, !project.isEmpty, scope == "project" {
            if let env = environment, !env.isEmpty {
                return "keysync/\(scope)/\(project)/env/\(env)"
            }
            return "keysync/\(scope)/\(project)"
        }
        return "keysync/\(scope)"
    }

    /// Parse "keysync/global" → ("global", nil, nil)
    /// Parse "keysync/project/my-app" → ("project", "my-app", nil)
    /// Parse "keysync/project/my-app/env/staging" → ("project", "my-app", "staging")
    static func parse(_ service: String) -> (scope: String, project: String?, environment: String?) {
        let trimmed = service
            .trimmingCharacters(in: .whitespaces)
            .replacingOccurrences(of: "keysync/", with: "")
        // "keysync" without a slash is equivalent to global
        if trimmed == "keysync" {
            return ("global", nil, nil)
        }
        let parts = trimmed.split(separator: "/", omittingEmptySubsequences: true)
        guard let scope = parts.first.map(String.init) else {
            return ("global", nil, nil)
        }
        guard parts.count > 1, scope == "project" else {
            return (scope, nil, nil)
        }

        // Check for "/env/" segment to detect environment.
        // envIndex > 1 ensures the "env" literal is not the project name itself.
        // envIndex < parts.count - 1 ensures there is a value after "env".
        if let envIndex = parts.firstIndex(of: "env"), envIndex > 1, envIndex < parts.count - 1 {
            let projectParts = parts[1..<envIndex]
            let envParts = parts[(envIndex + 1)...]
            let project = projectParts.joined(separator: "/")
            let environment = envParts.joined(separator: "/")
            return ("project", project, environment)
        }

        let project = parts[1...].joined(separator: "/")
        return ("project", project, nil)
    }
}

/// Errors thrown by the KeySync library.
public enum KeySyncError: Error, Equatable {
    /// The requested secret was not found in any scope.
    case notFound
    /// The keychain returned unexpected data (not valid UTF-8).
    case unexpectedData
    /// An OS-level keychain error occurred.
    case keychainError(String)
    /// The platform is not supported.
    case unsupportedPlatform
}

extension KeySyncError: CustomStringConvertible {
    public var description: String {
        switch self {
        case .notFound:
            return "secret not found"
        case .unexpectedData:
            return "keychain returned unexpected data"
        case .keychainError(let detail):
            return "keychain error: \(detail)"
        case .unsupportedPlatform:
            return "keychain access not available on this platform"
        }
    }
}
