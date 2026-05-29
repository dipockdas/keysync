#![allow(non_snake_case)]

use std::ptr;

use windows_sys::Win32::Security::Credentials::{
    CredReadW, CredFree, CredEnumerateW, CREDENTIALW, CRED_TYPE_GENERIC,
    CRED_ENUMERATE_ALL_CREDENTIALS,
};
use windows_sys::Win32::Foundation::{
    ERROR_NOT_FOUND, FALSE,
};

use crate::error::{KeySyncError, Result};
use crate::credential::CredentialEntry;

/// Convert a keysync service name to a Windows Credential Manager target,
/// stripping the /env/ keyword and replacing slashes with underscores.
///
/// "keysync/global"                  -> "keysync_global"
/// "keysync/project/my-app"           -> "keysync_project_my-app"
/// "keysync/project/my-app/env/dev"   -> "keysync_project_my-app_dev"
fn service_to_target(service: &str) -> String {
    // Strip /env/ keyword so environment is just part of the path
    let processed = service.replace("/env/", "/");
    if let Some(stripped) = processed.strip_prefix("keysync/") {
        format!("keysync_{}", stripped.replace('/', "_"))
    } else {
        format!("keysync_{}", processed)
    }
}

/// Parse a Windows target name back into scope, project, and environment.
///
/// "keysync_global"               -> ("global", "", "")
/// "keysync_project_my-app"       -> ("project", "my-app", "")
/// "keysync_project_my-app_dev"   -> ("project", "my-app", "dev")
fn parse_target(target: &str) -> (&str, &str, &str) {
    let trimmed = target.strip_prefix("keysync_").unwrap_or(target);
    let mut parts = trimmed.splitn(2, '_');
    let scope = parts.next().unwrap_or("global");

    if scope != "global" && scope != "project" {
        return ("global", "", "");
    }

    let rest = parts.next().unwrap_or("");

    if scope == "global" {
        return ("global", "", "");
    }

    // For project scope: split into project and env parts
    // Using the last underscore-delimited segment as env (e.g. "my_app_dev" -> project="my_app", env="dev")
    let (project, env) = match rest.rsplitn(2, '_').collect::<Vec<&str>>() {
        parts if parts.len() == 2 => (parts[1], parts[0]),
        _ => (rest, ""),
    };
    ("project", project, env)
}

/// Read a UTF-16 wide string starting at `ptr` up to `len` characters.
unsafe fn read_wide_str(ptr: *const u16, len: usize) -> String {
    if ptr.is_null() || len == 0 {
        return String::new();
    }
    // Find the actual length of the null-terminated string (capped at len)
    let mut actual_len = 0;
    while actual_len < len && *ptr.add(actual_len) != 0 {
        actual_len += 1;
    }
    let slice = std::slice::from_raw_parts(ptr, actual_len);
    String::from_utf16_lossy(slice)
}

/// Retrieve a secret from Windows Credential Manager.
pub(crate) fn get_secret(service: &str, account: &str) -> Result<String> {
    let target = service_to_target(service);

    // Convert target to wide string
    let w_target: Vec<u16> = target.encode_utf16().chain(std::iter::once(0)).collect();

    let mut p_cred: *mut CREDENTIALW = ptr::null_mut();
    let success = unsafe {
        CredReadW(
            w_target.as_ptr(),
            CRED_TYPE_GENERIC,
            0,
            &mut p_cred,
        )
    };

    if success == FALSE {
        let err = unsafe { windows_sys::Win32::Foundation::GetLastError() };
        if p_cred.is_null() {
            return Err(KeySyncError::KeychainError(
                format!("CredReadW failed with error code: {}", err)
            ));
        }
        // CredReadW returned a cred even with FALSE? Still report failure.
        unsafe { CredFree(p_cred as *mut std::ffi::c_void) };
        return Err(KeySyncError::KeychainError(
            format!("CredReadW failed with error code: {}", err)
        ));
    }

    let cred = unsafe { &*p_cred };

    // Verify the UserName matches the requested account
    let username = unsafe { read_wide_str(cred.UserName, 256) };
    if username != account {
        unsafe { CredFree(p_cred as *mut std::ffi::c_void) };
        return Err(KeySyncError::NotFound);
    }

    // Read the password from the CredentialBlob
    let password = if cred.CredentialBlobSize > 0 {
        let blob_ptr = cred.CredentialBlob as *const u16;
        let wchar_len = (cred.CredentialBlobSize as usize) / 2;
        unsafe {
            let slice = std::slice::from_raw_parts(blob_ptr, wchar_len);
            String::from_utf16_lossy(slice).trim_end_matches('\0').to_string()
        }
    } else {
        String::new()
    };

    unsafe { CredFree(p_cred as *mut std::ffi::c_void) };

    Ok(password)
}

