use std::process::Command;

use crate::error::{KeySyncError, Result};
use crate::credential::CredentialEntry;
use crate::service::parse_service_name;

/// Retrieve a secret from the macOS Keychain via the `security` CLI.
pub(crate) fn get_secret(service: &str, account: &str) -> Result<String> {
    let output = Command::new("security")
        .args([
            "find-generic-password",
            "-s", service,
            "-a", account,
            "-w",
        ])
        .output()?;

    // macOS 13+ may write the password to stderr instead of stdout.
    // Check both streams and use whichever is non-empty.
    let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();

    if !stdout.is_empty() {
        return Ok(stdout);
    }
    if !stderr.is_empty() {
        return Ok(stderr);
    }

    if !output.status.success() {
        // security returns exit code 44 when the item is not found
        if output.status.code() == Some(44) {
            return Err(KeySyncError::NotFound);
        }
        return Err(KeySyncError::KeychainError(
            format!("security find-generic-password exited with status {}",
                output.status)
        ));
    }

    Ok(String::new())
}

/// List all keysync secrets by parsing the keychain dump.
pub(crate) fn list_secrets() -> Result<Vec<CredentialEntry>> {
    let output = Command::new("security")
        .arg("dump-keychain")
        .output()?;

    let out = String::from_utf8_lossy(&output.stdout);
    let records: Vec<&str> = out.split("\nkeychain:").collect();

    let mut entries = Vec::new();
    for rec in records {
        let rec = rec.trim();
        if rec.is_empty() || !rec.contains("class: \"genp\"") {
            continue;
        }

        let svc = find_attr_value(rec, "svce");
        if svc.is_empty() || !svc.starts_with("keysync/") {
            continue;
        }

        let acct = find_attr_value(rec, "acct");
        if acct.is_empty() {
            continue;
        }

        let (scope, project) = parse_service_name(svc);
        entries.push(CredentialEntry::new(scope, project, acct));
    }

    Ok(entries)
}

/// Returns true if the error indicates the secret was not found.
pub(crate) fn is_not_found(err: &KeySyncError) -> bool {
    matches!(err, KeySyncError::NotFound)
}

/// Extract an attribute value from a dump-keychain record line.
fn find_attr_value(record: &str, attr_name: &str) -> &str {
    let needle = format!("\"{}\"", attr_name);
    let idx = match record.find(&needle) {
        Some(i) => i,
        None => return "",
    };

    let after = &record[idx + needle.len()..];
    let eq_idx = match after.find('=') {
        Some(i) => i,
        None => return "",
    };

    let val = after[eq_idx + 1..].trim();
    if val == "<NULL>" {
        return "";
    }

    if val.starts_with('"') {
        let inner = &val[1..];
        if let Some(end) = inner.find('"') {
            return &val[1..end + 1];
        }
    }

    val.trim_matches('"')
}
