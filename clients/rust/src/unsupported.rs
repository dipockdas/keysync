use crate::error::{KeySyncError, Result};
use crate::credential::CredentialEntry;
use crate::service::parse_service_name;

/// Unsupported platform stub — returns UnsupportedPlatform for all operations.
pub(crate) fn get_secret(_service: &str, _account: &str) -> Result<String> {
    Err(KeySyncError::UnsupportedPlatform)
}

pub(crate) fn list_secrets() -> Result<Vec<CredentialEntry>> {
    Err(KeySyncError::UnsupportedPlatform)
}

pub(crate) fn is_not_found(_err: &KeySyncError) -> bool {
    false
}
