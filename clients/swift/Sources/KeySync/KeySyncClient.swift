import Foundation

/// Platform-selected keychain implementation.
#if os(macOS)
typealias PlatformKeychain = DarwinKeychain
#elseif os(Linux)
typealias PlatformKeychain = LinuxKeychain
#else
typealias PlatformKeychain = WindowsKeychain
#endif

/// Keysync client — retrieve secrets from the OS keychain.
///
/// Usage:
/// ```swift
/// import KeySync
///
/// // Project-scoped secret with global fallback
/// let dbURL = try KeySync.getSecret("DATABASE_URL", project: "my-api")
///
/// // Global-only secret
/// let apiKey = try KeySync.getSecret("GLOBAL_API_KEY")
/// ```
public enum KeySync {

    private static let keychain = PlatformKeychain()

    /// Retrieve a secret from the OS keychain.
    ///
    /// - Parameters:
    ///   - key: The secret key name (e.g. "DATABASE_URL").
    ///   - project: Optional project name. If provided, checks project scope
    ///     first, then falls back to global scope.
    /// - Returns: The secret value.
    /// - Throws: `KeySyncError.notFound` if the secret doesn't exist in any scope.
    public static func getSecret(_ key: String, project: String? = nil) throws -> String {
        // Try project scope first
        if let project = project, !project.isEmpty {
            let service = ServiceName.forScope("project", project: project)
            do {
                return try keychain.getSecret(service: service, account: key)
            } catch KeySyncError.notFound {
                // Fall through to global
            } catch {
                throw error
            }
        }

        // Fall back to global scope
        let service = ServiceName.forScope("global", project: nil)
        return try keychain.getSecret(service: service, account: key)
    }

    /// List all stored secret key names.
    ///
    /// - Parameters:
    ///   - scope: Filter by scope ("global" or "project"). Empty string or
    ///     nil returns all scopes.
    ///   - project: Filter by project name. Empty string or nil returns all
    ///     projects.
    /// - Returns: Array of `(scope, project, key)` tuples.
    public static func listSecrets(scope: String? = nil, project: String? = nil) throws -> [(scope: String, project: String?, key: String)] {
        let entries: [(service: String, account: String)] = try keychain.listSecrets()

        return entries
            .compactMap { entry in
                let (s, p) = ServiceName.parse(entry.service)
                return (scope: s, project: p, key: entry.account)
            }
            .filter { entry in
                let scopeMatch = scope == nil || scope?.isEmpty == true || entry.scope == scope
                let projectMatch = project == nil || project?.isEmpty == true || entry.project == project
                return scopeMatch && projectMatch
            }
    }
}
