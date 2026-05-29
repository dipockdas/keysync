//! keysync — Read secrets from the OS-native keychain.
//!
//! This library provides zero-dependency access to secrets managed by
//! keysync. It reads directly from the OS keychain with no dependency
//! on the `keysync` binary.
//!
//! Each platform uses its native keychain tooling:
//!   - macOS:  `security` CLI (built-in)
//!   - Linux:  `secret-tool` CLI (libsecret)
//!   - Windows: Win32 Credential Manager API via `windows-sys`
//!
//! # Usage
//!
//! ```rust,no_run
//! use keysync::{get_secret, list_secrets};
//!
//! // Retrieve a global secret
//! let api_key = get_secret("API_KEY", None, None).unwrap();
//!
//! // Retrieve a project-scoped secret (falls back to global)
//! let db_url = get_secret("DATABASE_URL", Some("myapp"), None).unwrap();
//!
//! // Retrieve an environment-scoped secret
//! let staging_db = get_secret("DATABASE_URL", Some("myapp"), Some("staging")).unwrap();
//!
//! // List all global secrets
//! let globals = list_secrets(None, None).unwrap();
//!
//! // List project secrets + globals
//! let entries = list_secrets(Some("myapp"), None).unwrap();
//!
//! // List environment-scoped secrets + globals
//! let staging = list_secrets(Some("myapp"), Some("staging")).unwrap();
//! ```

mod error;
mod credential;
mod service;

pub use error::{KeySyncError, Result};
pub use credential::CredentialEntry;

#[cfg(target_os = "macos")]
mod macos;
#[cfg(target_os = "macos")]
use macos as platform;

#[cfg(target_os = "linux")]
mod linux;
#[cfg(target_os = "linux")]
use linux as platform;

#[cfg(target_os = "windows")]
mod windows;
#[cfg(target_os = "windows")]
use windows as platform;

#[cfg(not(any(target_os = "macos", target_os = "linux", target_os = "windows")))]
mod unsupported;
#[cfg(not(any(target_os = "macos", target_os = "linux", target_os = "windows")))]
use unsupported as platform;

/// Retrieve a secret.
///
/// Checks the environment variable identified by `key` first. If set, returns
/// it immediately without touching the OS keychain.
///
/// If the env var is not set, falls back to the OS keychain with this
/// resolution order:
/// 1. Environment-scoped: `keysync/project/<project>/env/<environment>`
/// 2. Project-scoped: `keysync/project/<project>`
/// 3. Global: `keysync/global`
///
/// # Examples
///
/// ```rust,no_run
/// use keysync::get_secret;
///
/// // Global secret
/// let api_key = get_secret("API_KEY", None, None).unwrap();
///
/// // Project-scoped secret (falls back to global)
/// let db_url = get_secret("DATABASE_URL", Some("myapp"), None).unwrap();
///
/// // Environment-scoped secret (falls back to project, then global)
/// let staging_db = get_secret("DATABASE_URL", Some("myapp"), Some("staging")).unwrap();
/// ```
///
/// # Errors
///
/// Returns `KeySyncError::NotFound` if the secret doesn't exist in any
/// checked scope. Returns `KeySyncError::KeychainError` if the keychain
/// tooling fails.
pub fn get_secret(
    key: &str,
    project: Option<&str>,
    environment: Option<&str>,
) -> Result<String> {
    // Primary path: check environment variable first.
    if let Ok(val) = std::env::var(key) {
        return Ok(val);
    }

    let project = project.unwrap_or("");
    let environment = environment.unwrap_or("");

    if !project.is_empty() {
        // 1. Try environment-scoped
        if !environment.is_empty() {
            let env_svc = service::service_name("project", project, environment);
            match platform::get_secret(&env_svc, key) {
                Ok(val) => return Ok(val),
                Err(e) if platform::is_not_found(&e) => {}
                Err(e) => return Err(e),
            }
        }

        // 2. Try project scope
        let svc = service::service_name("project", project, "");
        match platform::get_secret(&svc, key) {
            Ok(val) => return Ok(val),
            Err(e) if platform::is_not_found(&e) => {}
            Err(e) => return Err(e),
        }
    }

    // 3. Fall back to global scope
    let svc = service::service_name("global", "", "");
    match platform::get_secret(&svc, key) {
        Ok(val) => Ok(val),
        Err(e) if platform::is_not_found(&e) => Err(KeySyncError::NotFound),
        Err(e) => Err(e),
    }
}

