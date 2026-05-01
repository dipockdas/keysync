import Foundation

/// Windows keychain access — currently unsupported in Swift.
///
/// Swift on Windows is experimental and Win32 API bindings are not yet
/// mature enough for production use. This stub throws an error.
///
/// For Windows support, use the Go, Node, or Python client libraries.
struct WindowsKeychain {

    func getSecret(service: String, account: String) throws -> String {
        throw KeySyncError.unsupportedPlatform
    }

    func listSecrets() throws -> [(service: String, account: String)] {
        throw KeySyncError.unsupportedPlatform
    }
}
