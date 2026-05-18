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

    @Test func serviceNameProjectWithEnvironment() {
        let name = ServiceName.forScope("project", project: "my-app", environment: "staging")
        #expect(name == "keysync/project/my-app/env/staging")
    }

    @Test func serviceNameGlobalWithProject() {
        // Global scope ignores project parameter
        let name = ServiceName.forScope("global", project: "my-app")
        #expect(name == "keysync/global")
    }

    @Test func serviceNameGlobalWithProjectAndEnvironment() {
        // Global scope ignores project and environment parameters
        let name = ServiceName.forScope("global", project: "my-app", environment: "staging")
        #expect(name == "keysync/global")
    }

    @Test func serviceNameProjectNoName() {
        let name = ServiceName.forScope("project", project: nil)
        #expect(name == "keysync/project")
    }

    @Test func serviceNameProjectNoNameWithEnvironment() {
        // Empty project means no environment encoding either
        let name = ServiceName.forScope("project", project: nil, environment: "staging")
        #expect(name == "keysync/project")
    }

    @Test func serviceNameProjectEmptyEnvironment() {
        // Empty environment should not add /env/ segment
        let name = ServiceName.forScope("project", project: "my-app", environment: "")
        #expect(name == "keysync/project/my-app")
    }

    @Test func parseGlobal() {
        let (scope, project, environment) = ServiceName.parse("keysync/global")
        #expect(scope == "global")
        #expect(project == nil)
        #expect(environment == nil)
    }

    @Test func parseProject() {
        let (scope, project, environment) = ServiceName.parse("keysync/project/my-app")
        #expect(scope == "project")
        #expect(project == "my-app")
        #expect(environment == nil)
    }

    @Test func parseProjectWithEnvironment() {
        let (scope, project, environment) = ServiceName.parse("keysync/project/my-app/env/staging")
        #expect(scope == "project")
        #expect(project == "my-app")
        #expect(environment == "staging")
    }

    @Test func parseProjectWithNestedEnvironment() {
        let (scope, project, environment) = ServiceName.parse("keysync/project/my-app/env/staging/us-east")
        #expect(scope == "project")
        #expect(project == "my-app")
        #expect(environment == "staging/us-east")
    }

    @Test func parseProjectDeep() {
        let (scope, project, environment) = ServiceName.parse("keysync/project/my/deep/app")
        #expect(scope == "project")
        #expect(project == "my/deep/app")
        #expect(environment == nil)
    }

    @Test func parseProjectWithEnvLiteralInName() {
        // If project name contains "env" but not as a separate segment, no environment
        let (scope, project, environment) = ServiceName.parse("keysync/project/my-env-app")
        #expect(scope == "project")
        #expect(project == "my-env-app")
        #expect(environment == nil)
    }

    @Test func parseJustKeysync() {
        let (scope, project, environment) = ServiceName.parse("keysync")
        #expect(scope == "global")
        #expect(project == nil)
        #expect(environment == nil)
    }

    @Test func parseUnrecognized() {
        let (scope, project, environment) = ServiceName.parse("keysync/other/val")
        #expect(scope == "other")
        #expect(project == nil)
        #expect(environment == nil)
    }

    @Test func parseEmpty() {
        let (scope, project, environment) = ServiceName.parse("")
        #expect(scope == "global")
        #expect(project == nil)
        #expect(environment == nil)
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
