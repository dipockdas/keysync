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
//! let api_key = get_secret("API_KEY", None).unwrap();
//!
//! // Retrieve a project-scoped secret (falls back to global)
//! let db_url = get_secret("DATABASE_URL", Some("myapp")).unwrap();
//!
//! // List all global secrets
//! let globals = list_secrets(None).unwrap();
//!
//! // List project secrets
//! let entries = list_secrets(Some("myapp")).unwrap();
//! ```

mod error;
mod credential;
mod service;

pub use error::{KeySyncError, Result};
pub use credential::CredentialEntry;

// Platform-specific modules are included via cfg attributes.
// Each platform module exports three functions:
//   get_secret(service, account) -> Result<String>
//   list_secrets() -> Result<Vec<CredentialEntry>>
//   is_not_found(&KeySyncError) -> bool

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
/// it immediately without touching the OS keychain. This is the primary path
/// for both local development (where secrets are injected via
/// `eval $(keysync export)`) and cloud deployments (where platforms inject
/// environment variables directly).
///
/// If the env var is not set, falls back to the OS keychain. When `project`
/// is non-empty, it checks project scope first, then global scope.
///
/// # Examples
///
/// ```rust,no_run
/// use keysync::get_secret;
///
/// // Global secret
/// let api_key = get_secret("API_KEY", None).unwrap();
///
/// // Project-scoped secret (falls back to global)
/// let db_url = get_secret("DATABASE_URL", Some("myapp")).unwrap();
/// ```
///
/// # Errors
///
/// Returns `KeySyncError::NotFound` if the secret doesn't exist in any
/// checked scope. Returns `KeySyncError::KeychainError` if the keychain
/// tooling fails. Returns `KeySyncError::UnsupportedPlatform` on
/// unsupported platforms.
pub fn get_secret(key: &str, project: Option<&str>) -> Result<String> {
    // Primary path: check environment variable first.
    // In local dev the user runs `eval $(keysync export)` at shell startup;
    // in cloud/CI the platform injects env vars directly.
    if let Ok(val) = std::env::var(key) {
        return Ok(val);
    }

    // Try project scope first
    if let Some(project) = project {
        if !project.is_empty() {
            let svc = service::service_name("project", project);
            match platform::get_secret(&svc, key) {
                Ok(val) => return Ok(val),
                Err(e) if platform::is_not_found(&e) => {
                    // Fall through to global scope
                }
                Err(e) => return Err(e),
            }
        }
    }

    // Fall back to global scope
    let svc = service::service_name("global", "");
    match platform::get_secret(&svc, key) {
        Ok(val) => Ok(val),
        Err(e) if platform::is_not_found(&e) => {
            Err(KeySyncError::NotFound)
        }
        Err(e) => Err(e),
    }
}

