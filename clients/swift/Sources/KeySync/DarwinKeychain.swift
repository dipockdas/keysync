import Foundation
import Security

/// macOS keychain access via native Security.framework.
///
/// Uses SecItemCopyMatching to retrieve generic passwords stored by keysync.
/// No subprocess or CLI dependency — the cleanest integration of any language.
struct DarwinKeychain {

    func getSecret(service: String, account: String) throws -> String {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]

        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        switch status {
        case errSecItemNotFound:
            throw KeySyncError.notFound
        case errSecSuccess:
            guard let data = result as? Data else {
                throw KeySyncError.unexpectedData
            }
            guard let value = String(data: data, encoding: .utf8) else {
                throw KeySyncError.unexpectedData
            }
            return value
        default:
            let message = SecCopyErrorMessageString(status, nil) as String? ?? "unknown error (\(status))"
            throw KeySyncError.keychainError(message)
        }
    }

    /// List all keysync-managed secrets in the keychain.
    ///
    /// Fetches all generic password items and filters for those with a
    /// service name starting with "keysync/".
    func listSecrets() throws -> [(service: String, account: String)] {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecMatchLimit as String: kSecMatchLimitAll,
            kSecReturnAttributes as String: true,
        ]

        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status != errSecItemNotFound else {
            return []
        }
        guard status == errSecSuccess else {
            return []
        }

        guard let items = result as? [[String: Any]] else {
            return []
        }

        return items.compactMap { item in
            guard let svc = item[kSecAttrService as String] as? String,
                  svc.hasPrefix("keysync/"),
                  let acct = item[kSecAttrAccount as String] as? String else {
                return nil
            }
            return (svc, acct)
        }
    }
}
