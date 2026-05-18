import { describe, it, expect, vi, beforeEach } from "vitest";
import { KeySyncError } from "../index.js";
import { serviceName, parseServiceName } from "../types.js";

describe("serviceName", () => {
  it("returns keysync/global for global scope", () => {
    expect(serviceName("global")).toBe("keysync/global");
  });

  it("returns keysync/project/<name> for project scope", () => {
    expect(serviceName("project", "my-app")).toBe("keysync/project/my-app");
  });

  it("returns keysync/project/<name>/env/<env> for project with environment", () => {
    expect(serviceName("project", "my-app", "dev")).toBe(
      "keysync/project/my-app/env/dev"
    );
  });

  it("returns keysync/project/<name> when environment is empty", () => {
    expect(serviceName("project", "my-app", "")).toBe(
      "keysync/project/my-app"
    );
  });

  it("ignores environment for global scope", () => {
    expect(serviceName("global", "my-app", "dev")).toBe("keysync/global");
  });
});

describe("parseServiceName", () => {
  it("parses global scope", () => {
    expect(parseServiceName("keysync/global")).toEqual({ scope: "global" });
  });

  it("parses project scope without environment", () => {
    expect(parseServiceName("keysync/project/my-app")).toEqual({
      scope: "project",
      project: "my-app",
    });
  });

  it("parses project scope with environment", () => {
    expect(parseServiceName("keysync/project/my-app/env/production")).toEqual({
      scope: "project",
      project: "my-app",
      environment: "production",
    });
  });

  it("parses deep project path with environment", () => {
    expect(
      parseServiceName("keysync/project/my/deep/app/env/staging")
    ).toEqual({
      scope: "project",
      project: "my/deep/app",
      environment: "staging",
    });
  });
});

describe("KeySyncError", () => {
  it("has correct name and code for notFound", () => {
    const err = new KeySyncError("notFound", "secret not found");
    expect(err.name).toBe("KeySyncError");
    expect(err.code).toBe("notFound");
    expect(err.message).toBe("secret not found");
  });

  it("has correct code for keychainError", () => {
    const err = new KeySyncError("keychainError", "something broke");
    expect(err.code).toBe("keychainError");
  });

  it("has correct code for unsupportedPlatform", () => {
    const err = new KeySyncError("unsupportedPlatform", "not supported");
    expect(err.code).toBe("unsupportedPlatform");
  });
});