/// List all keysync secrets from Windows Credential Manager.
pub(crate) fn list_secrets() -> Result<Vec<CredentialEntry>> {
    let filter: Vec<u16> = "keysync_*"
        .encode_utf16()
        .chain(std::iter::once(0))
        .collect();

    let mut count: u32 = 0;
    let mut p_creds: *mut *mut CREDENTIALW = ptr::null_mut();

    let success = unsafe {
        CredEnumerateW(
            filter.as_ptr(),
            CRED_ENUMERATE_ALL_CREDENTIALS,
            &mut count,
            &mut p_creds,
        )
    };

    if success == FALSE || p_creds.is_null() || count == 0 {
        if !p_creds.is_null() {
            unsafe { CredFree(p_creds as *mut std::ffi::c_void) };
        }
        let err = unsafe { windows_sys::Win32::Foundation::GetLastError() };
        if err == ERROR_NOT_FOUND {
            return Ok(Vec::new());
        }
        return Err(KeySyncError::KeychainError(
            format!("CredEnumerateW failed with error code: {}", err)
        ));
    }

    let mut entries = Vec::new();

    for i in 0..count as usize {
        let cred_ptr = unsafe { *p_creds.add(i) };
        if cred_ptr.is_null() {
            continue;
        }

        let cred = unsafe { &*cred_ptr };

        // Read the target name
        let target = unsafe { read_wide_str(cred.TargetName, 512) };

        if !target.starts_with("keysync_") {
            continue;
        }

        let (scope, project, env) = parse_target(&target);
        let username = unsafe { read_wide_str(cred.UserName, 256) };

        if username.is_empty() {
            continue;
        }

        entries.push(CredentialEntry::new(scope, project, env, &username));
    }

    unsafe { CredFree(p_creds as *mut std::ffi::c_void) };

    Ok(entries)
}

/// Returns true if the error indicates the secret was not found.
pub(crate) fn is_not_found(err: &KeySyncError) -> bool {
    matches!(err, KeySyncError::NotFound)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_service_to_target_global() {
        assert_eq!(service_to_target("keysync/global"), "keysync_global");
    }

    #[test]
    fn test_service_to_target_project() {
        assert_eq!(
            service_to_target("keysync/project/my-app"),
            "keysync_project_my-app"
        );
    }

    #[test]
    fn test_service_to_target_environment() {
        assert_eq!(
            service_to_target("keysync/project/my-app/env/dev"),
            "keysync_project_my-app_dev"
        );
    }

    #[test]
    fn test_service_to_target_project_with_dashes() {
        assert_eq!(
            service_to_target("keysync/project/my-app-v2"),
            "keysync_project_my-app-v2"
        );
    }

    #[test]
    fn test_service_to_target_no_prefix() {
        assert_eq!(service_to_target("global"), "keysync_global");
    }

    #[test]
    fn test_parse_target_global() {
        let (scope, project, env) = parse_target("keysync_global");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_target_project() {
        let (scope, project, env) = parse_target("keysync_project_my-app");
        assert_eq!(scope, "project");
        assert_eq!(project, "my-app");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_target_project_with_env() {
        let (scope, project, env) = parse_target("keysync_project_my-app_dev");
        assert_eq!(scope, "project");
        assert_eq!(project, "my-app");
        assert_eq!(env, "dev");
    }

    #[test]
    fn test_parse_target_project_with_underscore() {
        // Last underscore segment is treated as env (matching Go client behavior)
        let (scope, project, env) = parse_target("keysync_project_my_app_name");
        assert_eq!(scope, "project");
        assert_eq!(project, "my_app");
        assert_eq!(env, "name");
    }

    #[test]
    fn test_parse_target_unknown() {
        let (scope, project, env) = parse_target("something_else");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_target_empty() {
        let (scope, project, env) = parse_target("");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
        assert_eq!(env, "");
    }

    #[test]
    fn test_roundtrip_environment() {
        let original = "keysync/project/my-app/env/staging";
        let target = service_to_target(original);
        let (scope, project, env) = parse_target(&target);
        assert_eq!(scope, "project");
        assert_eq!(project, "my-app");
        assert_eq!(env, "staging");
    }

    #[test]
    fn test_read_wide_str_empty() {
        unsafe {
            assert_eq!(read_wide_str(std::ptr::null(), 0), "");
        }
    }

    #[test]
    fn test_read_wide_str_simple() {
        let s: Vec<u16> = "hello".encode_utf16().chain(std::iter::once(0)).collect();
        unsafe {
            assert_eq!(read_wide_str(s.as_ptr(), s.len()), "hello");
        }
    }

    #[test]
    fn test_read_wide_str_truncates_at_null() {
        let s: Vec<u16> = "ab\0cd".encode_utf16().collect();
        unsafe {
            assert_eq!(read_wide_str(s.as_ptr(), s.len()), "ab");
        }
    }
}