/// List all stored secrets matching the given project filter.
///
/// When `project` is `None`, returns all global secrets plus all
/// project-scoped entries from any project.
///
/// When `project` is `Some(name)`, returns project-scoped entries
/// for that project plus all global entries.
///
/// # Examples
///
/// ```rust,no_run
/// use keysync::list_secrets;
///
/// // List all secrets across all scopes
/// let all = list_secrets(None).unwrap();
///
/// // List only project-specific secrets + globals
/// let project = list_secrets(Some("myapp")).unwrap();
/// ```
///
/// # Errors
///
/// Returns `KeySyncError::KeychainError` if the keychain tooling fails.
/// Returns `KeySyncError::UnsupportedPlatform` on unsupported platforms.
pub fn list_secrets(project: Option<&str>) -> Result<Vec<CredentialEntry>> {
    let entries = platform::list_secrets()?;

    if let Some(project_filter) = project {
        if !project_filter.is_empty() {
            // Return project-scoped entries for the given project
            // plus all global entries
            let filtered: Vec<CredentialEntry> = entries
                .into_iter()
                .filter(|e| {
                    e.scope == "global" || (e.scope == "project" && e.project == project_filter)
                })
                .collect();
            return Ok(filtered);
        }
    }

    // No filter — return all entries
    Ok(entries)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_get_secret_env_var() {
        // Set an env var and verify get_secret returns it
        std::env::set_var("KEY_SYNC_TEST_VAR", "test_value_123");
        let result = get_secret("KEY_SYNC_TEST_VAR", None);
        std::env::remove_var("KEY_SYNC_TEST_VAR");
        assert_eq!(result.unwrap(), "test_value_123");
    }

    #[test]
    fn test_get_secret_env_var_with_project() {
        std::env::set_var("KEY_SYNC_TEST_VAR", "from_env");
        // Env var should take priority even when project is provided
        let result = get_secret("KEY_SYNC_TEST_VAR", Some("myapp"));
        std::env::remove_var("KEY_SYNC_TEST_VAR");
        assert_eq!(result.unwrap(), "from_env");
    }

    #[test]
    fn test_get_secret_not_found() {
        // Without a keysync keychain entry, this should return NotFound
        let result = get_secret("NONEXISTENT_SECRET_12345", None);
        match result {
            Err(KeySyncError::NotFound) => {
                // Expected — no keychain entry exists for this
            }
            Err(KeySyncError::KeychainError(_)) => {
                // Also acceptable — the keychain tooling might not be available
            }
            other => {
                panic!("Expected NotFound or KeychainError, got: {:?}", other);
            }
        }
    }

    #[test]
    fn test_get_secret_with_project() {
        // Without any keychain entries, this should return NotFound
        let result = get_secret("NONEXISTENT_SECRET", Some("myapp"));
        match result {
            Err(KeySyncError::NotFound) | Err(KeySyncError::KeychainError(_)) => {}
            other => panic!("Expected NotFound or KeychainError, got: {:?}", other),
        }
    }

    #[test]
    fn test_get_secret_empty_project() {
        // Empty project should act like no project
        // Should fall through to global scope directly
        let result = get_secret("NONEXISTENT_SECRET", Some(""));
        match result {
            Err(KeySyncError::NotFound) | Err(KeySyncError::KeychainError(_)) => {}
            other => panic!("Expected NotFound or KeychainError, got: {:?}", other),
        }
    }

    #[test]
    fn test_list_secrets_no_filter() {
        let result = list_secrets(None);
        // This may succeed (return empty list) or fail depending on keychain state
        match result {
            Ok(entries) => {
                // Entries might exist if keychain has keysync data, or be empty
                for entry in &entries {
                    assert!(
                        entry.scope == "global" || entry.scope == "project",
                        "unexpected scope: {}", entry.scope
                    );
                }
            }
            Err(KeySyncError::KeychainError(_)) => {
                // Acceptable if keychain tooling is unavailable
            }
            Err(e) => panic!("Unexpected error: {:?}", e),
        }
    }

    #[test]
    fn test_list_secrets_with_project() {
        let result = list_secrets(Some("myapp"));
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
    fn test_list_secrets_empty_project() {
        let result = list_secrets(Some(""));
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
        // Just verify that Debug formatting works and contains the variant name
        let debug_str = format!("{:?}", KeySyncError::NotFound);
        assert!(debug_str.contains("NotFound"));

        let debug_str = format!(
            "{:?}",
            KeySyncError::KeychainError("some error".to_string())
        );
        assert!(debug_str.contains("KeychainError"));
        assert!(debug_str.contains("some error"));
    }

    #[test]
    fn test_key_sync_error_is_std_error() {
        // Verify KeySyncError implements std::error::Error
        fn check_is_error<T: std::error::Error>(_: &T) {}
        let err = KeySyncError::NotFound;
        check_is_error(&err);

        // source() should return None (we don't chain errors)
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
        // An empty env var is still "set" and should be returned
        std::env::set_var("KEY_SYNC_TEST_EMPTY", "");
        let result = get_secret("KEY_SYNC_TEST_EMPTY", None);
        std::env::remove_var("KEY_SYNC_TEST_EMPTY");
        assert_eq!(result.unwrap(), "");
    }
}
