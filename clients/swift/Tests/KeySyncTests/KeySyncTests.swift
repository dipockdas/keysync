import Testing
import Foundation
@testable import KeySync

struct KeySyncTests {

    // MARK: - ServiceName

    @Test func serviceNameGlobal() {
        let name = ServiceName.forScope("global", project: nil)
        #expect(name == "keysync/global")
    }

    @Test func serviceNameProject() {
        let name = ServiceName.forScope("project", project: "my-app")
        #expect(name == "keysync/project/my-app")
    }

    @Test func serviceNameGlobalWithProject() {
        // Global scope ignores project parameter
        let name = ServiceName.forScope("global", project: "my-app")
        #expect(name == "keysync/global")
    }

    @Test func serviceNameProjectNoName() {
        let name = ServiceName.forScope("project", project: nil)
        #expect(name == "keysync/project")
    }

    @Test func parseGlobal() {
        let (scope, project) = ServiceName.parse("keysync/global")
        #expect(scope == "global")
        #expect(project == nil)
    }

    @Test func parseProject() {
        let (scope, project) = ServiceName.parse("keysync/project/my-app")
        #expect(scope == "project")
        #expect(project == "my-app")
    }

    @Test func parseProjectDeep() {
        let (scope, project) = ServiceName.parse("keysync/project/my/deep/app")
        #expect(scope == "project")
        #expect(project == "my/deep/app")
    }

    @Test func parseJustKeysync() {
        let (scope, project) = ServiceName.parse("keysync")
        #expect(scope == "global")
        #expect(project == nil)
    }

    @Test func parseUnrecognized() {
        let (scope, project) = ServiceName.parse("keysync/other/val")
        #expect(scope == "other")
        #expect(project == nil)
    }

    @Test func parseEmpty() {
        let (scope, project) = ServiceName.parse("")
        #expect(scope == "global")
        #expect(project == nil)
    }

    // MARK: - KeySyncError

    @Test func errorDescriptions() {
        #expect(KeySyncError.notFound.description == "secret not found")
        #expect(KeySyncError.unexpectedData.description == "keychain returned unexpected data")
        #expect(KeySyncError.unsupportedPlatform.description == "keychain access not available on this platform")

        let keychainErr = KeySyncError.keychainError("something broke")
        #expect(keychainErr.description == "keychain error: something broke")
    }

    // MARK: - DarwinKeychain (unit tests with mocked behavior)

    @Test func darwinKeychainGetSecretNotFound() {
        // DarwinKeychain.listSecrets on a machine without keysync entries returns empty
        let keychain = DarwinKeychain()
        let results = try? keychain.listSecrets()
        // This should not throw, just return empty
        #expect(results != nil)
    }
}