/// List all stored secrets matching the given project and/or environment filter.
///
/// When no filters are provided, returns all entries.
/// When `project` is provided, returns that project's entries plus globals.
/// When both `project` and `environment` are provided, returns matching
/// environment-scoped entries plus globals.
///
/// # Examples
///
/// ```rust,no_run
/// use keysync::list_secrets;
///
/// // List all secrets across all scopes
/// let all = list_secrets(None, None).unwrap();
///
/// // List project-scoped + globals
/// let project = list_secrets(Some("myapp"), None).unwrap();
///
/// // List environment-scoped + globals
/// let staging = list_secrets(Some("myapp"), Some("staging")).unwrap();
/// ```
///
/// # Errors
///
/// Returns `KeySyncError::KeychainError` if the keychain tooling fails.
pub fn list_secrets(
    project: Option<&str>,
    environment: Option<&str>,
) -> Result<Vec<CredentialEntry>> {
    let entries = platform::list_secrets()?;

    let project_filter = project.unwrap_or("");
    let env_filter = environment.unwrap_or("");

    if project_filter.is_empty() && env_filter.is_empty() {
        return Ok(entries);
    }

    let filtered: Vec<CredentialEntry> = entries
        .into_iter()
        .filter(|e| {
            if e.scope == "global" {
                return true;
            }
            if e.scope == "project" && e.project == project_filter {
                if !env_filter.is_empty() {
                    return e.environment == env_filter || e.environment.is_empty();
                }
                return true;
            }
            false
        })
        .collect();

    Ok(filtered)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_get_secret_env_var() {
        std::env::set_var("KEY_SYNC_TEST_VAR", "test_value_123");
        let result = get_secret("KEY_SYNC_TEST_VAR", None, None);
        std::env::remove_var("KEY_SYNC_TEST_VAR");
        assert_eq!(result.unwrap(), "test_value_123");
    }

    #[test]
    fn test_get_secret_env_var_with_project() {
        std::env::set_var("KEY_SYNC_TEST_VAR", "from_env");
        let result = get_secret("KEY_SYNC_TEST_VAR", Some("myapp"), Some("staging"));
        std::env::remove_var("KEY_SYNC_TEST_VAR");
        assert_eq!(result.unwrap(), "from_env");
    }

    #[test]
    fn test_get_secret_not_found() {
        let result = get_secret("NONEXISTENT_SECRET_12345", None, None);
        match result {
            Err(KeySyncError::NotFound) | Err(KeySyncError::KeychainError(_)) => {}
            other => panic!("Expected NotFound or KeychainError, got: {:?}", other),
        }
    }

    #[test]
    fn test_get_secret_with_project() {
        let result = get_secret("NONEXISTENT_SECRET", Some("myapp"), None);
        match result {
            Err(KeySyncError::NotFound) | Err(KeySyncError::KeychainError(_)) => {}
            other => panic!("Expected NotFound or KeychainError, got: {:?}", other),
        }
    }

    #[test]
    fn test_get_secret_with_env() {
        let result = get_secret("NONEXISTENT_ENV_SECRET", Some("myapp"), Some("staging"));
        match result {
            Err(KeySyncError::NotFound) | Err(KeySyncError::KeychainError(_)) => {}
            other => panic!("Expected NotFound or KeychainError, got: {:?}", other),
        }
    }

    #[test]
    fn test_get_secret_empty_project() {
        let result = get_secret("NONEXISTENT_SECRET", Some(""), None);
        match result {
            Err(KeySyncError::NotFound) | Err(KeySyncError::KeychainError(_)) => {}
            other => panic!("Expected NotFound or KeychainError, got: {:?}", other),
        }
    }

    #[test]
    fn test_list_secrets_no_filter() {
        let result = list_secrets(None, None);
        match result {
            Ok(_entries) => {
                // Accept any valid entries; don't panic on unexpected scopes
                // which may be leftover test data from earlier runs.
            }
            Err(KeySyncError::KeychainError(_)) => {}
            Err(e) => panic!("Unexpected error: {:?}", e),
        }
    }

    #[test]
    fn test_list_secrets_with_project() {
        let result = list_secrets(Some("myapp"), None);
        match result {
            Ok(entries) => {
                for entry in &entries {
                    if entry.scope == "project" {
                        assert_eq!(entry.project, "myapp");
                    }
                }
            }
            Err(KeySyncError::KeychainError(_)) => {}
            Err(e) => panic!("Unexpected error: {:?}", e),
        }
    }

    #[test]
    fn test_list_secrets_with_env_filter() {
        let result = list_secrets(Some("myapp"), Some("staging"));
        match result {
            Ok(entries) => {
                for entry in &entries {
                    assert!(
                        entry.scope == "global"
                            || (entry.scope == "project"
                                && entry.project == "myapp"
                                && (entry.environment == "staging"
                                    || entry.environment.is_empty())),
                        "unexpected entry: {:?}", entry
                    );
                }
            }
            Err(KeySyncError::KeychainError(_)) => {}
            Err(e) => panic!("Unexpected error: {:?}", e),
        }
    }

    #[test]
    fn test_list_secrets_empty_project() {
        let result = list_secrets(Some(""), None);
        match result {
            Ok(_entries) => {}
            Err(KeySyncError::KeychainError(_)) => {}
            Err(e) => panic!("Unexpected error: {:?}", e),
        }
    }

    #[test]
    fn test_key_sync_error_display() {
        assert_eq!(format!("{}", KeySyncError::NotFound), "secret not found");
        assert_eq!(
            format!("{}", KeySyncError::KeychainError("test error".to_string())),
            "keychain error: test error"
        );
        assert_eq!(
            format!("{}", KeySyncError::UnsupportedPlatform),
            "keychain access not available on this platform"
        );
    }

    #[test]
    fn test_key_sync_error_debug() {
        let debug_str = format!("{:?}", KeySyncError::NotFound);
        assert!(debug_str.contains("NotFound"));

        let debug_str = format!("{:?}", KeySyncError::KeychainError("some error".to_string()));
        assert!(debug_str.contains("KeychainError"));
        assert!(debug_str.contains("some error"));
    }

    #[test]
    fn test_key_sync_error_is_std_error() {
        fn check_is_error<T: std::error::Error>(_: &T) {}
        let err = KeySyncError::NotFound;
        check_is_error(&err);
        assert!(std::error::Error::source(&err).is_none());
    }

    #[test]
    fn test_io_error_conversion() {
        let io_err = std::io::Error::new(std::io::ErrorKind::NotFound, "file not found");
        let sync_err: KeySyncError = io_err.into();
        match sync_err {
            KeySyncError::KeychainError(msg) => {
                assert!(msg.contains("file not found"));
            }
            _ => panic!("Expected KeychainError"),
        }
    }

    #[test]
    fn test_get_secret_env_var_empty_string() {
        std::env::set_var("KEY_SYNC_TEST_EMPTY", "");
        let result = get_secret("KEY_SYNC_TEST_EMPTY", None, None);
        std::env::remove_var("KEY_SYNC_TEST_EMPTY");
        assert_eq!(result.unwrap(), "");
    }
}
