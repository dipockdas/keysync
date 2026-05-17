/// Build the keychain service name.
///
/// Global:  "keysync/global"
/// Project: "keysync/project/<name>"
pub(crate) fn service_name(scope: &str, project: &str) -> String {
    if project.is_empty() || scope == "global" {
        format!("keysync/{}", scope)
    } else {
        format!("keysync/{}/{}", scope, project)
    }
}

/// Parse a service name back into scope and project parts.
///
/// "keysync/global"         -> ("global", "")
/// "keysync/project/my-app" -> ("project", "my-app")
pub(crate) fn parse_service_name(svc: &str) -> (&str, &str) {
    let trimmed = svc.strip_prefix("keysync/").unwrap_or(svc);
    if trimmed.is_empty() {
        return ("global", "");
    }

    let mut parts = trimmed.splitn(2, '/');
    let scope = parts.next().unwrap_or("global");

    if scope != "project" {
        return (scope, "");
    }

    let project = parts.next().unwrap_or("");
    (scope, project)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_service_name_global() {
        assert_eq!(service_name("global", ""), "keysync/global");
    }

    #[test]
    fn test_service_name_global_with_project() {
        // Global scope ignores the project parameter
        assert_eq!(service_name("global", "my-app"), "keysync/global");
    }

    #[test]
    fn test_service_name_project() {
        assert_eq!(service_name("project", "my-app"), "keysync/project/my-app");
    }

    #[test]
    fn test_parse_service_name_global() {
        let (scope, project) = parse_service_name("keysync/global");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
    }

    #[test]
    fn test_parse_service_name_project() {
        let (scope, project) = parse_service_name("keysync/project/my-app");
        assert_eq!(scope, "project");
        assert_eq!(project, "my-app");
    }

    #[test]
    fn test_parse_service_name_project_deep() {
        let (scope, project) = parse_service_name("keysync/project/my/deep/app");
        assert_eq!(scope, "project");
        assert_eq!(project, "my/deep/app");
    }

    #[test]
    fn test_parse_service_name_unprefixed() {
        let (scope, project) = parse_service_name("other/global");
        assert_eq!(scope, "other");
        assert_eq!(project, "");
    }

    #[test]
    fn test_parse_service_name_empty() {
        let (scope, project) = parse_service_name("");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
    }
}
