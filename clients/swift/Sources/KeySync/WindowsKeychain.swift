import Foundation

/// Windows keychain access — not planned.
///
/// Swift on Windows is experimental and most Windows users would use
/// a .NET language. For Windows support, use the Go, Node (TypeScript),
/// or Python client libraries.
struct WindowsKeychain {

    func getSecret(service: String, account: String) throws -> String {
        throw KeySyncError.unsupportedPlatform
    }

    func listSecrets() throws -> [(service: String, account: String)] {
        throw KeySyncError.unsupportedPlatform
    }
}
