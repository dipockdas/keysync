/// A credential entry returned by `list_secrets`.
///
/// Each entry identifies a secret by its scope, optional project, optional environment,
/// and key name.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CredentialEntry {
    /// The scope: "global" or "project".
    pub scope: String,
    /// The project name, if the entry is project-scoped.
    pub project: String,
    /// The environment name, if the entry is environment-scoped.
    pub environment: String,
    /// The secret key name (e.g. "DATABASE_URL").
    pub account: String,
}

impl CredentialEntry {
    /// Creates a new `CredentialEntry`.
    pub fn new(scope: &str, project: &str, environment: &str, account: &str) -> Self {
        CredentialEntry {
            scope: scope.to_string(),
            project: project.to_string(),
            environment: environment.to_string(),
            account: account.to_string(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_credential_entry_global() {
        let entry = CredentialEntry::new("global", "", "", "API_KEY");
        assert_eq!(entry.scope, "global");
        assert_eq!(entry.project, "");
        assert_eq!(entry.environment, "");
        assert_eq!(entry.account, "API_KEY");
    }

    #[test]
    fn test_credential_entry_project() {
        let entry = CredentialEntry::new("project", "myapp", "", "DATABASE_URL");
        assert_eq!(entry.scope, "project");
        assert_eq!(entry.project, "myapp");
        assert_eq!(entry.environment, "");
        assert_eq!(entry.account, "DATABASE_URL");
    }

    #[test]
    fn test_credential_entry_environment() {
        let entry = CredentialEntry::new("project", "myapp", "staging", "DB_URL");
        assert_eq!(entry.scope, "project");
        assert_eq!(entry.project, "myapp");
        assert_eq!(entry.environment, "staging");
        assert_eq!(entry.account, "DB_URL");
    }

    #[test]
    fn test_credential_entry_equality() {
        let a = CredentialEntry::new("global", "", "", "KEY");
        let b = CredentialEntry::new("global", "", "", "KEY");
        assert_eq!(a, b);
    }

    #[test]
    fn test_credential_entry_clone() {
        let entry = CredentialEntry::new("project", "myapp", "", "KEY");
        let cloned = entry.clone();
        assert_eq!(entry, cloned);
    }
}
