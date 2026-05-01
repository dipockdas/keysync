import Foundation

/// Linux keychain access via libsecret's `secret-tool` CLI.
///
/// Falls back to the `secret-tool` command-line tool, which is part of
/// libsecret. Requires `libsecret-tools` to be installed and a running
/// secret service (GNOME Keyring, KDE Wallet, KeePassXC).
struct LinuxKeychain {

    private let secretToolPath = "/usr/bin/secret-tool"

    func getSecret(service: String, account: String) throws -> String {
        let output = try runSecretTool(arguments: ["lookup", "service", service, "account", account])
        let value = output.trimmingCharacters(in: .whitespacesAndNewlines)
        if value.isEmpty {
            throw KeySyncError.notFound
        }
        return value
    }

    func listSecrets() throws -> [(service: String, account: String)] {
        // secret-tool search outputs lines like:
        //   service = keysync/global
        //   account = MY_KEY
        //   password = <value>
        // with blank lines between entries.
        let output = try runSecretTool(arguments: ["search", "service", "keysync"])
        let lines = output.components(separatedBy: .newlines)

        var results: [(service: String, account: String)] = []
        var currentService: String?
        var currentAccount: String?

        for line in lines {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if trimmed.isEmpty {
                // Blank line = end of entry
                if let svc = currentService, let acct = currentAccount {
                    results.append((svc, acct))
                }
                currentService = nil
                currentAccount = nil
                continue
            }
            if trimmed.hasPrefix("service") {
                currentService = parseAttributeValue(trimmed)
            } else if trimmed.hasPrefix("account") {
                currentAccount = parseAttributeValue(trimmed)
            }
        }
        // Last entry if no trailing blank line
        if let svc = currentService, let acct = currentAccount {
            results.append((svc, acct))
        }

        return results
    }

    // MARK: - Private

    private func runSecretTool(arguments: [String]) throws -> String {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: secretToolPath)
        process.arguments = arguments

        let outputPipe = Pipe()
        let errorPipe = Pipe()
        process.standardOutput = outputPipe
        process.standardError = errorPipe

        try process.run()
        process.waitUntilExit()

        let outputData = outputPipe.fileHandleForReading.readDataToEndOfFile()
        let errorData = errorPipe.fileHandleForReading.readDataToEndOfFile()

        if process.terminationStatus != 0 {
            let errorOutput = String(data: errorData, encoding: .utf8) ?? ""
            // secret-tool returns exit code 1 when the secret is not found
            if errorOutput.contains("not found") || errorOutput.isEmpty {
                throw KeySyncError.notFound
            }
            throw KeySyncError.keychainError("secret-tool exited with status \(process.terminationStatus): \(errorOutput)")
        }

        return String(data: outputData, encoding: .utf8) ?? ""
    }

    private func parseAttributeValue(_ line: String) -> String {
        let parts = line.split(separator: "=", maxSplits: 1, omittingEmptySubsequences: false)
        guard parts.count == 2 else { return "" }
        return String(parts[1]).trimmingCharacters(in: .whitespaces)
    }
}
