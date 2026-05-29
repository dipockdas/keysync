/** Service name helpers matching the keysync convention. */

/** Build a keychain service name from scope, project, and environment.
 *  Global:           "keysync/global"
 *  Project:          "keysync/project/<name>"
 *  Project + env:    "keysync/project/<name>/env/<env>"
 */
export function serviceName(scope: string, project?: string, environment?: string): string {
  if (!project || scope === "global") {
    return `keysync/${scope}`;
  }
  if (environment) {
    return `keysync/${scope}/${project}/env/${environment}`;
  }
  return `keysync/${scope}/${project}`;
}

/** Parse a service name back into scope, project, and environment. */
export function parseServiceName(svc: string): { scope: string; project?: string; environment?: string } {
  if (!svc.startsWith("keysync/")) {
    return { scope: "global" };
  }
  const trimmed = svc.slice("keysync/".length);
  const slashIdx = trimmed.indexOf("/");
  if (slashIdx < 0) {
    return { scope: trimmed || "global" };
  }
  const scope = trimmed.slice(0, slashIdx);
  if (scope !== "project") {
    return { scope };
  }
  const rest = trimmed.slice(slashIdx + 1);
  // Check for /env/ pattern: project/myapp/env/dev
  const envIdx = rest.indexOf("/env/");
  if (envIdx >= 0) {
    const project = rest.slice(0, envIdx);
    const environment = rest.slice(envIdx + "/env/".length);
    return { scope, project, environment };
  }
  return { scope, project: rest };
}
