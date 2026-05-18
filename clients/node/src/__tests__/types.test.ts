import { describe, it, expect } from "vitest";
import { serviceName, parseServiceName } from "../types.js";

describe("serviceName", () => {
  it("returns keysync/global for global scope", () => {
    expect(serviceName("global")).toBe("keysync/global");
  });

  it("returns keysync/project/<name> for project scope", () => {
    expect(serviceName("project", "my-app")).toBe("keysync/project/my-app");
  });

  it("ignores project parameter for global scope", () => {
    expect(serviceName("global", "my-app")).toBe("keysync/global");
  });

  it("returns keysync/project/<name>/env/<env> for project with environment", () => {
    expect(serviceName("project", "my-app", "dev")).toBe("keysync/project/my-app/env/dev");
  });

  it("returns keysync/project/<name> when environment is empty", () => {
    expect(serviceName("project", "my-app", "")).toBe("keysync/project/my-app");
  });

  it("returns keysync/project/<name> when environment is undefined", () => {
    expect(serviceName("project", "my-app", undefined)).toBe("keysync/project/my-app");
  });
});

describe("parseServiceName", () => {
  it("parses global scope", () => {
    expect(parseServiceName("keysync/global")).toEqual({ scope: "global" });
  });

  it("parses project scope", () => {
    expect(parseServiceName("keysync/project/my-app")).toEqual({
      scope: "project",
      project: "my-app",
    });
  });

  it("parses deep project paths", () => {
    expect(parseServiceName("keysync/project/my/deep/app")).toEqual({
      scope: "project",
      project: "my/deep/app",
    });
  });

  it("returns global for unprefixed input", () => {
    expect(parseServiceName("other/global")).toEqual({ scope: "global" });
  });

  it("returns global for empty input", () => {
    expect(parseServiceName("")).toEqual({ scope: "global" });
  });

  it("handles just keysync prefix", () => {
    expect(parseServiceName("keysync")).toEqual({ scope: "global" });
  });

  it("parses project scope with environment", () => {
    expect(parseServiceName("keysync/project/my-app/env/dev")).toEqual({
      scope: "project",
      project: "my-app",
      environment: "dev",
    });
  });

  it("parses project scope without environment", () => {
    expect(parseServiceName("keysync/project/my-app")).toEqual({
      scope: "project",
      project: "my-app",
    });
  });

  it("parses project scope with deep environment path", () => {
    expect(parseServiceName("keysync/project/my/deep/app/env/staging")).toEqual({
      scope: "project",
      project: "my/deep/app",
      environment: "staging",
    });
  });
});
