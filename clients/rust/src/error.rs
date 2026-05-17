use std::fmt;

/// Errors returned by the KeySync library.
#[derive(Debug)]
pub enum KeySyncError {
    /// The requested secret was not found in any scope.
    NotFound,
    /// An OS-level keychain error occurred.
    KeychainError(String),
    /// The current platform is not supported.
    UnsupportedPlatform,
}

impl fmt::Display for KeySyncError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            KeySyncError::NotFound => write!(f, "secret not found"),
            KeySyncError::KeychainError(detail) => write!(f, "keychain error: {}", detail),
            KeySyncError::UnsupportedPlatform => {
                write!(f, "keychain access not available on this platform")
            }
        }
    }
}

impl std::error::Error for KeySyncError {}

/// Convenience type alias for functions that return a KeySyncError.
pub type Result<T> = std::result::Result<T, KeySyncError>;

impl From<std::io::Error> for KeySyncError {
    fn from(err: std::io::Error) -> Self {
        KeySyncError::KeychainError(err.to_string())
    }
}
