import Foundation

/// Service name helpers matching the keysync convention.
///
/// Secrets are stored with a service name that encodes scope and project:
///   Global:  "keysync/global"
///   Project: "keysync/project/<name>"
enum ServiceName {
    static func forScope(_ scope: String, project: String?) -> String {
        if let project = project, !project.isEmpty, scope == "project" {
            return "keysync/\(scope)/\(project)"
        }
        return "keysync/\(scope)"
    }

    /// Parse "keysync/global" → ("global", nil)
    /// Parse "keysync/project/my-app" → ("project", "my-app")
    static func parse(_ service: String) -> (scope: String, project: String?) {
        let trimmed = service
            .trimmingCharacters(in: .whitespaces)
            .replacingOccurrences(of: "keysync/", with: "")
        let parts = trimmed.split(separator: "/", maxSplits: 1, omittingEmptySubsequences: true)
        guard let scope = parts.first.map(String.init) else {
            return ("global", nil)
        }
        guard parts.count > 1, scope == "project" else {
            return (scope, nil)
        }
        return (scope, String(parts[1]))
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
