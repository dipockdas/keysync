/// Build the keychain service name from scope, optional project, and optional environment.
///
/// Global:       "keysync/global"
/// Project:      "keysync/project/<name>"
/// Environment:  "keysync/project/<name>/env/<env>"
pub(crate) fn service_name(scope: &str, project: &str, environment: &str) -> String {
    if project.is_empty() || scope == "global" {
        format!("keysync/{}", scope)
    } else if !environment.is_empty() {
        format!("keysync/{}/{}/env/{}", scope, project, environment)
    } else {
        format!("keysync/{}/{}", scope, project)
    }
}

/// Parse a service name back into scope, project, and environment parts.
///
/// "keysync/global"                     -> ("global", "", "")
/// "keysync/project/my-app"             -> ("project", "my-app", "")
/// "keysync/project/my-app/env/staging" -> ("project", "my-app", "staging")
pub(crate) fn parse_service_name(svc: &str) -> (&str, &str, &str) {
    let trimmed = svc.strip_prefix("keysync/").unwrap_or(svc);
    if trimmed.is_empty() {
        return ("global", "", "");
    }

    let mut parts = trimmed.splitn(2, '/');
    let scope = parts.next().unwrap_or("global");

    if scope != "project" {
        return (scope, "", "");
    }

    let rest = parts.next().unwrap_or("");

    // Check for /env/ segment to detect environment
    if let Some(env_idx) = rest.find("/env/") {
        if env_idx > 0 {
            let project = &rest[..env_idx];
            let environment = &rest[env_idx + "/env/".len()..];
            return (scope, project, environment);
        }
    }

    (scope, rest, "")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_service_name_global() {
        assert_eq!(service_name("global", "", ""), "keysync/global");
    }

    #[test]
    fn test_service_name_global_with_project() {
        assert_eq!(service_name("global", "my-app", "staging"), "keysync/global");
    }

    #[test]
    fn test_service_name_project() {
        assert_eq!(service_name("project", "my-app", ""), "keysync/project/my-app");
    }

    #[test]
    fn test_service_name_project_with_env() {
        assert_eq!(
            service_name("project", "my-app", "staging"),
            "keysync/project/my-app/env/staging"
        );
    }

    #[test]
    fn test_parse_service_name_global() {
        let (scope, project, env) = parse_service_name("keysync/global");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_service_name_project() {
        let (scope, project, env) = parse_service_name("keysync/project/my-app");
        assert_eq!(scope, "project");
        assert_eq!(project, "my-app");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_service_name_project_deep() {
        let (scope, project, env) = parse_service_name("keysync/project/my/deep/app");
        assert_eq!(scope, "project");
        assert_eq!(project, "my/deep/app");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_service_name_with_env() {
        let (scope, project, env) =
            parse_service_name("keysync/project/my-app/env/staging");
        assert_eq!(scope, "project");
        assert_eq!(project, "my-app");
        assert_eq!(env, "staging");
    }

    #[test]
    fn test_parse_service_name_unprefixed() {
        let (scope, project, env) = parse_service_name("other/global");
        assert_eq!(scope, "other");
        assert_eq!(project, "");
        assert_eq!(env, "");
    }

    #[test]
    fn test_parse_service_name_empty() {
        let (scope, project, env) = parse_service_name("");
        assert_eq!(scope, "global");
        assert_eq!(project, "");
        assert_eq!(env, "");
    }
}
