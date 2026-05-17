use std::process::Command;

use crate::error::{KeySyncError, Result};
use crate::credential::CredentialEntry;
use crate::service::parse_service_name;

/// Retrieve a secret from libsecret via the `secret-tool` CLI.
pub(crate) fn get_secret(service: &str, account: &str) -> Result<String> {
    let output = Command::new("secret-tool")
        .args(["lookup", "service", service, "account", account])
        .output()?;

    if !output.status.success() {
        // secret-tool returns exit code 1 when the item is not found
        if output.status.code() == Some(1) {
            return Err(KeySyncError::NotFound);
        }
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(KeySyncError::KeychainError(
            format!("secret-tool lookup failed: {}", stderr.trim())
        ));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

/// List all keysync secrets by searching libsecret.
pub(crate) fn list_secrets() -> Result<Vec<CredentialEntry>> {
    let output = match Command::new("secret-tool")
        .args(["search", "service", "keysync"])
        .output()
    {
        Ok(o) => o,
        Err(_e) => return Ok(Vec::new()), // secret-tool not available, return empty
    };

    let out = String::from_utf8_lossy(&output.stdout);
    let lines: Vec<&str> = out.lines().collect();

    let mut entries = Vec::new();
    let mut current_svc = String::new();
    let mut current_acct = String::new();

    for line in &lines {
        let line = line.trim();

        if line.is_empty() {
            // End of a record block
            if !current_svc.is_empty() && !current_acct.is_empty() && current_svc.starts_with("keysync/") {
                let (scope, project) = parse_service_name(&current_svc);
                entries.push(CredentialEntry::new(scope, project, &current_acct));
            }
            current_svc.clear();
            current_acct.clear();
            continue;
        }

        if line.starts_with("service") {
            if let Some(val) = extract_attr(line) {
                current_svc = val;
            }
        } else if line.starts_with("account") {
            if let Some(val) = extract_attr(line) {
                current_acct = val;
            }
        }
    }

    // Handle last entry if no trailing blank line
    if !current_svc.is_empty() && !current_acct.is_empty() && current_svc.starts_with("keysync/") {
        let (scope, project) = parse_service_name(&current_svc);
        entries.push(CredentialEntry::new(scope, project, &current_acct));
    }

    Ok(entries)
}

/// Returns true if the error indicates the secret was not found.
pub(crate) fn is_not_found(err: &KeySyncError) -> bool {
    matches!(err, KeySyncError::NotFound)
}

/// Extract the value from an attribute line like "service = keysync/global"
fn extract_attr(line: &str) -> Option<String> {
    line.splitn(2, '=')
        .nth(1)
        .map(|v| v.trim().to_string())
}
