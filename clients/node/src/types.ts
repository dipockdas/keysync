/** Service name helpers matching the keysync convention. */

/** Build a keychain service name from scope and project.
 *  Global:  "keysync/global"
 *  Project: "keysync/project/<name>"
 */
export function serviceName(scope: string, project?: string): string {
  if (!project || scope === "global") {
    return `keysync/${scope}`;
  }
  return `keysync/${scope}/${project}`;
}

/** Parse a service name back into scope and project. */
export function parseServiceName(svc: string): { scope: string; project?: string } {
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
  return { scope, project: trimmed.slice(slashIdx + 1) };
}
